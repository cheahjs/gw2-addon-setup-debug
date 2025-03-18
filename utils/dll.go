package utils

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"

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
	IsGw2Load          bool
	IsGw2LoadAddon     bool
	IsD3D11Shim        bool
	IsDXGIShim         bool
	FileVersion        WinVersion
	Error              string
}

func (info *DllInfo) String() string {
	return fmt.Sprintf(
		"md5sum: %v, isArcdps: %v, isArcdpsAddon: %v, isAddonLoaderShim: %v, isAddonLoaderCore: %v, isAddonLoaderAddon: %v, isNexus: %v, isNexusAddon: %v, isD3D11Shim: %v, isDXGIShim: %v, isGw2Load: %v, isGw2LoadAddon: %v, fileVersion: %v",
		info.Md5sum,
		info.IsArcdps,
		info.IsArcdpsAddon,
		info.IsAddonLoaderShim,
		info.IsAddonLoaderCore,
		info.IsAddonLoaderAddon,
		info.IsNexus,
		info.IsNexusAddon,
		info.IsD3D11Shim,
		info.IsDXGIShim,
		info.IsGw2Load,
		info.IsGw2LoadAddon,
		info.FileVersion,
	)
}

// parseDll parses a DLL and returns information about the DLL
func parseDll(logger *zap.SugaredLogger, dllPath string) (*DllInfo, error) {
	info := &DllInfo{
		FilePath: dllPath,
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
	// Make a map of the exports
	exports := make(map[string]struct{})
	for _, export := range peFile.Export.Functions {
		exports[export.Name] = struct{}{}
	}

	// Get file version if available
	if winVersion, err := GetFileVersion(dllPath); err == nil {
		info.FileVersion = winVersion
	}

	// Get MD5 sum
	if md5sum, err := getMD5Sum(dllPath); err == nil {
		info.Md5sum = md5sum
	}

	// Check if the DLL is a D3D11 shim
	if isD3D11Shim(exports) {
		info.IsD3D11Shim = true
	}

	// Check if the DLL is a DXGI shim
	if isDXGIShim(exports) {
		info.IsDXGIShim = true
	}

	// Check if the DLL is arcdps
	if isArcdps(exports) {
		info.IsArcdps = true
	}

	// Check if the DLL is an arcdps addon
	if isArcdpsAddon(exports) {
		info.IsArcdpsAddon = true
	}

	// Check if the DLL is an addon loader shim
	if isAddonLoaderShim(exports, dllPath) {
		info.IsAddonLoaderShim = true
	}

	// Check if the DLL is an addon loader core
	if isAddonLoaderCore(exports, dllPath) {
		info.IsAddonLoaderCore = true
	}

	// Check if the DLL is an addon loader addon
	if isAddonLoaderAddon(exports) {
		info.IsAddonLoaderAddon = true
	}

	// Check if the DLL is Nexus
	if isNexus(exports, dllPath) {
		info.IsNexus = true
	}

	// Check if the DLL is a Nexus addon
	if isNexusAddon(exports) {
		info.IsNexusAddon = true
	}

	// Check if the DLL is Gw2Load
	if isGw2Load(exports) {
		info.IsGw2Load = true
	}

	// Check if the DLL is a Gw2Load addon
	if isGw2LoadAddon(exports) {
		info.IsGw2LoadAddon = true
	}

	return info, nil
}

func isArcdps(exports map[string]struct{}) bool {
	_, exists := exports["e0"]
	return exists
}

func isArcdpsAddon(exports map[string]struct{}) bool {
	_, exists := exports["get_init_addr"]
	return exists
}

func isAddonLoaderAddon(exports map[string]struct{}) bool {
	_, exists := exports["gw2addon_load"]
	return exists
}

func isNexus(exports map[string]struct{}, dllPath string) bool {
	// Check if it is a shim
	if !isD3D11Shim(exports) {
		return false
	}
	// Check for the Nexus API URL
	apiUrlPresent, err := searchBytesInLargeFile(dllPath, []byte(nexusApiUrl))
	if err != nil {
		return false
	}
	return apiUrlPresent
}

func isNexusAddon(exports map[string]struct{}) bool {
	_, exists := exports["GetAddonDef"]
	return exists
}

func isDXGIShim(exports map[string]struct{}) bool {
	_, exists := exports["CreateDXGIFactory"]
	return exists
}

func isD3D11Shim(exports map[string]struct{}) bool {
	_, exists := exports["D3D11CreateDevice"]
	return exists
}

func isAddonLoaderShim(exports map[string]struct{}, dllPath string) bool {
	// The shim must be one of dxgi.dll or d3d11.dll
	if !isDXGIShim(exports) && !isD3D11Shim(exports) {
		return false
	}
	// Check if there's an addonLoader.dll string
	loaderStringPresent, err := searchBytesInLargeFile(dllPath, addonLoaderDllUtf16)
	if err != nil {
		return false
	}
	return loaderStringPresent
}

func isAddonLoaderCore(exports map[string]struct{}, dllPath string) bool {
	if !isDXGIShim(exports) || !isD3D11Shim(exports) {
		return false
	}
	// Check if there's the description string
	loaderStringPresent, err := searchBytesInLargeFile(dllPath, addonLoaderCoreDescriptionUtf16)
	if err != nil {
		return false
	}
	return loaderStringPresent
}

func isGw2Load(exports map[string]struct{}) bool {
	_, exists := exports["GW2Load_CheckIfAddon"]
	return exists
}

func isGw2LoadAddon(exports map[string]struct{}) bool {
	_, exists := exports["GW2Load_GetAddonAPIVersion"]
	return exists
}

func asciiToWideString(s string) []byte {
	b := make([]byte, len(s)*2)
	for i, c := range s {
		b[i*2] = byte(c)
		b[i*2+1] = 0
	}
	return b
}

// getMD5Sum calculates the MD5 checksum of the given file and returns it as a hexadecimal string.
func getMD5Sum(filePath string) (string, error) {
	// Calculate MD5 by copying the file into the hash
	file, err := os.Open(filePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to open file %s", filePath)
	}
	defer file.Close()
	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}
	checksum := hash.Sum(nil)
	return hex.EncodeToString(checksum), nil
}

func searchBytesInLargeFile(filename string, pattern []byte) (found bool, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Buffer size should be at least as large as the pattern
	bufSize := max(4096, len(pattern)*2)
	buf := make([]byte, bufSize)

	position := int64(0)
	overlap := len(pattern) - 1 // Used for overlapping reads

	// Read the first chunk
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}

	data := buf[:n]

	for {
		// Search for pattern in the current chunk
		index := bytes.Index(data, pattern)
		if index >= 0 {
			return true, nil
		}

		if err == io.EOF {
			break
		}

		// Keep part of the old buffer to handle pattern that might span chunks
		if n <= overlap {
			break // Not enough data to continue
		}

		// Move the last 'overlap' bytes to the beginning of the buffer
		copy(buf[:overlap], buf[n-overlap:n])

		// Read the next chunk after the overlap
		readPos := overlap
		n, err = file.Read(buf[readPos:])
		if err != nil && err != io.EOF {
			return false, err
		}

		// Update position and data slice
		position += int64(readPos)
		data = buf[:readPos+n]
	}

	return false, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
