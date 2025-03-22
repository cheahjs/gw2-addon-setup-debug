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
	"github.com/cheahjs/gw2-addon-setup-debug/ui/registry_check"
	"github.com/cheahjs/gw2-addon-setup-debug/utils"
	"go.uber.org/zap"
)

type Report struct {
	logger            *zap.SugaredLogger
	saveButton        widget.Clickable
	exitButton        widget.Clickable
	gw2Dir            string
	includeDirListing bool
	includeLogs       bool
	dllInfos          []*utils.DllInfo
	processInfo       *utils.ProcessInfo
	registryInfo      *registry_check.RegistryInfo
	reportSaved       bool
	saveLocation      string
	errorMessage      string
	list              *layout.List
}

func NewReport(logger *zap.SugaredLogger, gw2Dir string, dllInfos []*utils.DllInfo, processInfo *utils.ProcessInfo, registryInfo *registry_check.RegistryInfo, includeDirListing bool, includeLogs bool) *Report {
	return &Report{
		logger:            logger,
		saveButton:        widget.Clickable{},
		exitButton:        widget.Clickable{},
		gw2Dir:            gw2Dir,
		includeDirListing: includeDirListing,
		includeLogs:       includeLogs,
		dllInfos:          dllInfos,
		processInfo:       processInfo,
		registryInfo:      registryInfo,
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
				var arcdpsCount, addonLoaderCount, nexusCount, arcdpsAddonCount, addonLoaderAddonCount, nexusAddonCount, gw2loadCount, gw2loadAddonCount int
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
					if dll.IsGw2Load {
						gw2loadCount++
					}
					if dll.IsGw2LoadAddon {
						gw2loadAddonCount++
					}
				}
				summary.WriteString(fmt.Sprintf("  - ArcDPS: %d\n", arcdpsCount))
				summary.WriteString(fmt.Sprintf("  - ArcDPS Addon: %d\n", arcdpsAddonCount))
				summary.WriteString(fmt.Sprintf("  - AddonLoader: %d\n", addonLoaderCount))
				summary.WriteString(fmt.Sprintf("  - AddonLoader Addon: %d\n", addonLoaderAddonCount))
				summary.WriteString(fmt.Sprintf("  - Nexus: %d\n", nexusCount))
				summary.WriteString(fmt.Sprintf("  - Nexus Addon: %d\n", nexusAddonCount))
				summary.WriteString(fmt.Sprintf("  - GW2Load: %d\n", gw2loadCount))
				summary.WriteString(fmt.Sprintf("  - GW2Load Addon: %d\n", gw2loadAddonCount))

				// Add process info
				if r.processInfo != nil {
					summary.WriteString(fmt.Sprintf("- GW2 Process Info:\n"))
					summary.WriteString(fmt.Sprintf("  - Executable: %s\n", r.processInfo.ExecutablePath))
					summary.WriteString(fmt.Sprintf("  - Working Directory: %s\n", r.processInfo.WorkingDir))
					summary.WriteString(fmt.Sprintf("  - Loaded modules: %d\n", len(r.processInfo.LoadedModules)))
					summary.WriteString(fmt.Sprintf("  - Loaded shims:\n"))
					exeDir := filepath.Dir(r.processInfo.ExecutablePath)
					for _, module := range r.processInfo.LoadedModules {
						// Check if the module is a shim loaded at gw2Dir/{d3d11,dxgi,msimg32}.dll and gw2Dir/bin64/cef/dxgi.dll
						if strings.EqualFold(module.ModuleName, filepath.Join(exeDir, "d3d11.dll")) ||
							strings.EqualFold(module.ModuleName, filepath.Join(exeDir, "dxgi.dll")) ||
							strings.EqualFold(module.ModuleName, filepath.Join(exeDir, "msimg32.dll")) ||
							strings.EqualFold(module.ModuleName, filepath.Join(exeDir, "bin64", "cef", "dxgi.dll")) {
							// Find the dll info for the module if it exists
							found := false
							for _, dll := range r.dllInfos {
								if strings.EqualFold(dll.FilePath, module.ModuleName) {
									summary.WriteString(fmt.Sprintf("    - %s (%s)\n", module.ModuleName, strings.TrimSpace(dll.Flags())))
									found = true
									break
								}
							}
							if !found {
								summary.WriteString(fmt.Sprintf("    - %s (unknown shim)\n", module.ModuleName))
							}
						}
					}
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

	addonLoaderInstalled, addonLoaderMessage := r.checkAddonLoaderInstallation()
	if !addonLoaderInstalled {
		flags.WriteString(addonLoaderMessage)
	}

	// Check if dxgi.dll is present, make sure dxgi.dll and bin64/cef/dxgi.dll are both present and are the same file
	// Grab the presence and md5 of both files from r.dllInfos
	dxgiPresent := false
	cefDxgiPresent := false
	dxgiMd5 := ""
	cefDxgiMd5 := ""
	for _, dll := range r.dllInfos {
		if strings.EqualFold(dll.FilePath, filepath.Join(r.gw2Dir, "dxgi.dll")) {
			dxgiPresent = true
			dxgiMd5 = dll.Md5sum
		}
		if strings.EqualFold(dll.FilePath, filepath.Join(r.gw2Dir, "bin64", "cef", "dxgi.dll")) {
			cefDxgiPresent = true
			cefDxgiMd5 = dll.Md5sum
		}
	}
	if dxgiPresent || cefDxgiPresent {
		if dxgiPresent && !cefDxgiPresent {
			flags.WriteString("dxgi.dll is present but bin64/cef/dxgi.dll is not present, they should be the same file\n")
		}
		if !dxgiPresent && cefDxgiPresent {
			flags.WriteString("bin64/cef/dxgi.dll is present but dxgi.dll is not present, they should be the same file\n")
		}
		if dxgiPresent && cefDxgiPresent && dxgiMd5 != cefDxgiMd5 {
			flags.WriteString("dxgi.dll and bin64/cef/dxgi.dll are present but they are different files\n")
		}
	}

	// If Nexus is installed, it's a addon manager so we don't need to check
	// If Arcdps is installed, it's a addon manager so we don't need to check

	return flags.String()
}

func (r *Report) checkAddonLoaderInstallation() (bool, string) {
	// Check if addon loader is installed correctly
	// It is installed correctly if:
	// 1. d3d11.dll, dxgi.dll, bin64/cef/dxgi.dll are present and are addon loader shims
	// 2. addonLoader.dll is present and is an addon loader core
	// 3. addons/lib_imgui/gw2addon_lib_imgui.dll is present and is an addon loader addon
	// 4. addons/d3d9_wrapper/gw2addon_d3d9_wrapper.dll is present and is an addon loader addon
	var d3d11Shim, dxgiShim, cefDxgiShim, libImguiAddon, addonLoaderCore, d3d9WrapperAddon bool
	for _, dll := range r.dllInfos {
		// Check if d3d11.dll is present and is an addon loader shim
		if strings.EqualFold(dll.FilePath, filepath.Join(r.gw2Dir, "d3d11.dll")) && dll.IsAddonLoaderShim {
			d3d11Shim = true
		}
		// Check if dxgi.dll is present and is an addon loader shim
		if strings.EqualFold(dll.FilePath, filepath.Join(r.gw2Dir, "dxgi.dll")) && dll.IsAddonLoaderShim {
			dxgiShim = true
		}
		// Check if bin64/cef/dxgi.dll is present and is an addon loader shim
		if strings.EqualFold(dll.FilePath, filepath.Join(r.gw2Dir, "bin64", "cef", "dxgi.dll")) && dll.IsAddonLoaderShim {
			cefDxgiShim = true
		}
		// Check if addons/lib_imgui/gw2addon_lib_imgui.dll is present and is an addon loader addon
		if strings.EqualFold(dll.FilePath, filepath.Join(r.gw2Dir, "addons", "lib_imgui", "gw2addon_lib_imgui.dll")) && dll.IsAddonLoaderAddon {
			libImguiAddon = true
		}
		// Check if addons/d3d9_wrapper/gw2addon_d3d9_wrapper.dll is present and is an addon loader addon
		if strings.EqualFold(dll.FilePath, filepath.Join(r.gw2Dir, "addons", "d3d9_wrapper", "gw2addon_d3d9_wrapper.dll")) && dll.IsAddonLoaderAddon {
			d3d9WrapperAddon = true
		}
		// Check if addonLoader.dll is present and is an addon loader core
		if strings.EqualFold(dll.FilePath, filepath.Join(r.gw2Dir, "addonLoader.dll")) && dll.IsAddonLoaderCore {
			addonLoaderCore = true
		}
	}

	if d3d11Shim && dxgiShim && cefDxgiShim && libImguiAddon && addonLoaderCore && d3d9WrapperAddon {
		return true, ""
	}
	// none of addon loader shims are present, so it is not installed
	if !d3d11Shim && !dxgiShim && !cefDxgiShim {
		return true, ""
	}
	var builder strings.Builder
	builder.WriteString("Addon Loader is not installed correctly - please reinstall with GW2 Addon Manager\n")
	if !d3d11Shim {
		builder.WriteString("  - d3d11.dll is missing or is not the addon loader\n")
	}
	if !dxgiShim {
		builder.WriteString("  - dxgi.dll is missing or is not the addon loader\n")
	}
	if !cefDxgiShim {
		builder.WriteString("  - bin64/cef/dxgi.dll is missing or is not the addon loader\n")
	}
	if !addonLoaderCore {
		builder.WriteString("  - addonLoader.dll is missing or is not the addon loader\n")
	}
	if !libImguiAddon {
		builder.WriteString("  - addons/lib_imgui/gw2addon_lib_imgui.dll is missing\n")
	}
	if !d3d9WrapperAddon {
		builder.WriteString("  - addons/d3d9_wrapper/gw2addon_d3d9_wrapper.dll is missing\n")
	}
	return false, builder.String()
}

func (r *Report) printLoadChain(report *strings.Builder, loadOrder []utils.LoadOrder) {
	// Create a map of DLLs to their children
	children := make(map[string][]*utils.LoadOrder)
	var roots []*utils.LoadOrder
	orderPtrs := make([]*utils.LoadOrder, len(loadOrder))

	// Create stable pointers to all entries
	for i := range loadOrder {
		orderPtrs[i] = &loadOrder[i]
	}

	// First pass: build the tree structure using filepath as key
	for _, entry := range orderPtrs {
		if entry.Parent == nil {
			roots = append(roots, entry)
		} else {
			parentPath := entry.Parent.DllInfo.FilePath
			children[parentPath] = append(children[parentPath], entry)
		}
	}

	// Helper function to print the tree
	var printTree func(entry *utils.LoadOrder, indent string)
	printTree = func(entry *utils.LoadOrder, indent string) {
		// Print this entry
		report.WriteString(fmt.Sprintf("%s- %s", indent, entry.DllInfo.FilePath))
		if entry.Source != "" {
			report.WriteString(fmt.Sprintf(" (%s)", entry.Source))
		}
		report.WriteString("\n")

		// Print all children
		if kids, ok := children[entry.DllInfo.FilePath]; ok {
			for _, child := range kids {
				printTree(child, indent+"  ")
			}
		}
	}

	report.WriteString("\n=== Inferred DLL Load Chain ===\n")
	for _, root := range roots {
		printTree(root, "")
	}
	report.WriteString("\n")
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

	// Add Registry info
	report.WriteString("=== Windows Registry Settings ===\n")
	if r.registryInfo != nil {
		if r.registryInfo.SafeDllSearchMode != nil {
			report.WriteString(fmt.Sprintf("SafeDllSearchMode: %d\n", *r.registryInfo.SafeDllSearchMode))
		} else {
			report.WriteString("SafeDllSearchMode: Not found\n")
		}

		if r.registryInfo.Gw2RegistryPath != nil {
			report.WriteString(fmt.Sprintf("\nGW2 Registry Installation Path: %s\n", *r.registryInfo.Gw2RegistryPath))
		} else {
			report.WriteString("\nGW2 Registry Installation Path: Not found\n")
		}

		report.WriteString("\nKnownDLLs:\n")
		if len(r.registryInfo.KnownDlls) > 0 {
			var knownDlls []string
			for name, value := range r.registryInfo.KnownDlls {
				knownDlls = append(knownDlls, fmt.Sprintf("%s = %s", name, value))
			}
			sort.Strings(knownDlls)
			for _, dll := range knownDlls {
				report.WriteString(fmt.Sprintf("  %s\n", dll))
			}
		} else {
			report.WriteString("  No KnownDLLs found\n")
		}
		report.WriteString("\n")
	} else {
		report.WriteString("Failed to retrieve registry information\n\n")
	}

	// Add DLL info
	report.WriteString(fmt.Sprintf("=== DLLs in Directory (%d total) ===\n", len(r.dllInfos)))
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
		flags := dll.Flags()
		if flags != "" {
			report.WriteString("  ")
			report.WriteString(flags)
			report.WriteString("\n")
		}
		report.WriteString(fmt.Sprintf("  Path: %s\n", dll.FilePath))
		if dll.Error != "" {
			report.WriteString(fmt.Sprintf("  Error: %s\n", dll.Error))
		} else {
			report.WriteString(fmt.Sprintf("  MD5: %s\n", dll.Md5sum))
			report.WriteString(fmt.Sprintf("  Version: %d.%d.%d.%d\n", dll.FileVersion.Major, dll.FileVersion.Minor, dll.FileVersion.Patch, dll.FileVersion.Build))
			if dll.FileDescription != "" {
				report.WriteString(fmt.Sprintf("  Description: %s\n", dll.FileDescription))
			}
			if dll.ProductName != "" {
				report.WriteString(fmt.Sprintf("  Product: %s\n", dll.ProductName))
			}
			if dll.ProductVersion != "" {
				report.WriteString(fmt.Sprintf("  Product Version: %s\n", dll.ProductVersion))
			}
		}
		report.WriteString("\n")
	}

	// Add load chain info
	loadOrder := utils.ResolveDllLoadResolution(r.dllInfos, r.gw2Dir)
	r.printLoadChain(&report, loadOrder)

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

	// Add this before the directory listing section in generateReport
	if r.includeLogs {
		// Check for arcdps logs
		arcdpsLogPath := filepath.Join(r.gw2Dir, "addons/arcdps/arcdps.log")
		arcdpsCrashLogPath := filepath.Join(r.gw2Dir, "addons/arcdps/arcdps_lastcrash.log")

		report.WriteString("\n=== ArcDPS Logs ===\n\n")

		// Try to read arcdps.log
		if logContent, err := os.ReadFile(arcdpsLogPath); err == nil {
			report.WriteString("=== arcdps.log ===\n")
			report.WriteString(string(logContent))
			report.WriteString("\n\n")
		} else {
			report.WriteString(fmt.Sprintf("Error reading arcdps.log: %s\n\n", err.Error()))
		}

		// Try to read arcdps_lastcrash.log
		if logContent, err := os.ReadFile(arcdpsCrashLogPath); err == nil {
			report.WriteString("=== arcdps_lastcrash.log ===\n")
			report.WriteString(string(logContent))
			report.WriteString("\n\n")
		} else {
			report.WriteString(fmt.Sprintf("Error reading arcdps_lastcrash.log: %s\n\n", err.Error()))
		}
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

	// Open and focus explorer window showing the report
	err = exec.Command("explorer.exe", "/e,/select,", absPath).Run()
	if err != nil {
		r.logger.Errorw("Failed to open report folder", "error", err)
	}
}
