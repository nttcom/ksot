package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/iancoleman/orderedmap"
	"github.com/labstack/echo"
	"github.com/nttcom/ksot/github-server/pkg/config"
	"github.com/nttcom/ksot/github-server/pkg/model"
)

type handler struct {
	config config.Config
}

func NewHandler(config config.Config) *handler {
	return &handler{config: config}
}

func matchDataWithGithub() error {
	if out, err := exec.Command("git", "-C", config.Cfg.GitRepoPath, "fetch", "origin", "main").CombinedOutput(); err != nil {
		fmt.Println("fetch: ", string(out))
		return fmt.Errorf("matchDataWithGithub: %w: %v", err, string(out))
	}
	if out, err := exec.Command("git", "-C", config.Cfg.GitRepoPath, "reset", "--hard", "origin/main").CombinedOutput(); err != nil {
		fmt.Println("reset: ", string(out))
		return fmt.Errorf("matchDataWithGithub: %w: %v", err, string(out))
	}
	return nil
}

func updateGithubWithLocal(editFunc func() error, commitMessage string) error {
	if err := matchDataWithGithub(); err != nil {
		return fmt.Errorf("updateGithubWithLocal: %w", err)
	}
	if err := editFunc(); err != nil {
		return fmt.Errorf("updateGithubWithLocal: %w", err)
	}
	if out, err := exec.Command("git", "-C", config.Cfg.GitRepoPath, "add", ".").CombinedOutput(); err != nil {
		fmt.Println("add: ", string(out))
		return fmt.Errorf("updateGithubWithLocal: %w: %v", err, string(out))
	}
	if out, err := exec.Command("git", "-C", config.Cfg.GitRepoPath, "commit", "--allow-empty", "-m", commitMessage).CombinedOutput(); err != nil {
		fmt.Println("commit: ", string(out))
		return fmt.Errorf("updateGithubWithLocal: %w: %v", err, string(out))
	}
	if out, err := exec.Command("git", "-C", config.Cfg.GitRepoPath, "push").CombinedOutput(); err != nil {
		fmt.Println("commit: ", string(out))
		return fmt.Errorf("updateGithubWithLocal: %w: %v", err, string(out))
	}
	return nil
}

func (h *handler) GetFileData(c echo.Context) error {
	filePath := c.QueryParam("path")
	if filePath == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "query parameter for file path does not exist:")
	}
	fmt.Println("GetFileData: ", h.config.GitRepoPath+filePath)
	bytes, err := os.ReadFile(h.config.GitRepoPath + filePath)
	if err != nil {
		fmt.Println(err, "check")
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("GetFileData: %v", err))
	}
	fmt.Println("GetFileData: ", string(bytes))
	return c.JSON(http.StatusOK, model.StringData{StringData: string(bytes)})
}

func (h *handler) PostFileData(c echo.Context) error {
	var req model.ReqStringData
	post_option := c.Request().Header.Get("X-POST-OPTION")
	if err := c.Bind(&req); err != nil {
		fmt.Println(err)
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("PostFileData: %v", err))
	}
	jsonByte := []byte(req.StringData)
	omap := orderedmap.New()
	if err := json.Unmarshal(jsonByte, &omap); err != nil {
		fmt.Println(err)
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("PostFileData: %v", err))
	}
	jsonIndentByte, err := json.MarshalIndent(omap, "", "\t")
	if err != nil {
		fmt.Println(err)
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
		return nil
	}

	err = updateGithubWithLocal(editFunc, req.Path)
	if err != nil {
		fmt.Println(err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("PostFileData: %v", err))
	}
	return c.JSON(http.StatusOK, fmt.Sprintf("Post %v", req.Path))
}

func (h *handler) PutFileData(c echo.Context) error {
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
	err = updateGithubWithLocal(editFunc, req.Path)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("PutFileData: %v", err))
	}
	return c.JSON(http.StatusOK, fmt.Sprintf("Put %v", req.Path))
}

func (h *handler) DeleteFileData(c echo.Context) error {
	values := c.QueryParams()
	fmt.Println("DeleteFileData", values)
	editFunc := func() error {
		for _, v := range values["path"] {
			if err := os.RemoveAll(h.config.GitRepoPath + v); err != nil {
				fmt.Println(err)
				return err
			}
		}
		return nil
	}
	if err := updateGithubWithLocal(editFunc, "delete files: "+strings.Join(values["path"], "&")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("DelFileData: %v", err))
	}
	return c.JSON(http.StatusOK, fmt.Sprintf("Delete: %v", values["path"]))
}
