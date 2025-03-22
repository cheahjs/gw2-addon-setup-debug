package ui

import (
	"os"

	"gioui.org/app"
	"gioui.org/op"
	"github.com/cheahjs/gw2-addon-setup-debug/ui/admin"
	"github.com/cheahjs/gw2-addon-setup-debug/ui/process_modules"
	"github.com/cheahjs/gw2-addon-setup-debug/ui/registry_check"
	"github.com/cheahjs/gw2-addon-setup-debug/ui/result"
	"github.com/cheahjs/gw2-addon-setup-debug/ui/scan_directory"
	"github.com/cheahjs/gw2-addon-setup-debug/ui/select_directory"
	"github.com/cheahjs/gw2-addon-setup-debug/ui/start"
	"github.com/cheahjs/gw2-addon-setup-debug/utils"
	"go.uber.org/zap"
)

type UI struct {
	Logger *zap.SugaredLogger

	// UI Components
	startMenu       *start.Menu
	adminCheck      *admin.AdminCheck
	directoryPicker *select_directory.Directory
	dllScanner      *scan_directory.Scanner
	processMonitor  *process_modules.Monitor
	registryChecker *registry_check.RegistryChecker
	resultReport    *result.Report

	currentState uiState

	// Data passed between screens
	gw2Directory      string
	includeDirListing bool
	includeLogs       bool
	dllInfos          []*utils.DllInfo
	processInfo       *utils.ProcessInfo
	registryInfo      *registry_check.RegistryInfo

	// Function pointers for platform-specific operations
	scanDllFunc     func(string) (*utils.DllInfo, error)
	findProcessFunc func() (*utils.ProcessInfo, error)
}

func NewUI(logger *zap.SugaredLogger) *UI {
	return &UI{
		Logger:          logger,
		startMenu:       start.NewMenu(),
		adminCheck:      admin.NewAdminCheck(logger),
		directoryPicker: select_directory.NewDirectory(logger),
		currentState:    startMenuState,
	}
}

func (ui *UI) SetScanDllFunc(fn func(string) (*utils.DllInfo, error)) {
	ui.scanDllFunc = fn
}

func (ui *UI) SetFindProcessFunc(fn func() (*utils.ProcessInfo, error)) {
	ui.findProcessFunc = fn
}

func (ui *UI) Run(w *app.Window) error {
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			// This graphics context is used for managing the rendering state.
			gtx := app.NewContext(&ops, e)

			// Process UI state
			switch ui.currentState {
			case startMenuState:
				if ui.startMenu.Run(gtx, e) {
					// Check if we're already running as admin
					isAdmin, err := utils.IsRunningAsAdmin()
					if err != nil {
						ui.Logger.Errorw("Failed to check admin status", "error", err)
						// Continue to admin check screen on error to be safe
						ui.currentState = adminCheckState
					} else if isAdmin {
						// Skip admin check if already running as admin
						ui.currentState = selectDirectoryState
					} else {
						ui.currentState = adminCheckState
					}
				}

			case adminCheckState:
				continueToNext, shouldElevate := ui.adminCheck.Run(gtx, e)
				if continueToNext {
					if shouldElevate {
						// Exit the current process - the elevated process will take over
						os.Exit(0)
					}
					ui.currentState = selectDirectoryState
				}

			case selectDirectoryState:
				continueToNextStep, selectedDir, includeDirListing, includeLogs := ui.directoryPicker.Run(w, gtx, e)
				if continueToNextStep {
					ui.gw2Directory = selectedDir
					ui.includeDirListing = includeDirListing
					ui.includeLogs = includeLogs
					ui.dllScanner = scan_directory.NewScanner(ui.Logger, selectedDir, w)
					ui.currentState = scanDllsState
				}

			case scanDllsState:
				if ui.dllScanner.Run(gtx, e, ui.scanDllFunc) {
					ui.dllInfos = ui.dllScanner.GetDllInfos()
					ui.processMonitor = process_modules.NewMonitor(ui.Logger, w)
					ui.currentState = processMonitorState
				}

			case processMonitorState:
				if ui.processMonitor.Run(gtx, e, ui.findProcessFunc) {
					ui.processInfo = ui.processMonitor.GetProcessInfo()
					ui.registryChecker = registry_check.NewRegistryChecker(ui.Logger, w)
					ui.currentState = registryCheckState
				}

			case registryCheckState:
				if ui.registryChecker.Run(gtx, e) {
					ui.registryInfo = ui.registryChecker.GetRegistryInfo()
					ui.resultReport = result.NewReport(ui.Logger, ui.gw2Directory, ui.dllInfos, ui.processInfo, ui.registryInfo, ui.includeDirListing, ui.includeLogs)
					ui.currentState = resultState
				}

			case resultState:
				if ui.resultReport.Run(gtx, e) {
					// Exit the application
					return nil
				}
			}

			// Render the frame - must be done in the main FrameEvent handler
			e.Frame(gtx.Ops)
		}
	}
}
