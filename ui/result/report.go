package result

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	logger            *zap.SugaredLogger
	saveButton        widget.Clickable
	exitButton        widget.Clickable
	gw2Dir            string
	includeDirListing bool
	dllInfos          []*utils.DllInfo
	processInfo       *utils.ProcessInfo
	reportSaved       bool
	saveLocation      string
	errorMessage      string
	list              *layout.List
}

func NewReport(logger *zap.SugaredLogger, gw2Dir string, dllInfos []*utils.DllInfo, processInfo *utils.ProcessInfo, includeDirListing bool) *Report {
	return &Report{
		logger:            logger,
		saveButton:        widget.Clickable{},
		exitButton:        widget.Clickable{},
		gw2Dir:            gw2Dir,
		includeDirListing: includeDirListing,
		dllInfos:          dllInfos,
		processInfo:       processInfo,
		list:              &layout.List{Axis: layout.Vertical},
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

	r.list.Layout(gtx, 1, func(gtx layout.Context, _ int) layout.Dimensions {
		return layout.Flex{
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
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				paragraph := material.Body1(th, r.getWarningFlags())
				paragraph.Color = color.NRGBA{R: 200, A: 255}
				return paragraph.Layout(gtx)
			}),
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
					summary.WriteString(fmt.Sprintf("  - Executable: %s\n", r.processInfo.ExecutablePath))
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
			}),
		)
	})

	return false
}

