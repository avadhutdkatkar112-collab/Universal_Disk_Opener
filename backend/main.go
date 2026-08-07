package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/user/vhd-opener/internal/infrastructure/database"
	"github.com/user/vhd-opener/internal/infrastructure/logger"
	"github.com/user/vhd-opener/internal/ui"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Get app data directory
	appData, err := os.UserConfigDir()
	if err != nil {
		appData = "."
	}
	appDir := filepath.Join(appData, "VHD-Opener")
	os.MkdirAll(appDir, 0755)

	// Initialize logger
	logDir := filepath.Join(appDir, "logs")
	if err := logger.Init(logDir); err != nil {
		log.Printf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Log.Info("Starting VHD Opener")

	// Initialize database
	dbPath := filepath.Join(appDir, "recent.db")
	recentService, err := database.NewRecentFilesService(dbPath)
	if err != nil {
		logger.Log.Error("Failed to initialize database", logger.Log.Error("error", err))
		// Continue without database
		recentService = nil
	}

	// Create application handler
	app := ui.NewApp(recentService)

	// Create Wails application
	err = wails.Run(&options.App{
		Title:     "VHD Opener",
		Width:     1280,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			DisableFramelessWindowImplications: false,
			WebviewUserDataPath: filepath.Join(appDir, "webview"),
		},
	})

	if err != nil {
		logger.Log.Fatal("Application error", logger.Log.Error("error", err))
	}
}
