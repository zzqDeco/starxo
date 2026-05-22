package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "Starxo",
		Width:            1400,
		Height:           900,
		MinWidth:         1000,
		MinHeight:        600,
		WindowStartState: options.Maximised,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: options.NewRGBA(246, 246, 247, 0),
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHiddenInset(),
			Appearance:           mac.DefaultAppearance,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "Starxo",
				Message: "AI Coding Agent Desktop Application",
				Icon:    appIcon,
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			Theme:                windows.SystemDefault,
			BackdropType:         windows.Mica,
			CustomTheme: &windows.ThemeSettings{
				DarkModeTitleBar:           windows.RGB(32, 32, 35),
				DarkModeTitleBarInactive:   windows.RGB(38, 38, 41),
				DarkModeTitleText:          windows.RGB(245, 245, 247),
				DarkModeTitleTextInactive:  windows.RGB(171, 171, 176),
				DarkModeBorder:             windows.RGB(68, 68, 74),
				DarkModeBorderInactive:     windows.RGB(55, 55, 60),
				LightModeTitleBar:          windows.RGB(247, 247, 248),
				LightModeTitleBarInactive:  windows.RGB(241, 241, 242),
				LightModeTitleText:         windows.RGB(31, 31, 35),
				LightModeTitleTextInactive: windows.RGB(106, 106, 112),
				LightModeBorder:            windows.RGB(213, 213, 218),
				LightModeBorderInactive:    windows.RGB(224, 224, 228),
			},
		},
		Linux: &linux.Options{
			Icon:             appIcon,
			ProgramName:      "starxo",
			WebviewGpuPolicy: linux.WebviewGpuPolicyOnDemand,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app.chatService,
			app.sandboxService,
			app.fileService,
			app.settingsService,
			app.sessionService,
			app.containerService,
			app.platformService,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
