package scan_directory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"go.uber.org/zap"
)

type DllInfo struct {
	FilePath           string
	Md5sum             string
	IsArcdps           bool
	IsArcdpsAddon      bool
	IsAddonLoaderShim  bool
	IsAddonLoaderCore  bool
	IsAddonLoaderAddon bool
	IsNexus            bool
	IsNexusAddon       bool
	IsD3D11Shim        bool
	IsDXGIShim         bool
	FileVersion        string
}

type Scanner struct {
	logger          *zap.SugaredLogger
	directory       string
	continueButton  widget.Clickable
	dllInfos        []*DllInfo
	status          string
	scanningStarted bool
	scanningDone    bool
}

func NewScanner(logger *zap.SugaredLogger, directory string) *Scanner {
	return &Scanner{
		logger:         logger,
		directory:      directory,
		continueButton: widget.Clickable{},
	}
}

func (s *Scanner) Run(gtx layout.Context, e app.FrameEvent, scanDll func(string) (*DllInfo, error)) bool {
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

	// Pass the drawing operations to the GPU.
	e.Frame(gtx.Ops)

	return false
}

func (s *Scanner) scanDlls(scanDll func(string) (*DllInfo, error)) {
	s.status = "Looking for DLL files..."
	var dllPaths []string

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
			dllPaths = append(dllPaths, path)
		}
		return nil
	})

	if err != nil {
		s.logger.Errorw("Failed to walk directory", "error", err)
		s.status = "Error scanning directory: " + err.Error()
		s.scanningDone = true
		return
	}

	// Scan each DLL
	s.status = fmt.Sprintf("Found %d DLLs, analyzing...", len(dllPaths))
	for i, dllPath := range dllPaths {
		s.status = fmt.Sprintf("Analyzing DLL %d/%d: %s", i+1, len(dllPaths), filepath.Base(dllPath))

		info, err := scanDll(dllPath)
		if err != nil {
			s.logger.Errorw("Failed to scan DLL", "path", dllPath, "error", err)
			continue
		}

		s.dllInfos = append(s.dllInfos, info)
		s.logger.Infow("Scanned DLL",
			"path", dllPath,
			"info", info)
	}

	s.status = fmt.Sprintf("Completed! Analyzed %d DLLs", len(dllPaths))
	s.scanningDone = true
}

func (s *Scanner) GetDllInfos() []*DllInfo {
	return s.dllInfos
}
