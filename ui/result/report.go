package result

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"image/color"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/cheahjs/gw2-addon-setup-debug/utils"
	"go.uber.org/zap"
)

type Report struct {
	logger       *zap.SugaredLogger
	saveButton   widget.Clickable
	exitButton   widget.Clickable
	gw2Dir       string
	dllInfos     []*utils.DllInfo
	processInfo  *utils.ProcessInfo
	reportSaved  bool
	saveLocation string
	errorMessage string
}

func NewReport(logger *zap.SugaredLogger, gw2Dir string, dllInfos []*utils.DllInfo, processInfo *utils.ProcessInfo) *Report {
	return &Report{
		logger:      logger,
		saveButton:  widget.Clickable{},
		exitButton:  widget.Clickable{},
		gw2Dir:      gw2Dir,
		dllInfos:    dllInfos,
		processInfo: processInfo,
	}
}

func (r *Report) Run(gtx layout.Context, e app.FrameEvent) bool {
	th := material.NewTheme()

	if r.saveButton.Clicked(gtx) {
		go r.saveReport()
	}

	if r.exitButton.Clicked(gtx) {
		return true
	}

	layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			paragraph := material.Body1(th, "Debug Report Generated")
			paragraph.Alignment = text.Middle
			return paragraph.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			paragraph := material.Body1(th, "Summary of findings:")
			return paragraph.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 10}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			var summary strings.Builder

			// Add GW2 directory info
			summary.WriteString(fmt.Sprintf("- GW2 Directory: %s\n", r.gw2Dir))

			// Add DLL info
			summary.WriteString(fmt.Sprintf("- Found %d DLLs in directory\n", len(r.dllInfos)))
			var arcdpsCount, addonLoaderCount, nexusCount, arcdpsAddonCount, addonLoaderAddonCount, nexusAddonCount int
			for _, dll := range r.dllInfos {
				if dll.IsArcdps {
					arcdpsCount++
				}
				if dll.IsAddonLoaderShim || dll.IsAddonLoaderCore {
					addonLoaderCount++
				}
				if dll.IsNexus {
					nexusCount++
				}
				if dll.IsArcdpsAddon {
					arcdpsAddonCount++
				}
				if dll.IsAddonLoaderAddon {
					addonLoaderAddonCount++
				}
				if dll.IsNexusAddon {
					nexusAddonCount++
				}
			}
			summary.WriteString(fmt.Sprintf("  - ArcDPS: %d\n", arcdpsCount))
			summary.WriteString(fmt.Sprintf("  - ArcDPS Addon: %d\n", arcdpsAddonCount))
			summary.WriteString(fmt.Sprintf("  - AddonLoader: %d\n", addonLoaderCount))
			summary.WriteString(fmt.Sprintf("  - AddonLoader Addon: %d\n", addonLoaderAddonCount))
			summary.WriteString(fmt.Sprintf("  - Nexus: %d\n", nexusCount))
			summary.WriteString(fmt.Sprintf("  - Nexus Addon: %d\n", nexusAddonCount))

			// Add process info
			if r.processInfo != nil {
				summary.WriteString(fmt.Sprintf("- GW2 Process Info:\n"))
				summary.WriteString(fmt.Sprintf("  - Executable: %s\n", filepath.Base(r.processInfo.ExecutablePath)))
				summary.WriteString(fmt.Sprintf("  - Working Directory: %s\n", r.processInfo.WorkingDir))
				summary.WriteString(fmt.Sprintf("  - Loaded modules: %d\n", len(r.processInfo.LoadedModules)))
			} else {
				summary.WriteString("- No GW2 process info captured\n")
			}

			paragraph := material.Body1(th, summary.String())
			return paragraph.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if r.reportSaved {
				paragraph := material.Body1(th, fmt.Sprintf("Report saved to: %s", r.saveLocation))
				return paragraph.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(
			layout.Spacer{Height: 10}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if r.errorMessage != "" {
				paragraph := material.Body1(th, r.errorMessage)
				return paragraph.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th, &r.saveButton, "Save Report")
			if r.reportSaved {
				btn.Background = color.NRGBA{R: 150, G: 150, B: 150, A: 255}
			}
			return btn.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 10}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th, &r.exitButton, "Exit")
			return btn.Layout(gtx)
		}))

	return false
}

