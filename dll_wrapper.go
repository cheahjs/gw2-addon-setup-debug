package main

import (
	"strconv"

	"github.com/cheahjs/gw2-addon-setup-debug/ui/scan_directory"
	"go.uber.org/zap"
)

func ScanDll(logger *zap.SugaredLogger, dllPath string) (*scan_directory.DllInfo, error) {
	// Use the existing DLL parsing logic from dll.go
	info, err := parseDll(logger, dllPath)
	if err != nil {
		return nil, err
	}

	// Convert the dllInfo to scan_directory.DllInfo
	result := &scan_directory.DllInfo{
		FilePath:           info.filePath,
		Md5sum:             info.md5sum,
		IsArcdps:           info.isArcdps,
		IsArcdpsAddon:      info.isArcdpsAddon,
		IsAddonLoaderShim:  info.isAddonLoaderShim,
		IsAddonLoaderCore:  info.isAddonLoaderCore,
		IsAddonLoaderAddon: info.isAddonLoaderAddon,
		IsNexus:            info.isNexus,
		IsNexusAddon:       info.isNexusAddon,
		IsD3D11Shim:        info.isD3D11Shim,
		IsDXGIShim:         info.isDXGIShim,
	}

	// Convert the file version
	if info.fileVersion.Major > 0 || info.fileVersion.Minor > 0 || info.fileVersion.Patch > 0 || info.fileVersion.Build > 0 {
		result.FileVersion = strconv.FormatUint(uint64(info.fileVersion.Major), 10) + "." +
			strconv.FormatUint(uint64(info.fileVersion.Minor), 10) + "." +
			strconv.FormatUint(uint64(info.fileVersion.Patch), 10) + "." +
			strconv.FormatUint(uint64(info.fileVersion.Build), 10)
	} else {
		result.FileVersion = "Unknown"
	}

	return result, nil
}
