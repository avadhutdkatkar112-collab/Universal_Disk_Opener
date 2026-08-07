package main

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"

	appservices "github.com/user/vhd-opener/internal/app/services"
	"github.com/user/vhd-opener/internal/api"
	"github.com/user/vhd-opener/internal/forensic/search"
	"github.com/user/vhd-opener/internal/jobs"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	appData, err := os.UserConfigDir()
	if err != nil {
		appData = "."
	}
	appDir := filepath.Join(appData, "VHD-Explorer")
	os.MkdirAll(appDir, 0755)

	eventBus := appservices.NewEventBus()
	jobManager := appservices.NewJobManager(eventBus)
	gateway := appservices.NewGateway(eventBus, jobManager)
	workspaceManager := appservices.NewWorkspace(appDir)

	gateway.RegisterCapability(search.NewVFSSearchCapability())
	_ = jobs.HashJobManager{}
	log.Println("[Kernel] Registered: cap.disk.search")

	wailsBridge := api.NewWailsBridge(gateway, eventBus, jobManager, workspaceManager)
	legacyApp := api.NewApp()
	storageHandler := api.NewStorageHandler()
	sessionHandler := api.NewSessionHandler()

	err = wails.Run(&options.App{
		Title:     "Universal Disk Platform",
		Width:     1440,
		Height:    900,
		MinWidth:  900,
		MinHeight: 600,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 243, G: 243, B: 243, A: 1},
		OnStartup: func(ctx context.Context) {
			wailsBridge.Startup(ctx)
			legacyApp.Startup(ctx)
			storageHandler.Startup(ctx)
			sessionHandler.Startup(ctx)
		},
		OnShutdown: func(ctx context.Context) {
			legacyApp.Shutdown(ctx)
		},
		Bind: []interface{}{
			wailsBridge,
			legacyApp,
			storageHandler,
			sessionHandler,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			Theme:                windows.Dark,
			WebviewUserDataPath:  filepath.Join(appDir, "webview"),
		},
	})

	if err != nil {
		log.Fatalf("Error launching platform app: %v", err)
	}
}
