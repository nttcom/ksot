package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iancoleman/orderedmap"
	"github.com/labstack/echo"
	"github.com/nttcom/ksot/github-server/pkg/config"
	"github.com/nttcom/ksot/github-server/pkg/model"
	"github.com/stretchr/testify/assert"
)

var (
	// TestGetFileData.
	getPaths        = []string{"testdata/testJson.json"}
	expectedGetJson = []string{
		"{\"string_data\":\"{\\n    \\\"test\\\": \\\"testjson\\\"\\n}\"}\n",
	}
	wantGetStatus = []int{http.StatusOK}
	wantGetErrors = []error{nil}
	// TestPostFileDatas.
	postDatas = []string{
		`{"path": "path", "string_data": "{\"aa\":\"test\"}"}`,
	}
	expectedPostReses = []string{
		"\"Post path\"\n",
	}
	wantPostStatus = []int{http.StatusOK}
	wantPostErrors = []error{nil}
	// TestPutFileDatas.
	putDatas = []string{
		`{"path": "path", "string_data": "{\"aa\":\"test\"}"}`,
	}
	expectedPutReses = []string{
		"\"Put path\"\n",
	}
	wantPutStatus = []int{http.StatusOK}
	wantPutErrors = []error{nil}
	// TestDeleteFileDatas.
	deletePaths         = []string{"/testdata/testJson.json", "testdata/testJson2.json&path=/testdata/testJson3.json"}
	expectedDeleteBodys = []string{
		"\"Delete: [/testdata/testJson.json]\"\n",
		"\"Delete: [/testdata/testJson2.json /testdata/testJson3.json]\"\n",
	}
	wantDeleteStatus = []int{http.StatusOK, http.StatusOK}
	wantDeleteErrors = []error{nil, nil}
)

type TestHandler struct {
	handler
}

func updateGithubWithLocalForTest(editFunc func() error, commitMessage string) error {
	return nil
}

func (h *TestHandler) PostFileData(c echo.Context) error {
	var req model.ReqStringData
	post_option := c.Request().Header.Get("X-POST-OPTION")
	if err := c.Bind(&req); err != nil {
		fmt.Println(err)
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("PostFileData: %v", err))
	}
	jsonByte := []byte(req.StringData)
	omap := orderedmap.New()
	if err := json.Unmarshal(jsonByte, &omap); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("PostFileData: %v", err))
	}
	jsonIndentByte, err := json.MarshalIndent(omap, "", "\t")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("PostFileData: %v", err))
	}
	postdir := filepath.Dir(req.Path)
	editFunc := func() error {
		switch post_option {
		case "new":
			if _, err := os.Stat(h.config.GitRepoPath + req.Path); err == nil {
				return errors.New("The file already exists")
			}
		case "new_safe":
			if _, err := os.Stat(h.config.GitRepoPath + req.Path); err == nil {
				return c.String(http.StatusOK, fmt.Sprintf("file already exists %v", req.Path))
			}
		case "update":
		default:
			return errors.New("X-POST-OPTION header is not set or invalid value")
		}
		if err := os.MkdirAll(h.config.GitRepoPath+postdir, 0600); err != nil {
			return err
		}
		if err := os.WriteFile(h.config.GitRepoPath+req.Path, jsonIndentByte, 0600); err != nil {
			fmt.Println(err)
			return err
		}
		if _, err := os.Stat(h.config.GitRepoPath + req.Path); err != nil {
			if err := os.WriteFile(h.config.GitRepoPath+req.Path, jsonByte, 0600); err != nil {
				fmt.Println(err)
				return err
			}
			return nil
		}
		return errors.New("The file already exists")
	}

	// for test
	err = updateGithubWithLocalForTest(editFunc, req.Path)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("PostFileData: %v", err))
	}
	return c.JSON(http.StatusOK, fmt.Sprintf("Post %v", req.Path))
}

