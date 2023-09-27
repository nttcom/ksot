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
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/labstack/echo"
	"github.com/nttcom/kuesta/internal/file"
	"github.com/nttcom/kuesta/internal/gogit"
	"github.com/nttcom/kuesta/internal/logger"
	kcue "github.com/nttcom/kuesta/pkg/cue"
	"github.com/nttcom/kuesta/pkg/kuesta"
)

type HttpGetBody struct {
	Paths []string `json:"paths"`
}

type HttpSetBody struct {
	Path  string                 `json:"path"`
	Value map[string]interface{} `json:"value"`
}

type HttpSetRes struct {
	Before map[string]interface{} `json:"before"`
	After  map[string]interface{} `json:"after"`
}

func RunServeHttp(ctx context.Context, cfg *ServeCfg) error {
	l := logger.FromContext(ctx)
	e := echo.New()

	l.Infow("starting to listen", "address", 8080)
	listen, err := net.Listen("tcp", cfg.HttpAddr)
	if err != nil {
		e.Logger.Fatal(err)
	}
	cGit, err := gogit.NewGit(cfg.ConfigGitOptions().ShouldCloneIfNotExist())
	if err != nil {
		e.Logger.Fatal(err)
	}

	e.GET("/capabilities", func(c echo.Context) error {
		return HttpCapabilities(c, cfg.ConfigRootPath)
	})
	e.POST("/get", func(c echo.Context) error {
		return HttpGet(c, cfg.ConfigRootPath)
	})
	e.POST("/set", func(c echo.Context) error {
		return HttpSet(c, ctx, cGit, cfg)
	})

	e.Listener = listen
	e.Logger.Fatal(e.Start(""))
	return nil
}

// Capabilities responds the server capabilities containing the available services.
func HttpCapabilities(c echo.Context, path string) error {
	c.Logger().Info("CapabilityRequest called for http")
	mlist, err := kuesta.ReadServiceMetaAll(path)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("HttpCapabilities error: ReadServiceMetaAll:  %v", err))
	}
	return c.JSON(http.StatusOK, mlist)
}

// Get responds the multiple service inputs requested by GetRequest.
func HttpGet(c echo.Context, rootpath string) error {
	c.Logger().Info("get called for http")
	req := HttpGetBody{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("HttpGet Please provide valid credentials: %v", err))
	}

	res := make([]interface{}, 0)
	for _, v := range req.Paths {
		path := filepath.Clean(filepath.Join(rootpath, v, "input.cue"))
		buf, err := os.ReadFile(path)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("HttpGet error: ReadFile: %v", err))
		}
		cctx := cuecontext.New()
		val, err := kcue.NewValueFromBytes(cctx, buf)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("HttpGet error: NewValueFromBytes: %v", err))
		}
		jsonDump, err := val.MarshalJSON()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("HttpGet error: MarshalJSON: %v", err))
		}
		var mapData map[string]interface{}
		if err := json.Unmarshal(jsonDump, &mapData); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("HttpGet error: Unmarshal: %v", err))
		}
		res = append(res, mapData)
	}

	return c.JSON(http.StatusOK, res)
}

// Set executes specified Replace/Update/Delete operations and responds what is done by SetRequest.
func HttpSet(c echo.Context, ctx context.Context, gogit *gogit.Git, scfg *ServeCfg) error {
	c.Logger().Info("set called for http")
	req := echo.Map{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide valid credentials")
	}

	var reqBody HttpSetBody

	if _, ok := req["path"]; !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "HttpSet error: not found path")
	}
	if _, ok := req["path"].(string); !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "HttpSet error: not string path")
	}
	if _, ok := req["value"]; !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "HttpSet error: not found val")
	}
	if _, ok := req["value"].(map[string]interface{}); !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "HttpSet error: not map val")
	}

	reqBody.Path = req["path"].(string)
	reqBody.Value = req["value"].(map[string]interface{})
	inputPath := filepath.Clean(filepath.Join(scfg.ConfigRootPath, reqBody.Path, "input.cue"))
	buf, err := os.ReadFile(inputPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("HttpSet error: ReadFile:  %v", err))
	}

	cctx := cuecontext.New()
	val, err := kcue.NewValueFromBytes(cctx, buf)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("HttpSet error: NewValueFromBytes:  %v", err))
	}
	jsonDump, err := val.MarshalJSON()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("HttpSet error: MarshalJSON:  %v", err))
	}
	before := make(map[string]interface{}, 0)
	if err := json.Unmarshal(jsonDump, &before); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("HttpSet error: Unmarshal:  %v", err))
	}

	expr := kcue.NewAstExpr(reqBody.Value)
	inputVal := cctx.BuildExpr(expr)
	if inputVal.Err() != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("HttpSet error: BuildExpr:  %v", inputVal.Err()))
	}

	b, err := kcue.FormatCue(inputVal, cue.Final())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("HttpSet error: MarshalJSON:  %v", err))
	}

	if err := file.WriteFileWithMkdir(inputPath, b); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("HttpSet error: WriteFileWithMkdir: %v", err))
	}

	sp := kuesta.ServicePath{RootDir: scfg.ConfigRootPath}
	if err := gogit.Add(sp.ServiceDirPath(kuesta.ExcludeRoot)); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("HttpSet error: gogit add: %v", err))
	}

	serviceApplyCfg := ServiceApplyCfg{RootCfg: scfg.RootCfg}
	if err := RunServiceApply(ctx, &serviceApplyCfg); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("HttpSet error: RunServiceApply: %v", err))
	}

	gitCommitCfg := GitCommitCfg{RootCfg: scfg.RootCfg}
	if err := RunGitCommit(ctx, &gitCommitCfg); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("HttpSet error: RunGitCommit: %v", err))
	}

	res := HttpSetRes{
		Before: before,
		After:  reqBody.Value,
	}
	return c.JSON(http.StatusOK, res)
}
