package scan_directory

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/cheahjs/gw2-addon-setup-debug/utils"
	"go.uber.org/zap"
)

type Scanner struct {
	logger          *zap.SugaredLogger
	directory       string
	continueButton  widget.Clickable
	dllInfos        []*utils.DllInfo
	status          string
	scanningStarted bool
	scanningDone    bool
	window          *app.Window // Optional: only used in UI mode
}

// NewScanner creates a new Scanner instance for UI mode.
func NewScanner(logger *zap.SugaredLogger, directory string, window *app.Window) *Scanner {
	return &Scanner{
		logger:         logger,
		directory:      directory,
		continueButton: widget.Clickable{},
		window:         window,
	}
}

// NewScannerNonInteractive creates a new Scanner instance for non-interactive mode.
func NewScannerNonInteractive(logger *zap.SugaredLogger, directory string) *Scanner {
	return &Scanner{
		logger:    logger,
		directory: directory,
	}
}

func (s *Scanner) Run(gtx layout.Context, e app.FrameEvent, scanDllFunc func(string) (*utils.DllInfo, error)) bool {
	th := material.NewTheme()

	// Start scanning if it hasn't started yet
	if !s.scanningStarted {
		s.scanningStarted = true
		go func() {
			_, err := s.ScanDпииNonInteractive(scanDllFunc) // Use a more descriptive name
			if err != nil {
				s.logger.Errorw("Error scanning DLLs", "error", err)
				s.status = "Error scanning DLLs: " + err.Error()
			}
			s.scanningDone = true
			if s.window != nil {
				s.window.Invalidate()
			}
		}()
	}

	// Continue button clicked and scanning is done
	if s.continueButton.Clicked(gtx) && s.scanningDone {
		return true
	}

	layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			paragraph := material.Body1(th, "Scanning DLLs in Guild Wars 2 directory")
			paragraph.Alignment = text.Middle
			return paragraph.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			paragraph := material.Body1(th, s.status)
			return paragraph.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if s.scanningDone {
				paragraph := material.Body1(th, fmt.Sprintf("Found %d DLLs", len(s.dllInfos)))
				return paragraph.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if s.scanningDone {
				btn := material.Button(th, &s.continueButton, "Continue")
				return btn.Layout(gtx)
			}
			return layout.Dimensions{}
		}))

	return false
}

// ScanDпииNonInteractive performs DLL scanning without UI interaction.
// It updates the status for UI mode if a window is available.
func (s *Scanner) ScanDпииNonInteractive(scanDllFunc func(string) (*utils.DllInfo, error)) ([]*utils.DllInfo, error) {
	s.status = "Looking for DLL files..."
	if s.window != nil {
		s.window.Invalidate()
	}
	dllPaths := make(map[string]struct{})

	// Find all DLLs in the directory
	err := filepath.Walk(s.directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			s.logger.Errorw("Failed to walk directory", "error", err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) == ".dll" {
			dllPaths[path] = struct{}{}
		}
		return nil
	})

	if err != nil {
		s.logger.Errorw("Failed to walk directory", "error", err)
		s.status = "Error scanning directory: " + err.Error()
		s.scanningDone = true
		if s.window != nil {
			s.window.Invalidate()
		}
		return nil, err
	}

	// Scan each DLL
	s.status = fmt.Sprintf("Found %d DLLs, analyzing...", len(dllPaths))
	if s.window != nil {
		s.window.Invalidate()
	}

	// Map the DLL paths to a slice
	dllPathsSlice := make([]string, 0, len(dllPaths))
	for path := range dllPaths {
		dllPathsSlice = append(dllPathsSlice, path)
	}

	for i, dllPath := range dllPathsSlice {
		s.status = fmt.Sprintf("Analyzing DLL %d/%d: %s", i+1, len(dllPathsSlice), filepath.Base(dllPath))
		if s.window != nil {
			s.window.Invalidate()
		}

		info, err := scanDllFunc(dllPath)
		if err != nil {
			s.logger.Errorw("Failed to scan DLL", "path", dllPath, "error", err)
			s.dllInfos = append(s.dllInfos, &utils.DllInfo{
				Error:    err.Error(),
				FilePath: dllPath,
			})
			continue
		}

		s.dllInfos = append(s.dllInfos, info)
		s.logger.Infow("Scanned DLL",
			"path", dllPath,
			"info", info)
		// Force garbage collection to prevent memory leak
		runtime.GC()
	}

	s.status = fmt.Sprintf("Completed! Analyzed %d DLLs", len(s.dllInfos))
	s.scanningDone = true
	if s.window != nil {
		s.window.Invalidate()
	}
	return s.dllInfos, nil
}

func (s *Scanner) GetDllInfos() []*utils.DllInfo {
	return s.dllInfos
}
