package main

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cheahjs/gw2-addon-setup-debug/ui"
	"github.com/cheahjs/gw2-addon-setup-debug/ui/process_modules"
	"github.com/cheahjs/gw2-addon-setup-debug/utils"

	"net/http"
	_ "net/http/pprof"

	"gioui.org/app"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func main() {
	// Delete log file if it exists, on a best effort basis
	if _, err := os.Stat("gw2-addon-debug.log"); err == nil {
		if err := os.Remove("gw2-addon-debug.log"); err != nil {
			println("Failed to delete gw2-addon-debug.log, log file will be appended to.")
		}
	}
	go func() {
		println(http.ListenAndServe("localhost:6060", nil))
	}()
	// Initialize logger
	logConfig := zap.NewDevelopmentConfig()
	logConfig.OutputPaths = []string{"stdout", "gw2-addon-debug.log"}
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
		return utils.ScanDll(logger, dllPath)
	})

	gui.SetFindProcessFunc(func() (*process_modules.ProcessInfo, error) {
		return utils.FindGW2Process()
	})

	go func() {
		window := new(app.Window)
		window.Option(app.Title("Guild Wars 2 Addon Setup Debugger"))
		window.Option(app.Size(700, 500))

		err := gui.Run(window)
		if err != nil {
			logger.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func printFiles(logger *zap.SugaredLogger, workDir string) {
	filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Fatalw("Failed to walk directory",
				"error", err)
		}
		if info.IsDir() {
			return nil
		}
		logger.Infow("Found file",
			"path", strings.TrimPrefix(path, workDir+string(os.PathSeparator)),
			"size", info.Size())
		return nil
	})
}

func getDllPaths(logger *zap.SugaredLogger, workDir string) []string {
	var dllPaths []string
	err := filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Fatalw("Failed to walk directory",
				"error", err)
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".dll" {
			dllPaths = append(dllPaths, strings.TrimPrefix(path, workDir+string(os.PathSeparator)))
		}
		return nil
	})
	if err != nil {
		logger.Fatalw("Failed to walk directory",
			"error", err)
	}
	return dllPaths
}

func isGameDir(workDir string) error {
	// Check if Gw2-64.exe exists
	_, err := os.Stat(path.Join(workDir, "Gw2-64.exe"))
	if err != nil {
		return errors.New("Gw2-64.exe not found")
	}
	// Check if Gw2.dat exists
	_, err = os.Stat(path.Join(workDir, "Gw2.dat"))
	if err != nil {
		return errors.New("Gw2.dat not found")
	}
	return nil
}