func (h *TestHandler) PutFileData(c echo.Context) error {
	var req model.ReqStringData
	if err := c.Bind(&req); err != nil {
		fmt.Println(err)
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("PutFileData: %v", err))
	}
	jsonByte := []byte(req.StringData)
	omap := orderedmap.New()
	if err := json.Unmarshal(jsonByte, &omap); err != nil {
		fmt.Println(err)
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("PutFileData: %v", err))
	}
	jsonIndentByte, err := json.MarshalIndent(omap, "", "\t")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("PutFileData: %v", err))
	}
	postdir := filepath.Dir(req.Path)

	editFunc := func() error {
		if _, err := os.Stat(h.config.GitRepoPath + postdir); err != nil {
			return errors.New("no such directory")
		}

		if _, err := os.Stat(h.config.GitRepoPath + req.Path); err == nil {
			if err := os.WriteFile(h.config.GitRepoPath+req.Path, jsonIndentByte, 0600); err != nil {
				fmt.Println(err)
				return err
			}
			return nil
		}
		return errors.New("no such file")
	}
	// for test
	err = updateGithubWithLocalForTest(editFunc, req.Path)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("PutFileData: %v", err))
	}
	return c.JSON(http.StatusOK, fmt.Sprintf("Put %v", req.Path))
}

func (h *TestHandler) DeleteFileData(c echo.Context) error {
	values := c.QueryParams()
	editFunc := func() error {
		for _, v := range values["path"] {
			if err := os.RemoveAll(h.config.GitRepoPath + v); err != nil {
				fmt.Println(err)
				return err
			}
		}
		return nil
	}
	// for test
	if err := updateGithubWithLocalForTest(editFunc, "delete files: "+strings.Join(values["path"], "&")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("DelFileData: %v", err))
	}
	return c.JSON(http.StatusOK, fmt.Sprintf("Delete: %v", values["path"]))
}

var testHandler *TestHandler
var e *echo.Echo

func TestMain(m *testing.M) {
	e = echo.New()
	testHandler = &TestHandler{
		handler: handler{
			config: config.Config{
				GitRepoPath: "./",
			},
		},
	}
	m.Run()
}

func TestGetFileData(t *testing.T) {
	t.Parallel()
	type test struct {
		arg1       string
		wantStatus int
		want       string
		wantErr    error
	}
	testNames := []string{"正常系: POST成功"}
	tests := make(map[string]test)
	for i, name := range testNames {
		tests[name] = test{
			arg1:       getPaths[i],
			wantStatus: wantGetStatus[i],
			want:       expectedGetJson[i],
			wantErr:    wantGetErrors[i],
		}
	}
	fmt.Println("tests:", tests)
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/file?path=%v", tt.arg1), nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			assert.Equal(t, tt.wantErr, testHandler.GetFileData(c))
			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.want, rec.Body.String())
		})
	}
}

func TestPostFileData(t *testing.T) {
	t.Parallel()
	type test struct {
		arg1       string
		wantStatus int
		want       string
		wantErr    error
	}
	testNames := []string{"正常系: POST成功"}
	tests := make(map[string]test)
	for i, name := range testNames {
		tests[name] = test{
			arg1:       postDatas[i],
			wantStatus: wantPostStatus[i],
			want:       expectedPostReses[i],
			wantErr:    wantPostErrors[i],
		}
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/file", strings.NewReader(tt.arg1))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			assert.Equal(t, tt.wantErr, testHandler.PostFileData(c))
			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.want, rec.Body.String())
		})
	}
}

func TestPutFileData(t *testing.T) {
	t.Parallel()
	type test struct {
		arg1       string
		wantStatus int
		want       string
		wantErr    error
	}
	testNames := []string{"正常系: PUT成功"}
	tests := make(map[string]test)
	for i, name := range testNames {
		tests[name] = test{
			arg1:       putDatas[i],
			wantStatus: wantPutStatus[i],
			want:       expectedPutReses[i],
			wantErr:    wantPutErrors[i],
		}
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/file", strings.NewReader(tt.arg1))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			assert.Equal(t, tt.wantErr, testHandler.PutFileData(c))
			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.want, rec.Body.String())
		})
	}
}

func TestDeleteFileData(t *testing.T) {
	t.Parallel()
	type test struct {
		arg1       string
		wantStatus int
		want       string
		wantErr    error
	}
	testNames := []string{"正常系: DELETE成功"}
	tests := make(map[string]test)
	for i, name := range testNames {
		tests[name] = test{
			arg1:       deletePaths[i],
			wantStatus: wantDeleteStatus[i],
			want:       expectedDeleteBodys[i],
			wantErr:    wantDeleteErrors[i],
		}
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/file?path=%v", tt.arg1), nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			assert.Equal(t, tt.wantErr, testHandler.DeleteFileData(c))
			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.want, rec.Body.String())
		})
	}
}
