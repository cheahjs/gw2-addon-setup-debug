package main

import (
	"os"
	"path"
	"path/filepath"
	"strings"

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
	// Initialize logger
	logConfig := zap.NewDevelopmentConfig()
	logConfig.OutputPaths = []string{"stdout", "gw2-addon-debug.log"}
	rawLogger, err := logConfig.Build()
	if err != nil {
		panic(errors.Wrap(err, "failed to initialize logger"))
	}
	logger := rawLogger.Sugar()
	defer rawLogger.Sync()

	// Get current working directory
	workDir, err := os.Getwd()
	if err != nil {
		logger.Fatalw("Failed to get current working directory",
			"error", err)
	}
	logger.Infow("Starting debug", "cwd", workDir)

	// Check if we are in the game directory
	if err = isGameDir(workDir); err != nil {
		logger.Fatalw("Are you running this from the game directory?",
			"error", err)
	}

	// Print list of files in game directory
	printFiles(logger, workDir)

	// Build list of DLLs
	dllPaths := getDllPaths(logger, workDir)

	// Check what type of DLLs we have
	for _, dllPath := range dllPaths {
		dllInfo, err := parseDll(logger, dllPath)
		if err != nil {
			logger.Errorw("Failed to parse DLL",
				"file", dllPath,
				"error", err)
			continue
		}
		logger.Infow("Parsed DLL",
			"file", dllPath,
			"info", dllInfo,
		)
	}
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
