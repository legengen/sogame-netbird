//go:build windows && amd64

package main

import (
	"embed"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	appcore "github.com/legengen/sogame-netbird/client/app"
	"github.com/legengen/sogame-netbird/client/internal/observability"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logger := slog.New(observability.NewRedactingHandler(os.Stderr, slog.LevelInfo))
	application := appcore.NewWindowsController(logger)

	err := wails.Run(&options.App{
		Title:            "Sogame",
		Width:            960,
		Height:           680,
		MinWidth:         720,
		MinHeight:        520,
		DisableResize:    false,
		Frameless:        false,
		BackgroundColour: &options.RGBA{R: 246, G: 247, B: 249, A: 1},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        application.Startup,
		OnShutdown:       application.Shutdown,
		Bind:             []interface{}{application},
	})
	if err != nil {
		logger.Error("wails runtime stopped", "error", err)
	}
}
