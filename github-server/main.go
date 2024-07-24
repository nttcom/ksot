package main

import (
	"github.com/labstack/echo"
	"github.com/nttcom/ksot/github-server/pkg/config"
	"github.com/nttcom/ksot/github-server/pkg/handler"
)

func main() {
	e := echo.New()
	h := handler.NewHandler(config.Cfg)
	e.GET("/file", h.GetFileData)
	e.POST("/file", h.PostFileData)
	e.PUT("/file", h.PutFileData)
	e.DELETE("/file", h.DeleteFileData)
	e.Logger.Fatal(e.Start(":8080"))
}
