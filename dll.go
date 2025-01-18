package main

import (
	"bytes"
	"fmt"
	"os"
	"crypto/md5"
	"encoding/hex"

	"github.com/pkg/errors"
	peparser "github.com/saferwall/pe"
	"go.uber.org/zap"
)

const (
	nexusApiUrl = "https://api.raidcore.gg"
)

var (
	addonLoaderDllUtf16             = asciiToWideString("addonLoader.dll")
	addonLoaderCoreDescriptionUtf16 = asciiToWideString("core addon loading library")
)

type dllInfo struct {
	filePath           string
	md5sum			   string
	isArcdps           bool
	isArcdpsAddon      bool
	isAddonLoaderShim  bool
	isAddonLoaderCore  bool
	isAddonLoaderAddon bool
	isNexus            bool
	isNexusAddon       bool
	isD3D11Shim        bool
	isDXGIShim         bool
	fileVersion        WinVersion
}

func (info *dllInfo) String() string {
	return fmt.Sprintf("md5sum: %v, isArcdps: %v, isArcdpsAddon: %v, isAddonLoaderShim: %v, isAddonLoaderCore: %v, isAddonLoaderAddon: %v, isNexus: %v, isNexusAddon: %v, isD3D11Shim: %v, isDXGIShim: %v, fileVersion: %v",
		info.md5sum, info.isArcdps, info.isArcdpsAddon, info.isAddonLoaderShim, info.isAddonLoaderCore, info.isAddonLoaderAddon, info.isNexus, info.isNexusAddon, info.isD3D11Shim, info.isDXGIShim, info.fileVersion)
}

// parseDll parses a DLL and returns information about the DLL
func parseDll(logger *zap.SugaredLogger, dllPath string) (*dllInfo, error) {
	info := &dllInfo{
		filePath: dllPath,
	}
	// Parse PE file
	peFile, err := peparser.New(dllPath, &peparser.Options{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open PE file %s", dllPath)
	}
	defer peFile.Close()
	if err = peFile.Parse(); err != nil {
		return nil, errors.Wrapf(err, "failed to parse PE file %s", dllPath)
	}
	// Load file as bytes
	fileBytes, err := os.ReadFile(dllPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", dllPath)
	}

	// Get file version if available
	if winVersion, err := GetFileVersion(dllPath); err == nil {
		info.fileVersion = winVersion
	}

	// Get MD5 sum
	if md5sum, err := GetMD5Sum(fileBytes); err == nil {
		info.md5sum = md5sum
	}

	// Check if the DLL is a D3D11 shim
	if isD3D11Shim(peFile) {
		info.isD3D11Shim = true
	}

	// Check if the DLL is a DXGI shim
	if isDXGIShim(peFile) {
		info.isDXGIShim = true
	}

	// Check if the DLL is arcdps
	if isArcdps(peFile) {
		info.isArcdps = true
	}

	// Check if the DLL is an arcdps addon
	if isArcdpsAddon(peFile) {
		info.isArcdpsAddon = true
	}

	// Check if the DLL is an addon loader shim
	if isAddonLoaderShim(peFile, fileBytes) {
		info.isAddonLoaderShim = true
	}

	// Check if the DLL is an addon loader core
	if isAddonLoaderCore(peFile, fileBytes) {
		info.isAddonLoaderCore = true
	}

	// Check if the DLL is an addon loader addon
	if isAddonLoaderAddon(peFile) {
		info.isAddonLoaderAddon = true
	}

	// Check if the DLL is Nexus
	if isNexus(peFile, fileBytes) {
		info.isNexus = true
	}

	// Check if the DLL is a Nexus addon
	if isNexusAddon(peFile) {
		info.isNexusAddon = true
	}

	return info, nil
}

func isArcdps(peFile *peparser.File) bool {
	// Check for e0
	for _, export := range peFile.Export.Functions {
		if export.Name == "e0" {
			return true
		}
	}
	return false
}

func isArcdpsAddon(peFile *peparser.File) bool {
	// Check for get_init_addr
	for _, export := range peFile.Export.Functions {
		if export.Name == "get_init_addr" {
			return true
		}
	}
	return false
}

func isAddonLoaderAddon(peFile *peparser.File) bool {
	// Check for gw2addon_load
	for _, export := range peFile.Export.Functions {
		if export.Name == "gw2addon_load" {
			return true
		}
	}
	return false
}

func isNexus(peFile *peparser.File, fileBytes []byte) bool {
	// Check if it is a shim
	if !isD3D11Shim(peFile) {
		return false
	}
	// Check for the Nexus API URL
	apiUrlPresent := bytes.Contains(fileBytes, []byte(nexusApiUrl))
	return apiUrlPresent
}

func isNexusAddon(peFile *peparser.File) bool {
	// Check for GetAddonDef
	for _, export := range peFile.Export.Functions {
		if export.Name == "GetAddonDef" {
			return true
		}
	}
	return false
}

func isDXGIShim(peFile *peparser.File) bool {
	// Check for CreateDXGIFactory
	for _, export := range peFile.Export.Functions {
		if export.Name == "CreateDXGIFactory" {
			return true
		}
	}
	return false
}

func isD3D11Shim(peFile *peparser.File) bool {
	// Check for D3D11CreateDevice
	for _, export := range peFile.Export.Functions {
		if export.Name == "D3D11CreateDevice" {
			return true
		}
	}
	return false
}

func isAddonLoaderShim(peFile *peparser.File, fileBytes []byte) bool {
	// The shim must be one of dxgi.dll or d3d11.dll
	if !isDXGIShim(peFile) && !isD3D11Shim(peFile) {
		return false
	}
	// Check if there's an addonLoader.dll string
	loaderStringPresent := bytes.Contains(fileBytes, addonLoaderDllUtf16)
	return loaderStringPresent
}

func isAddonLoaderCore(peFile *peparser.File, fileBytes []byte) bool {
	if !isDXGIShim(peFile) || !isD3D11Shim(peFile) {
		return false
	}
	// Check if there's the description string
	loaderStringPresent := bytes.Contains(fileBytes, addonLoaderCoreDescriptionUtf16)
	return loaderStringPresent
}

func asciiToWideString(s string) []byte {
	b := make([]byte, len(s)*2)
	for i, c := range s {
		b[i*2] = byte(c)
		b[i*2+1] = 0
	}
	return b
}

// GetMD5Sum calculates the MD5 checksum of the given data and returns it as a hexadecimal string.
func GetMD5Sum(data []byte) (string, error) {
	hash := md5.New()
	_, err := hash.Write(data)
	if err != nil {
		return "", err
	}
	checksum := hash.Sum(nil)
	return hex.EncodeToString(checksum), nil
}