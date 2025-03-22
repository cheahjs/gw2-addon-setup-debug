package utils

import (
	"path/filepath"
	"strings"
)

// LoadOrder represents the order in which DLLs will be loaded
type LoadOrder struct {
	DllInfo *DllInfo
	Source  string     // Description of where this DLL was found
	Parent  *LoadOrder // The DLL that loaded this DLL
}

// LoadType represents how a DLL was loaded
type LoadType int

const (
	LoadTypeShim             LoadType = iota // Loaded as d3d11.dll, dxgi.dll, or chainloaded as a shim
	LoadTypeArcdpsAddon                      // Loaded as an arcdps addon
	LoadTypeAddonLoaderAddon                 // Loaded as an addon loader addon
	LoadTypeAddonLoaderCore                  // Loaded as addon loader core
	LoadTypeNexusAddon                       // Loaded as a nexus addon
	LoadTypeGw2LoadAddon                     // Loaded as a GW2Load addon
)

func ResolveDllLoadResolution(dllInfos []*DllInfo, gw2Path string) []LoadOrder {
	var loadOrder []LoadOrder
	dllMap := make(map[string]*DllInfo)
	processed := make(map[string]bool) // Track which DLLs we've already processed

	// Create a map of lowercase filepath to DllInfo for case-insensitive lookup
	for _, info := range dllInfos {
		dllMap[strings.ToLower(info.FilePath)] = info
	}

	// First check the primary proxy shims
	primaryShims := []string{
		filepath.Join(gw2Path, "msimg32.dll"),
		filepath.Join(gw2Path, "d3d11.dll"),
		filepath.Join(gw2Path, "dxgi.dll"),
	}

	// Process each shim and its chain of loaded DLLs
	for _, shimPath := range primaryShims {
		if info, exists := dllMap[strings.ToLower(shimPath)]; exists {
			processLoadChain(info, dllMap, gw2Path, &loadOrder, processed, LoadTypeShim, nil)
		}
	}

	return loadOrder
}

// processLoadChain recursively processes a DLL and any DLLs it would load
func processLoadChain(info *DllInfo, dllMap map[string]*DllInfo, gw2Path string, loadOrder *[]LoadOrder, processed map[string]bool, loadType LoadType, parent *LoadOrder) {
	// Skip if we've already processed this DLL
	lowerPath := strings.ToLower(info.FilePath)
	if processed[lowerPath] {
		return
	}
	processed[lowerPath] = true

	// Create this DLL's LoadOrder entry
	currentLoadOrder := &LoadOrder{
		DllInfo: info,
		Source:  getDllSource(loadType),
		Parent:  parent,
	}

	// Add this DLL to the load order
	*loadOrder = append(*loadOrder, *currentLoadOrder)

	dllDir := filepath.Dir(info.FilePath)

	if info.IsArcdps {
		// Only chainload if arcdps was loaded as a shim (either directly or through chainload)
		if loadType == LoadTypeShim {
			arcDllName := strings.TrimSuffix(filepath.Base(info.FilePath), filepath.Ext(info.FilePath))
			chainloadPath := filepath.Join(gw2Path, arcDllName+"_chainload.dll")
			if chainloadInfo, exists := dllMap[strings.ToLower(chainloadPath)]; exists {
				// Only chainload if the target is also a d3d11/dxgi shim
				if chainloadInfo.IsD3D11Shim || chainloadInfo.IsDXGIShim {
					processLoadChain(chainloadInfo, dllMap, gw2Path, loadOrder, processed, LoadTypeShim, currentLoadOrder)
				}
			}
		}

		// Search for arcdps addons in various directories
		for _, searchDir := range []string{dllDir, gw2Path, filepath.Join(gw2Path, "bin64")} {
			searchDirLower := strings.ToLower(searchDir)
			for dllPath, dllInfo := range dllMap {
				if dllInfo.IsArcdpsAddon && strings.HasPrefix(strings.ToLower(dllPath), searchDirLower) {
					processLoadChain(dllInfo, dllMap, gw2Path, loadOrder, processed, LoadTypeArcdpsAddon, currentLoadOrder)
				}
			}
		}
	}

	if info.IsAddonLoaderShim {
		addonLoaderCorePath := filepath.Join(gw2Path, "addonLoader.dll")
		if addonLoaderCoreInfo, exists := dllMap[strings.ToLower(addonLoaderCorePath)]; exists {
			processLoadChain(addonLoaderCoreInfo, dllMap, gw2Path, loadOrder, processed, LoadTypeAddonLoaderCore, currentLoadOrder)
		}
	}

	if info.IsAddonLoaderCore {
		addonsPath := filepath.Join(gw2Path, "addons")
		for dllPath, dllInfo := range dllMap {
			if dllInfo.IsAddonLoaderAddon && strings.HasPrefix(strings.ToLower(dllPath), strings.ToLower(addonsPath)) {
				processLoadChain(dllInfo, dllMap, gw2Path, loadOrder, processed, LoadTypeAddonLoaderAddon, currentLoadOrder)
			}
		}
	}

	if info.IsNexus {
		nexusRoot := dllDir

		chainloadPath := filepath.Join(nexusRoot, "d3d11_chainload.dll")
		if chainloadInfo, exists := dllMap[strings.ToLower(chainloadPath)]; exists {
			processLoadChain(chainloadInfo, dllMap, gw2Path, loadOrder, processed, LoadTypeShim, currentLoadOrder)
		}

		addonsPath := filepath.Join(nexusRoot, "addons")
		for dllPath, dllInfo := range dllMap {
			if dllInfo.IsNexusAddon && strings.HasPrefix(strings.ToLower(dllPath), strings.ToLower(addonsPath)) {
				processLoadChain(dllInfo, dllMap, gw2Path, loadOrder, processed, LoadTypeNexusAddon, currentLoadOrder)
			}
		}

		arcdpsIntegrationPath := filepath.Join(nexusRoot, "addons", "Nexus", "arcdps_integration64.dll")
		if integrationInfo, exists := dllMap[strings.ToLower(arcdpsIntegrationPath)]; exists {
			processLoadChain(integrationInfo, dllMap, gw2Path, loadOrder, processed, LoadTypeNexusAddon, currentLoadOrder)
		}
	}

	if info.IsGw2Load {
		addonsPath := filepath.Join(filepath.Dir(info.FilePath), "addons")
		// Search through dllMap for GW2Load addons
		for dllPath, dllInfo := range dllMap {
			lowerDllPath := strings.ToLower(dllPath)
			if dllInfo.IsGw2LoadAddon && strings.HasPrefix(lowerDllPath, strings.ToLower(addonsPath)) {
				// Skip files in folders starting with . or _
				dirName := filepath.Base(filepath.Dir(dllPath))
				if !strings.HasPrefix(dirName, ".") && !strings.HasPrefix(dirName, "_") {
					processLoadChain(dllInfo, dllMap, gw2Path, loadOrder, processed, LoadTypeGw2LoadAddon, currentLoadOrder)
				}
			}
		}
	}
}

// Helper function to get the source description for a DLL
func getDllSource(loadType LoadType) string {
	switch loadType {
	case LoadTypeShim:
		return "loaded as shim"
	case LoadTypeArcdpsAddon:
		return "loaded by arcdps"
	case LoadTypeAddonLoaderAddon:
		return "loaded by addon loader"
	case LoadTypeNexusAddon:
		return "loaded by nexus"
	case LoadTypeGw2LoadAddon:
		return "loaded by gw2load"
	default:
		return "unknown load type"
	}
}
