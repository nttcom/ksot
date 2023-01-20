/*
 Copyright (c) 2022-2023 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package core

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/nttcom/kuesta/internal/derrors"
	"github.com/nttcom/kuesta/internal/gogit"
	"github.com/nttcom/kuesta/internal/logger"
	"github.com/nttcom/kuesta/internal/util"
	"github.com/nttcom/kuesta/internal/validator"
	"github.com/nttcom/kuesta/pkg/credentials"
	kcue "github.com/nttcom/kuesta/pkg/cue"
	"github.com/nttcom/kuesta/pkg/kuesta"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

type ServeCfg struct {
	RootCfg

	Addr            string `validate:"required"`
	SyncPeriod      int    `validate:"required"`
	PersistGitState bool
	NoTLS           bool
	Insecure        bool
	TLSCrtPath      string
	TLSKeyPath      string
	TLSCACrtPath    string
}

func (c *ServeCfg) TLSServerConfig() *credentials.TLSServerConfig {
	cfg := &credentials.TLSServerConfig{
		TLSConfigBase: credentials.TLSConfigBase{
			NoTLS:     c.NoTLS,
			CrtPath:   c.TLSCrtPath,
			KeyPath:   c.TLSKeyPath,
			CACrtPath: c.TLSCACrtPath,
		},
	}
	if c.Insecure {
		cfg.ClientAuth = tls.VerifyClientCertIfGiven
	} else {
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return cfg
}

// Validate validates exposed fields according to the `validate` tag.
func (c *ServeCfg) Validate() error {
	if !c.NoTLS {
		if c.TLSKeyPath == "" || c.TLSCrtPath == "" {
			return fmt.Errorf("tls-key and tls-crt options must be set to use TLS")
		}
	}
	if c.SyncPeriod < 10 {
		c.SyncPeriod = 10
	}
	return validator.Validate(c)
}

// Mask returns the copy whose sensitive data are masked.
func (c *ServeCfg) Mask() *ServeCfg {
	cc := *c
	cc.RootCfg = *c.RootCfg.Mask()
	return &cc
}

type PathType string

const (
	NodeService              = "service"
	NodeDevice               = "device"
	KeyServiceKind           = "kind"
	KeyDeviceName            = "name"
	PathTypeService PathType = NodeService
	PathTypeDevice  PathType = NodeDevice
)

func RunServe(ctx context.Context, cfg *ServeCfg) error {
	l := logger.FromContext(ctx)
	l.Debugw("serve called", "config", cfg.Mask())

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	credOpts, err := credentials.GRPCServerCredentials(cfg.TLSServerConfig())
	if err != nil {
		return fmt.Errorf("setup credentials: %w", err)
	}
	g := grpc.NewServer(credOpts...)
	s, err := NewNorthboundServer(cfg)
	if err != nil {
		return fmt.Errorf("init gNMI impl server: %w", err)
	}
	if err := s.cGit.Pull(); err != nil {
		return fmt.Errorf("git pull config repo: %w", err)
	}
	if err := s.sGit.Pull(); err != nil {
		return fmt.Errorf("git pull status repo: %w", err)
	}

	pb.RegisterGNMIServer(g, s)
	reflection.Register(g)

	dur := time.Duration(s.cfg.SyncPeriod) * time.Second
	s.RunConfigSyncLoop(ctx, dur)
	s.RunStatusSyncLoop(ctx, dur)

	l.Infow("starting to listen", "address", cfg.Addr)
	listen, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	if err := g.Serve(listen); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

type NorthboundServer struct {
	pb.UnimplementedGNMIServer

	mu   sync.RWMutex // mu is the RW lock to protect the access to config
	smu  sync.Mutex   // smu is the lock to avoid git operation conflicts
	cfg  *ServeCfg
	cGit *gogit.Git
	sGit *gogit.Git
	impl GnmiRequestHandler
}

// NewNorthboundServer creates new NorthboundServer with supplied ServeCfg.
func NewNorthboundServer(cfg *ServeCfg) (*NorthboundServer, error) {
	cGit, err := gogit.NewGit(cfg.ConfigGitOptions().ShouldCloneIfNotExist())
	if err != nil {
		return nil, err
	}
	sGit, err := gogit.NewGit(cfg.StatusGitOptions().ShouldCloneIfNotExist())
	if err != nil {
		return nil, err
	}
	s := &NorthboundServer{
		cfg:  cfg,
		mu:   sync.RWMutex{},
		smu:  sync.Mutex{},
		cGit: cGit,
		sGit: sGit,
		impl: NewNorthboundServerImpl(cfg),
	}
	return s, nil
}

func NewNorthboundServerWithGit(cfg *ServeCfg, cGit, sGit *gogit.Git) *NorthboundServer {
	return &NorthboundServer{
		cfg:  cfg,
		mu:   sync.RWMutex{},
		smu:  sync.Mutex{},
		cGit: cGit,
		sGit: sGit,
		impl: NewNorthboundServerImpl(cfg),
	}
}

func (s *NorthboundServer) RunStatusSyncLoop(ctx context.Context, dur time.Duration) {
	syncStatusFunc := func() {
		if _, err := s.sGit.Checkout(); err != nil {
			logger.ErrorWithStack(ctx, err, "git checkout")
		}
		if err := s.sGit.Pull(); err != nil {
			logger.ErrorWithStack(ctx, err, "git pull")
		}
	}
	util.SetInterval(ctx, syncStatusFunc, dur, "sync from status repo")
}

func (s *NorthboundServer) RunConfigSyncLoop(ctx context.Context, dur time.Duration) {
	syncConfigFunc := func() {
		s.smu.Lock()
		defer s.smu.Unlock()
		if _, err := s.cGit.Checkout(); err != nil {
			logger.ErrorWithStack(ctx, err, "git checkout")
		}
		if err := s.cGit.Pull(); err != nil {
			logger.ErrorWithStack(ctx, err, "git pull")
		}
	}
	util.SetInterval(ctx, syncConfigFunc, dur, "sync from config repo")
}

var supportedEncodings = []pb.Encoding{pb.Encoding_JSON}

// Capabilities responds the server capabilities containing the available services.
func (s *NorthboundServer) Capabilities(ctx context.Context, req *pb.CapabilityRequest) (*pb.CapabilityResponse, error) {
	l := logger.FromContext(ctx)
	l.Info("CapabilityRequest called")

	resp, err := s.impl.Capabilities(ctx, req)
	grpcerr, werr := derrors.ToGRPCError(err)
	if werr != nil {
		logger.ErrorWithStack(ctx, err, "gnmi CapabilityRequest")
		l.Infof("CapabilityRequest failed with request: \n%s", prototext.Format(req))
	}
	if grpcerr != nil {
		return nil, grpcerr
	}

	l.Info("CapabilityRequest completed")
	return resp, nil
}

// Get responds the multiple service inputs requested by GetRequest.
func (s *NorthboundServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	l := logger.FromContext(ctx)
	l.Info("GetRequest called")
	if !s.mu.TryRLock() {
		l.Info("GetRequest locked")
		return nil, status.Error(codes.Unavailable, "GetRequest is locked. Try again later.")
	}
	defer s.mu.RUnlock()

	resp, err := s.get(ctx, req)
	grpcerr, werr := derrors.ToGRPCError(err)
	if werr != nil {
		logger.ErrorWithStack(ctx, err, "gnmi GetRequest")
		l.Infof("GetRequest failed with request: \n%s", prototext.Format(req))
	}
	if grpcerr != nil {
		return nil, grpcerr
	}

	l.Info("GetRequest completed")
	return resp, nil
}

func (s *NorthboundServer) get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	prefix := req.GetPrefix()
	paths := req.GetPath()
	var notifications []*pb.Notification

	// TODO support wildcard
	for _, path := range paths {
		n, err := s.impl.Get(ctx, prefix, path)
		if err != nil {
			return nil, err
		}
		n.Timestamp = s.getCommitTimeOrNow()
		notifications = append(notifications, n)
	}

	return &pb.GetResponse{Notification: notifications}, nil
}

// Set executes specified Replace/Update/Delete operations and responds what is done by SetRequest.
func (s *NorthboundServer) Set(ctx context.Context, req *pb.SetRequest) (*pb.SetResponse, error) {
	l := logger.FromContext(ctx)
	l.Info("SetRequest called")

	if !s.mu.TryLock() {
		l.Info("SetRequest locked")
		return nil, status.Error(codes.Unavailable, "SetRequest is locked. Try again later.")
	}
	s.smu.Lock()
	defer func() {
		s.smu.Unlock()
		s.mu.Unlock()
	}()

	resp, err := s.set(ctx, req)
	grpcerr, werr := derrors.ToGRPCError(err)
	if werr != nil {
		logger.ErrorWithStack(ctx, err, "gnmi SetRequest")
		l.Infof("SetRequest failed with request: \n%s", prototext.Format(req))
	}
	if grpcerr != nil {
		return nil, grpcerr
	}

	l.Info("SetRequest completed")
	return resp, nil
}

func (s *NorthboundServer) set(ctx context.Context, req *pb.SetRequest) (*pb.SetResponse, error) {
	defer func() {
		if !s.cfg.PersistGitState {
			if err := s.cGit.Reset(gogit.ResetOptsHard()); err != nil {
				logger.ErrorWithStack(ctx, err, "deferred git reset")
			}
			if _, err := s.cGit.Checkout(); err != nil {
				logger.ErrorWithStack(ctx, err, "deferred git checkout")
			}
		}
	}()

	if err := s.cGit.Reset(gogit.ResetOptsHard()); err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("git reset hard: %w", err),
			codes.Internal,
			"Failed to perform 'git reset --hard'",
		)
	}
	if _, err := s.cGit.Checkout(); err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("git checkout to %s: %w", s.cfg.GitTrunk, err),
			codes.Internal,
			"Failed to perform 'git checkout' to %s", s.cfg.GitTrunk,
		)
	}
	if err := s.cGit.Pull(); err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("git pull: %w", err),
			codes.Internal,
			"Failed to perform 'git pull'",
		)
	}

	prefix := req.GetPrefix()
	var results []*pb.UpdateResult

	// TODO performance enhancement
	// TODO support wildcard
	for _, path := range req.GetDelete() {
		res, err := s.impl.Delete(ctx, prefix, path)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	for _, upd := range req.GetReplace() {
		res, err := s.impl.Replace(ctx, prefix, upd.GetPath(), upd.GetVal())
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	for _, upd := range req.GetUpdate() {
		res, err := s.impl.Update(ctx, prefix, upd.GetPath(), upd.GetVal())
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}

	sp := kuesta.ServicePath{RootDir: s.cfg.ConfigRootPath}
	if err := s.cGit.Add(sp.ServiceDirPath(kuesta.ExcludeRoot)); err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("git add: %w", err),
			codes.Internal,
			"Failed to perform 'git add'",
		)
	}

	serviceApplyCfg := ServiceApplyCfg{RootCfg: s.cfg.RootCfg}
	if err := RunServiceApply(ctx, &serviceApplyCfg); err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("service apply: %w", err),
			codes.Internal,
			"Failed to apply service template mapping: %v", err,
		)
	}

	gitCommitCfg := GitCommitCfg{RootCfg: s.cfg.RootCfg}
	if err := RunGitCommit(ctx, &gitCommitCfg); err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("git commit"),
			codes.Internal,
			"Failed to create PullRequest",
		)
	}

	return &pb.SetResponse{
		Prefix:    prefix,
		Response:  results,
		Timestamp: s.getCommitTimeOrNow(),
	}, nil
}

func (s *NorthboundServer) getCommitTimeOrNow() int64 {
	commit, err := s.cGit.Head()
	if err != nil {
		return time.Now().UnixNano()
	}
	return commit.Author.When.UnixNano()
}

type GnmiRequestHandler interface {
	Capabilities(ctx context.Context, req *pb.CapabilityRequest) (*pb.CapabilityResponse, error)
	Get(ctx context.Context, prefix, path *pb.Path) (*pb.Notification, error)
	Delete(ctx context.Context, prefix, path *pb.Path) (*pb.UpdateResult, error)
	Update(ctx context.Context, prefix, path *pb.Path, val *pb.TypedValue) (*pb.UpdateResult, error)
	Replace(ctx context.Context, prefix, path *pb.Path, val *pb.TypedValue) (*pb.UpdateResult, error)
}

var _ GnmiRequestHandler = &NorthboundServerImpl{}

type NorthboundServerImpl struct {
	cfg       *ServeCfg
	converter *GnmiPathConverter
}

func NewNorthboundServerImpl(cfg *ServeCfg) *NorthboundServerImpl {
	return &NorthboundServerImpl{
		cfg:       cfg,
		converter: NewGnmiPathConverter(cfg),
	}
}

// Capabilities responds the server capabilities containing the available services.
func (s *NorthboundServerImpl) Capabilities(ctx context.Context, req *pb.CapabilityRequest) (*pb.CapabilityResponse, error) {
	ver, err := GetGNMIServiceVersion()
	if err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("get gnmi service version: %w", err),
			codes.Internal,
			"Failed to get gnmi service version",
		)
	}
	mlist, err := kuesta.ReadServiceMetaAll(s.cfg.ConfigRootPath)
	if err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("get service metadata: %w", err),
			codes.Internal,
			"Failed to get service metadata",
		)
	}

	models := make([]*pb.ModelData, len(mlist))
	for i, m := range mlist {
		models[i] = m.ModelData()
	}

	return &pb.CapabilityResponse{
		SupportedModels:    models,
		SupportedEncodings: supportedEncodings,
		GNMIVersion:        ver,
	}, nil
}

// Get returns the service input stored at the supplied path.
func (s *NorthboundServerImpl) Get(ctx context.Context, prefix, path *pb.Path) (*pb.Notification, error) {
	l := logger.FromContext(ctx)

	req, err := s.converter.Convert(prefix, path)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Path is invalid: %s", err.Error())
	}
	l = l.With("path", req.String())
	l.Info("getting")

	var buf []byte
	switch r := req.(type) {
	case ServicePathReq:
		buf, err = r.Path().ReadServiceInput()
	case DevicePathReq:
		buf, err = r.Path().ReadActualDeviceConfigFile()
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, status.Errorf(codes.NotFound, "Not found: %s", req.String())
		} else {
			return nil, derrors.GRPCErrorf(
				fmt.Errorf("open file: %w", err),
				codes.Internal,
				"Failed to get resource on requested path: %s", req.String(),
			)
		}
	}

	cctx := cuecontext.New()
	val, err := kcue.NewValueFromBytes(cctx, buf)
	if err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("convert to cue.Value: %w", err),
			codes.Internal,
			"Failed to get convert to cue: %s", req.String(),
		)
	}

	// TODO get only nested tree
	jsonDump, err := val.MarshalJSON()
	if err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("encode to json: %w", err),
			codes.Internal,
			"Failed to encode to json: %s", req.String(),
		)
	}

	update := &pb.Update{
		Path: path,
		Val:  &pb.TypedValue{Value: &pb.TypedValue_JsonVal{JsonVal: jsonDump}},
	}
	// TODO use timestamp when updated
	return &pb.Notification{Prefix: prefix, Update: []*pb.Update{update}}, nil
}

// Delete deletes the service input stored at the supplied path.
func (s *NorthboundServerImpl) Delete(ctx context.Context, prefix, path *pb.Path) (*pb.UpdateResult, error) {
	l := logger.FromContext(ctx)

	// TODO delete partial nested data
	req, err := s.converter.Convert(prefix, path)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Path is invalid: %s", err.Error())
	}
	r, ok := req.(ServicePathReq)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "Only service mutation is supported: %s", r.String())
	}
	l = l.With("path", req.String())
	l.Info("deleting")

	sp := r.Path()
	if err = os.Remove(sp.ServiceInputPath(kuesta.IncludeRoot)); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, derrors.GRPCErrorf(
				fmt.Errorf("delete file: %w", err),
				codes.Internal,
				"Failed to delete file: %s", r.String(),
			)
		}
	}
	return &pb.UpdateResult{Path: path, Op: pb.UpdateResult_DELETE}, nil
}

// Replace replaces the service input stored at the supplied path.
func (s *NorthboundServerImpl) Replace(ctx context.Context, prefix, path *pb.Path, val *pb.TypedValue) (*pb.UpdateResult, error) {
	l := logger.FromContext(ctx)

	// TODO replace partial nested data
	req, err := s.converter.Convert(prefix, path)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Path is invalid: %v", err)
	}
	r, ok := req.(ServicePathReq)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "Only service mutation is supported: %s", r.String())
	}
	l = l.With("path", req.String())
	l.Infow("replacing", "input", string(val.GetJsonVal()))

	cctx := cuecontext.New()
	sp := r.Path()

	input := map[string]any{}
	if err := json.Unmarshal(val.GetJsonVal(), &input); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Failed to decode request payload: path=%s: %v", r.String(), err)
	}

	// resolve unique keys
	transformer, err := sp.ReadServiceTransform(cctx)
	if err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("load transform file: %w", err),
			codes.Internal,
			"Failed to load service transform file: path=%s: %v", r.String(), err,
		)
	}
	convertedKeys, err := transformer.ConvertInputType(r.Keys())
	if err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("convert types of path keys: %w", err),
			codes.InvalidArgument,
			"Path keys are invalid: path=%s: %v", r.String(), err,
		)
	}

	expr := kcue.NewAstExpr(util.MergeMap(input, convertedKeys))
	inputVal := cctx.BuildExpr(expr)
	if inputVal.Err() != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("create input cue value: %w", inputVal.Err()),
			codes.Internal,
			"Failed to create input cue value: path=%s: %v", r.String(), inputVal.Err(),
		)
	}

	b, err := kcue.FormatCue(inputVal, cue.Final())
	if err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("format input cue to bytes: %w", err),
			codes.Internal,
			"Failed to format input cue to bytes: path=%s: %v", r.String(), err,
		)
	}
	if err := sp.WriteServiceInputFile(b); err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("write service input: %w", err),
			codes.Internal,
			"Failed to write service input: path=%s: %v", r.String(), err,
		)
	}

	return &pb.UpdateResult{Path: path, Op: pb.UpdateResult_REPLACE}, nil
}

// Update updates the service input stored at the supplied path.
func (s *NorthboundServerImpl) Update(ctx context.Context, prefix, path *pb.Path, val *pb.TypedValue) (*pb.UpdateResult, error) {
	l := logger.FromContext(ctx)

	// TODO update partial nested data
	req, err := s.converter.Convert(prefix, path)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Path is invalid: %s", err.Error())
	}
	r, ok := req.(ServicePathReq)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "Only service mutation is supported: %s", r.String())
	}
	l = l.With("path", req.String())
	l.Infow("updating", "input", string(val.GetJsonVal()))

	cctx := cuecontext.New()
	sp := r.Path()

	// current input
	buf, err := sp.ReadServiceInput()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, status.Errorf(codes.NotFound, "Not found: %s", r.String())
		} else {
			return nil, derrors.GRPCErrorf(
				fmt.Errorf("open file: %w", err),
				codes.Internal,
				"Failed to get resource on requested path: %s", r.String(),
			)
		}
	}
	curInputVal := cctx.CompileBytes(buf)

	curInput := map[string]any{}
	if err := curInputVal.Decode(&curInput); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Failed to decode request payload: path=%s: %v", r.String(), err)
	}

	// new input
	input := map[string]any{}
	if err := json.Unmarshal(val.GetJsonVal(), &input); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Failed to decode request payload: %s", r.String())
	}

	// resolve unique keys
	transformer, err := sp.ReadServiceTransform(cctx)
	if err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("load transform file: %w", err),
			codes.Internal,
			"Failed to load service transform file: path=%s: %v", r.String(), err,
		)
	}
	convertedKeys, err := transformer.ConvertInputType(r.Keys())
	if err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("convert types of path keys: %w", err),
			codes.InvalidArgument,
			"Path keys are invalid: path=%s: %v", r.String(), err,
		)
	}

	expr := kcue.NewAstExpr(util.MergeMap(curInput, input, convertedKeys))
	inputVal := cctx.BuildExpr(expr)
	if inputVal.Err() != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("create input cue value: %w", inputVal.Err()),
			codes.Internal,
			"Failed to create input cue value: path=%s: %v", r.String(), inputVal.Err(),
		)
	}

	b, err := kcue.FormatCue(inputVal, cue.Final())
	if err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("format input cue to bytes: %w", err),
			codes.Internal,
			"Failed to format input cue to bytes: path=%s: %v", r.String(), err,
		)
	}
	if err := sp.WriteServiceInputFile(b); err != nil {
		return nil, derrors.GRPCErrorf(
			fmt.Errorf("write service input: %w", err),
			codes.Internal,
			"Failed to write service input: path=%s: %v", r.String(), err,
		)
	}

	return &pb.UpdateResult{Path: path, Op: pb.UpdateResult_UPDATE}, nil
}

// GetGNMIServiceVersion returns a pointer to the gNMI service version string.
// The method is non-trivial because of the way it is defined in the proto file.
func GetGNMIServiceVersion() (string, error) {
	gzB, _ := (&pb.Update{}).Descriptor() // nolint
	r, err := gzip.NewReader(bytes.NewReader(gzB))
	if err != nil {
		return "", fmt.Errorf("error in initializing gzip reader: %w", err)
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("error in reading gzip data: %w", err)
	}
	desc := &descriptor.FileDescriptorProto{}
	if err := proto.Unmarshal(b, desc); err != nil {
		return "", fmt.Errorf("error in unmarshaling proto: %w", err)
	}
	ver := proto.GetExtension(desc.Options, pb.E_GnmiService)
	return (ver).(string), nil
}
