package main

import (
	"os"

	"github.com/cheahjs/gw2-addon-setup-debug/ui"
	"github.com/cheahjs/gw2-addon-setup-debug/utils"

	"gioui.org/app"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func main() {
	// Delete log file if it exists, on a best effort basis
	if _, err := os.Stat("gw2-addon-setup-debug.log"); err == nil {
		if err := os.Remove("gw2-addon-setup-debug.log"); err != nil {
			println("Failed to delete gw2-addon-setup-debug.log, log file will be appended to.")
		}
	}
	// Initialize logger
	logConfig := zap.NewDevelopmentConfig()
	logConfig.OutputPaths = []string{"stdout", "gw2-addon-setup-debug.log"}
	rawLogger, err := logConfig.Build()
	if err != nil {
		panic(errors.Wrap(err, "failed to initialize logger"))
	}
	logger := rawLogger.Sugar()
	defer rawLogger.Sync()

	// Start GUI
	gui := ui.NewUI(logger)

	// Register platform-specific functions
	gui.SetScanDllFunc(func(dllPath string) (*utils.DllInfo, error) {
		return utils.ParseDll(logger, dllPath)
	})

	gui.SetFindProcessFunc(func() (*utils.ProcessInfo, error) {
		return utils.FindGW2Process()
	})

	go func() {
		window := new(app.Window)
		window.Option(app.Title("Guild Wars 2 Addon Setup Debugger"))
		window.Option(app.Size(800, 600))

		err := gui.Run(window)
		if err != nil {
			logger.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}
