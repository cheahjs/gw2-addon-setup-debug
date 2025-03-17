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
	window          *app.Window
}

func NewScanner(logger *zap.SugaredLogger, directory string, window *app.Window) *Scanner {
	return &Scanner{
		logger:         logger,
		directory:      directory,
		continueButton: widget.Clickable{},
		window:         window,
	}
}

func (s *Scanner) Run(gtx layout.Context, e app.FrameEvent, scanDll func(string) (*utils.DllInfo, error)) bool {
	th := material.NewTheme()

	// Start scanning if it hasn't started yet
	if !s.scanningStarted {
		s.scanningStarted = true
		go s.scanDlls(scanDll)
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

func (s *Scanner) scanDlls(scanDll func(string) (*utils.DllInfo, error)) {
	s.status = "Looking for DLL files..."
	s.window.Invalidate()
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
		s.window.Invalidate()
		return
	}

	// Scan each DLL
	s.status = fmt.Sprintf("Found %d DLLs, analyzing...", len(dllPaths))
	s.window.Invalidate()

	// Map the DLL paths to a slice
	dllPathsSlice := make([]string, 0, len(dllPaths))
	for path := range dllPaths {
		dllPathsSlice = append(dllPathsSlice, path)
	}

	for i, dllPath := range dllPathsSlice {
		s.status = fmt.Sprintf("Analyzing DLL %d/%d: %s", i+1, len(dllPathsSlice), filepath.Base(dllPath))
		s.window.Invalidate()

		info, err := scanDll(dllPath)
		if err != nil {
			s.logger.Errorw("Failed to scan DLL", "path", dllPath, "error", err)
			s.dllInfos = append(s.dllInfos, &utils.DllInfo{
				Error: err.Error(),
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
	s.window.Invalidate()
}

func (s *Scanner) GetDllInfos() []*utils.DllInfo {
	return s.dllInfos
}