func (r *Report) getWarningFlags() string {
	var flags strings.Builder

	// Check if the path the user provided is different from the path to the GW2 executable
	if r.processInfo != nil {
		executablePath := filepath.Dir(r.processInfo.ExecutablePath)
		if r.gw2Dir != executablePath {
			flags.WriteString("GW2 Directory Mismatch - GW2 is running from a different directory than the one you provided\n")
			flags.WriteString(fmt.Sprintf("  User provided: %s\n", r.gw2Dir))
			flags.WriteString(fmt.Sprintf("  Executable path: %s\n", executablePath))
		}
	}

	return flags.String()
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
	// Sort DLLs by IsNexus, IsArcdps, IsAddonLoaderShim, IsD3D11Shim, IsDXGIShim, IsAddonLoaderCore, IsAddonLoaderAddon, IsNexusAddon, IsArcdpsAddon, then alphabetically
	sort.Slice(r.dllInfos, func(i, j int) bool {
		if r.dllInfos[i].IsNexus != r.dllInfos[j].IsNexus {
			return r.dllInfos[i].IsNexus
		}
		if r.dllInfos[i].IsArcdps != r.dllInfos[j].IsArcdps {
			return r.dllInfos[i].IsArcdps
		}
		if r.dllInfos[i].IsAddonLoaderShim != r.dllInfos[j].IsAddonLoaderShim {
			return r.dllInfos[i].IsAddonLoaderShim
		}
		if r.dllInfos[i].IsGw2Load != r.dllInfos[j].IsGw2Load {
			return r.dllInfos[i].IsGw2Load
		}
		if r.dllInfos[i].IsD3D11Shim != r.dllInfos[j].IsD3D11Shim {
			return r.dllInfos[i].IsD3D11Shim
		}
		if r.dllInfos[i].IsDXGIShim != r.dllInfos[j].IsDXGIShim {
			return r.dllInfos[i].IsDXGIShim
		}
		if r.dllInfos[i].IsAddonLoaderCore != r.dllInfos[j].IsAddonLoaderCore {
			return r.dllInfos[i].IsAddonLoaderCore
		}
		if r.dllInfos[i].IsAddonLoaderAddon != r.dllInfos[j].IsAddonLoaderAddon {
			return r.dllInfos[i].IsAddonLoaderAddon
		}
		if r.dllInfos[i].IsNexusAddon != r.dllInfos[j].IsNexusAddon {
			return r.dllInfos[i].IsNexusAddon
		}
		if r.dllInfos[i].IsArcdpsAddon != r.dllInfos[j].IsArcdpsAddon {
			return r.dllInfos[i].IsArcdpsAddon
		}
		if r.dllInfos[i].IsGw2LoadAddon != r.dllInfos[j].IsGw2LoadAddon {
			return r.dllInfos[i].IsGw2LoadAddon
		}
		return r.dllInfos[i].FilePath < r.dllInfos[j].FilePath
	})

	for i, dll := range r.dllInfos {
		report.WriteString(fmt.Sprintf("DLL #%d: %s", i+1, filepath.Base(dll.FilePath)))
		if r.processInfo != nil {
			// Check if the dll is loaded by the GW2 process
			found := false
			for _, module := range r.processInfo.LoadedModules {
				if module.ModuleName == dll.FilePath {
					found = true
					break
				}
			}
			if found {
				report.WriteString(" (Loaded)")
			}
		}
		report.WriteString("\n")
		// Add a header with all the flags
		var flags strings.Builder
		if dll.IsNexus {
			flags.WriteString("[Nexus] ")
		}
		if dll.IsArcdps {
			flags.WriteString("[Arcdps] ")
		}
		if dll.IsAddonLoaderShim {
			flags.WriteString("[AddonLoaderShim] ")
		}
		if dll.IsD3D11Shim {
			flags.WriteString("[D3D11Shim] ")
		}
		if dll.IsDXGIShim {
			flags.WriteString("[DXGIShim] ")
		}
		if dll.IsAddonLoaderCore {
			flags.WriteString("[AddonLoaderCore] ")
		}
		if dll.IsAddonLoaderAddon {
			flags.WriteString("[AddonLoaderAddon] ")
		}
		if dll.IsNexusAddon {
			flags.WriteString("[NexusAddon] ")
		}
		if dll.IsArcdpsAddon {
			flags.WriteString("[ArcdpsAddon] ")
		}
		if dll.IsGw2Load {
			flags.WriteString("[Gw2Load] ")
		}
		if dll.IsGw2LoadAddon {
			flags.WriteString("[Gw2LoadAddon] ")
		}
		if flags.Len() > 0 {
			flags.WriteString("\n")
			report.WriteString("  " + flags.String())
		}
		report.WriteString(fmt.Sprintf("  Path: %s\n", dll.FilePath))
		if dll.Error != "" {
			report.WriteString(fmt.Sprintf("  Error: %s\n", dll.Error))
		} else {
			report.WriteString(fmt.Sprintf("  MD5: %s\n", dll.Md5sum))
			report.WriteString(fmt.Sprintf("  Version: %d.%d.%d.%d\n", dll.FileVersion.Major, dll.FileVersion.Minor, dll.FileVersion.Patch, dll.FileVersion.Build))
		}
		report.WriteString("\n")
	}

	// Add process info
	if r.processInfo != nil {
		report.WriteString("=== GW2 Process Information ===\n")
		report.WriteString(fmt.Sprintf("Process ID: %d\n", r.processInfo.ProcessID))
		report.WriteString(fmt.Sprintf("Executable Path: %s\n", r.processInfo.ExecutablePath))
		report.WriteString(fmt.Sprintf("Working Directory: %s\n", r.processInfo.WorkingDir))
		report.WriteString(fmt.Sprintf("Command Line: %s\n", r.processInfo.CommandLine))
		report.WriteString(fmt.Sprintf("Captured At: %s\n\n", r.processInfo.Timestamp.Format(time.RFC1123)))

		report.WriteString(fmt.Sprintf("=== Loaded Modules (%d total) ===\n", len(r.processInfo.LoadedModules)))
		// Sort modules by module name (case insensitive)
		sort.Slice(r.processInfo.LoadedModules, func(i, j int) bool {
			return strings.ToLower(r.processInfo.LoadedModules[i].ModuleName) < strings.ToLower(r.processInfo.LoadedModules[j].ModuleName)
		})
		for _, module := range r.processInfo.LoadedModules {
			report.WriteString(fmt.Sprintf("%s (0x%x) - %d bytes\n", module.ModuleName, module.BaseAddress, module.ModuleSize))
		}
	} else {
		report.WriteString("=== GW2 Process Information ===\n")
		report.WriteString("No process information captured\n")
	}

	// Add directory listing if opted in
	if r.includeDirListing {
		report.WriteString("\n=== GW2 Directory Listing ===\n\n")

		// Helper function to recursively list directory contents
		var listDirRecursive func(dirPath string, indent string, maxDepth int, currentDepth int)
		listDirRecursive = func(dirPath string, indent string, maxDepth int, currentDepth int) {
			// Skip if we've reached max depth
			if maxDepth > 0 && currentDepth > maxDepth {
				report.WriteString(fmt.Sprintf("%s[max depth reached]\n", indent))
				return
			}

			files, err := filepath.Glob(filepath.Join(dirPath, "*"))
			if err != nil {
				r.logger.Errorw("Failed to get directory listing", "error", err, "path", dirPath)
				report.WriteString(fmt.Sprintf("%sError getting directory listing: %s\n", indent, err.Error()))
				return
			}

			// Sort files alphabetically
			sort.Strings(files)
			for _, file := range files {
				// Get file info
				info, err := os.Stat(file)
				if err != nil {
					report.WriteString(fmt.Sprintf("%s%s (error: %s)\n", indent, filepath.Base(file), err.Error()))
					continue
				}

				// Show file size for files
				if !info.IsDir() {
					report.WriteString(fmt.Sprintf("%s%s (%d bytes, %s)\n", indent, filepath.Base(file), info.Size(), info.ModTime().Format(time.RFC1123)))
				} else {
					report.WriteString(fmt.Sprintf("%s%s/ (directory)\n", indent, filepath.Base(file)))
					// Recursively list subdirectory contents
					listDirRecursive(file, indent+"  ", maxDepth, currentDepth+1)
				}
			}
		}

		// List main directory and subdirectories recursively
		// Use a max depth of 3 to prevent extremely large reports
		listDirRecursive(r.gw2Dir, "", 3, 0)
		report.WriteString("\n")
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
