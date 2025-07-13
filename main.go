package main

import (
	"flag"
	"os"

	"github.com/cheahjs/gw2-addon-setup-debug/ui"
	"github.com/cheahjs/gw2-addon-setup-debug/utils"

	"gioui.org/app"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	detectProcess    = flag.Bool("detect-process", false, "Find Gw2-64.exe and run through the process detection flow non-interactively")
	gw2Path          = flag.String("gw2-path", "", "Path to the GW2 directory")
	reportOutputPath = flag.String("report-output-path", "", "Path to write the report to")
)

func main() {
	flag.Parse()

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

	if *detectProcess || *gw2Path != "" || *reportOutputPath != "" {
		// Non-interactive mode
		logger.Info("Running in non-interactive mode")

		var currentGw2Path string
		if *detectProcess {
			logger.Info("Detecting GW2 process")
			processInfo, err := utils.FindGW2Process()
			if err != nil {
				logger.Fatalw("Failed to find GW2 process", "error", err)
			}
			logger.Infow("Found GW2 process", "pid", processInfo.ProcessID, "path", processInfo.ExecutablePath)
			currentGw2Path = processInfo.WorkingDir
		} else if *gw2Path != "" {
			logger.Info("Using provided GW2 path")
			currentGw2Path = *gw2Path
		} else {
			logger.Fatal("Either --detect-process or --gw2-path must be set in non-interactive mode")
		}
		logger.Infow("Using GW2 path", "path", currentGw2Path)

		// Create non-interactive instances of components
		scanner := ui.NewScannerNonInteractive(logger, currentGw2Path)
		processMonitor := ui.NewMonitorNonInteractive(logger)
		registryChecker := ui.NewRegistryCheckerNonInteractive(logger)

		// Perform DLL scanning
		logger.Info("Scanning DLLs")
		dllInfos, err := scanner.ScanDпииNonInteractive(func(dllPath string) (*utils.DllInfo, error) {
			return utils.ParseDll(logger, dllPath)
		})
		if err != nil {
			logger.Fatalw("Failed to scan DLLs", "error", err)
		}
		logger.Infow("DLL scanning complete", "count", len(dllInfos))

		// Perform process monitoring (optional, only if --detect-process was not used for path)
		var processInfo *utils.ProcessInfo
		if !*detectProcess { // If we didn't already get process info from --detect-process
			logger.Info("Finding GW2 process for module information")
			// In non-interactive mode, we assume the user wants the current running process if not using --detect-process for path.
			// If GW2 is not running, this will fail, which is acceptable.
			processInfo, err = processMonitor.FindProcessNonInteractive(utils.FindGW2Process)
			if err != nil {
				logger.Warnw("Could not find GW2 process for module information. Report will not include loaded modules.", "error", err)
				// Continue without process info if not found, it's not strictly fatal for report generation.
			} else {
				logger.Infow("GW2 process found for module information", "pid", processInfo.ProcessID)
			}
		} else {
			// If --detect-process was used, we already have the process info (or it failed earlier)
			// We need to re-fetch it to ensure it's the most up-to-date,
			// as the one from initial path detection might be stale if GW2 restarted.
			logger.Info("Re-fetching GW2 process info for module information")
			processInfo, err = processMonitor.FindProcessNonInteractive(utils.FindGW2Process)
			if err != nil {
				logger.Warnw("Could not re-fetch GW2 process info. Report may use stale module data or none.", "error", err)
			} else {
				logger.Infow("GW2 process info re-fetched", "pid", processInfo.ProcessID)
			}
		}

		// Perform registry check
		logger.Info("Checking registry")
		registryInfo := registryChecker.CheckRegistryNonInteractive()
		logger.Info("Registry check complete")

		// Generate report
		logger.Info("Generating report")
		// For non-interactive mode, always include directory listing and logs.
		// These could be made configurable with more flags if needed.
		reportGenerator := ui.NewReportNonInteractive(logger, currentGw2Path, dllInfos, processInfo, registryInfo, true, true)
		reportPath := *reportOutputPath
		if reportPath == "" {
			// If no output path is specified, print to stdout
			reportString, err := reportGenerator.GenerateReportString()
			if err != nil {
				logger.Fatalw("Failed to generate report string", "error", err)
			}
			fmt.Println(reportString)
			logger.Info("Report printed to stdout")
		} else {
			// Save report to file
			savedPath, err := reportGenerator.SaveReportToFile(reportPath)
			if err != nil {
				logger.Fatalw("Failed to save report", "error", err, "path", reportPath)
			}
			logger.Infow("Report saved", "path", savedPath)
		}
		os.Exit(0) // Terminate after non-interactive execution
	} else {
		// Interactive mode
		logger.Info("Running in interactive mode")
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
			window.Option(app.MinSize(800, 600))

			err := gui.Run(window)
			if err != nil {
				logger.Fatal(err)
			}
			os.Exit(0)
		}()
		app.Main()
	}
}
