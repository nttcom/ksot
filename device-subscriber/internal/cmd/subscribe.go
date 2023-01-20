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

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/nttcom/kuesta/device-subscriber/internal/logger"
	"github.com/nttcom/kuesta/device-subscriber/internal/model"
	"github.com/nttcom/kuesta/pkg/credentials"
	kcue "github.com/nttcom/kuesta/pkg/cue"
	gclient "github.com/openconfig/gnmi/client"
	gnmiclient "github.com/openconfig/gnmi/client/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/pkg/errors"
)

func Run(cfg Config) error {
	ctx := context.Background()
	l := logger.FromContext(ctx)
	l.Infow("start main run", "cfg", cfg.Mask())

	dest, err := gNMIDestination(cfg)
	if err != nil {
		return fmt.Errorf("setup gnmi destination: %w", err)
	}

	c, err := gnmiclient.New(ctx, dest)
	if err != nil {
		return fmt.Errorf("create gNMI client: %w", errors.WithStack(err))
	}

	fn := func() error {
		return Sync(ctx, cfg, c.(*gnmiclient.Client))
	}
	if err := fn(); err != nil {
		logger.ErrorWithStack(ctx, err, "initial sync")
	}
	if err := Subscribe(ctx, c, fn); err != nil {
		return err
	}
	return nil
}

func Subscribe(ctx context.Context, c gclient.Impl, fn func() error) error {
	l := logger.FromContext(ctx)

	query := gclient.Query{
		Type: gclient.Stream,
		NotificationHandler: func(noti gclient.Notification) error {
			if err, ok := noti.(error); ok {
				return fmt.Errorf("error received: %w", err)
			}
			// NOTE Run something if needed
			return nil
		},
	}

	l.Infow("subscribe starting")
	if err := c.Subscribe(ctx, query); err != nil {
		return fmt.Errorf("open subscribe channel: %w", errors.WithStack(err))
	}
	l.Infow("subscribe started")

	defer func() {
		if err := c.Close(); err != nil {
			l.Errorf("close gNMI subscription: %w", err)
		}
	}()

	for {
		recvErr := c.Recv()
		l.Infow("recv hooked")
		if err := fn(); err != nil {
			logger.ErrorWithStack(ctx, err, "handle notification")
		}

		if errors.Is(recvErr, io.EOF) {
			l.Debugw("EOF received")
			return nil
		} else if recvErr != nil {
			return fmt.Errorf("error received on gNMI subscribe channel: %w", recvErr)
		}
	}
}

func Sync(ctx context.Context, cfg Config, client *gnmiclient.Client) error {
	l := logger.FromContext(ctx)
	l.Infow("sync started")

	buf, err := GetEntireConfig(ctx, client)
	if err != nil {
		return fmt.Errorf("get device config: %w", err)
	}

	// unmarshal
	var obj model.Device
	schema, err := model.Schema()
	if err != nil {
		return err
	}
	if err := schema.Unmarshal(buf, &obj); err != nil {
		return fmt.Errorf("decode JSON IETF val of gNMI update: %w", err)
	}

	// convert to CUE
	cctx := cuecontext.New()
	v := cctx.Encode(obj)
	b, err := kcue.FormatCue(v, cue.Final())
	if err != nil {
		return fmt.Errorf("encode cue.Value to bytes: %w", err)
	}

	if err := PostDeviceConfig(ctx, cfg, b); err != nil {
		return err
	}

	l.Infow("sync completed")
	return nil
}

type SaveConfigRequest struct {
	Device string `json:"device"`
	Config string `json:"config"`
}

// PostDeviceConfig sends HTTP POST with supplied device config.
func PostDeviceConfig(ctx context.Context, cfg Config, data []byte) error {
	l := logger.FromContext(ctx)

	u, err := url.Parse(cfg.AggregatorURL)
	if err != nil {
		return fmt.Errorf("url parse error: %w", errors.WithStack(err))
	}
	u.Path = path.Join(u.Path, "commit")
	body := SaveConfigRequest{
		Device: cfg.Device,
		Config: string(data),
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&body); err != nil {
		return fmt.Errorf("json encode error: %w", errors.WithStack(err))
	}

	c, err := httpClient(cfg.TLSClientConfig())
	if err != nil {
		return fmt.Errorf("create http client: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), &buf)
	if err != nil {
		return fmt.Errorf("create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("post: %w", errors.WithStack(err))
	}
	defer resp.Body.Close()

	var bodyBuf []byte
	if _, err := io.ReadFull(resp.Body, bodyBuf); err != nil {
		l.Errorw("reading response body", "error", err)
	}
	if resp.StatusCode != 200 {
		return errors.WithStack(fmt.Errorf("error code=%d: %s", resp.StatusCode, bodyBuf))
	}
	return nil
}

func gNMIDestination(cfg Config) (gclient.Destination, error) {
	dest := gclient.Destination{
		Addrs:   []string{cfg.Addr},
		Target:  "",
		Timeout: 60 * time.Second,
		Credentials: &gclient.Credentials{
			Username: cfg.Username,
			Password: cfg.Password,
		},
	}
	if cfg.NoTLS {
		return dest, nil
	}
	tlsDeviceCfg := cfg.DeviceTLSClientConfig()
	tlsCfg, err := credentials.NewTLSConfig(tlsDeviceCfg.Certificates(false), tlsDeviceCfg.VerifyServer())
	if err != nil {
		return gclient.Destination{}, fmt.Errorf("get tls config: %w", err)
	}
	dest.TLS = tlsCfg
	return dest, nil
}

func httpClient(cfg *credentials.TLSClientConfig) (*http.Client, error) {
	c := &http.Client{}
	if cfg.NoTLS {
		return c, nil
	}
	tlsCfg, err := credentials.NewTLSConfig(cfg.Certificates(false), cfg.VerifyServer())
	if err != nil {
		return nil, fmt.Errorf("new tls config: %w", err)
	}
	c.Transport = &http.Transport{
		TLSClientConfig: tlsCfg,
	}
	return c, nil
}

// ExtractJsonIetfVal extracts the JSON IETF field of the supplied TypedValue.
func ExtractJsonIetfVal(tv *gnmi.TypedValue) ([]byte, error) {
	v, ok := tv.GetValue().(*gnmi.TypedValue_JsonIetfVal)
	if !ok {
		return nil, errors.WithStack(fmt.Errorf("value did not contain IETF JSON"))
	}
	return v.JsonIetfVal, nil
}

// GetEntireConfig requests gNMI GetRequest and returns entire device config as.
func GetEntireConfig(ctx context.Context, client *gnmiclient.Client) ([]byte, error) {
	req := gnmi.GetRequest{
		Path: []*gnmi.Path{
			{}, // TODO consider to specify target and path
		},
		Encoding: gnmi.Encoding_JSON_IETF,
	}

	resp, err := client.Get(ctx, &req)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// resolve gnmi TypedValue
	var tv *gnmi.TypedValue
	for _, v := range resp.GetNotification() {
		for _, u := range v.GetUpdate() {
			tv = u.GetVal()
			break
		}
	}
	if tv == nil {
		return nil, errors.WithStack(fmt.Errorf("no content from gNMI server"))
	}

	return ExtractJsonIetfVal(tv)
}
