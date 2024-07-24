package main

import (
	"github.com/labstack/echo"
	"github.com/nttcom/ksot/nb-server/pkg/config"
	"github.com/nttcom/ksot/nb-server/pkg/handler"
)

func main() {
	e := echo.New()
	h := handler.NewHandler(config.Cfg)
	e.GET("/services/:service", h.GetService)
	e.GET("/devices/:device", h.GetDevice)
	e.POST("/services", h.CreateServices)
	e.PUT("/services", h.UpdateServices)
	e.DELETE("/services", h.DeleteServices)
	e.PUT("/sync/devices", h.SyncDevices)
	e.Logger.Fatal(e.Start(":8080"))
}
