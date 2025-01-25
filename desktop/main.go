package main

import (
	"embed"
	"fmt"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"log"
	"time"
)

//go:embed frontend/out/_next/static/**/*.js frontend/out/_next/static/chunks/pages/*.js frontend/out/_next/static/chunks/app/*.js frontend/out/_next/static/chunks/app/**/*.js frontend/out/_next/static/**/*.woff2 frontend/out/_next/static/**/*.css frontend/out/*.html frontend/out/*.ico frontend/out/*.txt frontend/out/*.json frontend/out/*.png frontend/out/*.js
var assets embed.FS

//go:embed build/logo.png
var icon []byte

func main() {

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:              "Raspberry Pi Alpine Image Builder",
		Width:              1024,
		Height:             768,
		MinWidth:           1024,
		MinHeight:          768,
		MaxWidth:           1280,
		MaxHeight:          800,
		DisableResize:      false,
		Fullscreen:         false,
		Frameless:          false,
		StartHidden:        false,
		HideWindowOnClose:  false,
		BackgroundColour:   &options.RGBA{R: 255, G: 255, B: 255, A: 255},
		Assets:             assets,
		Menu:               nil,
		Logger:             nil,
		LogLevel:           logger.DEBUG,
		LogLevelProduction: logger.ERROR,
		OnStartup:          app.Startup,
		OnDomReady:         app.DomReady,
		OnBeforeClose:      app.BeforeClose,
		OnShutdown:         app.Shutdown,
		WindowStartState:   options.Normal,
		Bind: []interface{}{
			app,
		},
		// Windows platform specific options
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			// DisableFramelessWindowDecorations: false,
			//WebviewUserDataPath: "",
		},
		// Mac platform specific options
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: false,
				HideTitle:                  true,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
				HideToolbarSeparator:       true,
			},
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "Raspberry Pi Alpine Image Builder",
				Message: fmt.Sprintf("© %d Raspberry Pi Alpine Image Builder", time.Now().Year()),
				Icon:    icon,
			},
		},
		Debug: options.Debug{
			OpenInspectorOnStartup: false,
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}