func (r *Report) saveReport() {
	// Create filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("gw2-addon-debug-%s.log", timestamp)

	var report strings.Builder
	report.WriteString("=== Guild Wars 2 Addon Debug Report ===\n")
	report.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC1123)))

	// Add GW2 directory info
	report.WriteString(fmt.Sprintf("GW2 Directory: %s\n\n", r.gw2Dir))

	// Add DLL info
	report.WriteString(fmt.Sprintf("=== DLLs in Directory (%d total) ===\n", len(r.dllInfos)))
	for i, dll := range r.dllInfos {
		report.WriteString(fmt.Sprintf("DLL #%d: %s\n", i+1, filepath.Base(dll.FilePath)))
		report.WriteString(fmt.Sprintf("  Path: %s\n", dll.FilePath))
		if dll.Error != "" {
			report.WriteString(fmt.Sprintf("  Error: %s\n", dll.Error))
		} else {
			report.WriteString(fmt.Sprintf("  MD5: %s\n", dll.Md5sum))
			report.WriteString(fmt.Sprintf("  Version: %d.%d.%d.%d\n", dll.FileVersion.Major, dll.FileVersion.Minor, dll.FileVersion.Patch, dll.FileVersion.Build))
			report.WriteString(fmt.Sprintf("  IsArcdps: %v\n", dll.IsArcdps))
			report.WriteString(fmt.Sprintf("  IsArcdpsAddon: %v\n", dll.IsArcdpsAddon))
			report.WriteString(fmt.Sprintf("  IsAddonLoaderShim: %v\n", dll.IsAddonLoaderShim))
			report.WriteString(fmt.Sprintf("  IsAddonLoaderCore: %v\n", dll.IsAddonLoaderCore))
			report.WriteString(fmt.Sprintf("  IsAddonLoaderAddon: %v\n", dll.IsAddonLoaderAddon))
			report.WriteString(fmt.Sprintf("  IsNexus: %v\n", dll.IsNexus))
			report.WriteString(fmt.Sprintf("  IsNexusAddon: %v\n", dll.IsNexusAddon))
			report.WriteString(fmt.Sprintf("  IsD3D11Shim: %v\n", dll.IsD3D11Shim))
			report.WriteString(fmt.Sprintf("  IsDXGIShim: %v\n", dll.IsDXGIShim))
		}
		report.WriteString("\n")
	}

	// Add process info
	if r.processInfo != nil {
		report.WriteString("=== GW2 Process Information ===\n")
		report.WriteString(fmt.Sprintf("Process ID: %d\n", r.processInfo.ProcessID))
		report.WriteString(fmt.Sprintf("Executable Path: %s\n", r.processInfo.ExecutablePath))
		report.WriteString(fmt.Sprintf("Working Directory: %s\n", r.processInfo.CommandLine))
		report.WriteString(fmt.Sprintf("Command Line: %s\n", r.processInfo.CommandLine))
		report.WriteString(fmt.Sprintf("Captured At: %s\n\n", r.processInfo.Timestamp.Format(time.RFC1123)))

		report.WriteString(fmt.Sprintf("=== Loaded Modules (%d total) ===\n", len(r.processInfo.LoadedModules)))
		for i, module := range r.processInfo.LoadedModules {
			report.WriteString(fmt.Sprintf("%d. %s (0x%x) - %d bytes\n", i+1, module.ModuleName, module.BaseAddress, module.ModuleSize))
		}
	} else {
		report.WriteString("=== GW2 Process Information ===\n")
		report.WriteString("No process information captured\n")
	}

	// Write to file
	err := os.WriteFile(filename, []byte(report.String()), 0644)
	if err != nil {
		r.errorMessage = fmt.Sprintf("Error saving report: %s", err.Error())
		r.logger.Errorw("Failed to save report", "error", err)
		return
	}

	// Get absolute path for display
	absPath, err := filepath.Abs(filename)
	if err != nil {
		absPath = filename
	}

	r.saveLocation = absPath
	r.reportSaved = true
	r.logger.Infow("Report saved", "path", absPath)

	// Open the folder containing the report
	err = exec.Command("explorer", "/select,", absPath).Run()
	if err != nil {
		r.logger.Errorw("Failed to open report folder", "error", err)
	}
}
