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
	IsD3D11Shim        bool
	IsDXGIShim         bool
	FileVersion        WinVersion
	Error              string
}

func (info *DllInfo) String() string {
	return fmt.Sprintf("md5sum: %v, isArcdps: %v, isArcdpsAddon: %v, isAddonLoaderShim: %v, isAddonLoaderCore: %v, isAddonLoaderAddon: %v, isNexus: %v, isNexusAddon: %v, isD3D11Shim: %v, isDXGIShim: %v, fileVersion: %v",
		info.Md5sum, info.IsArcdps, info.IsArcdpsAddon, info.IsAddonLoaderShim, info.IsAddonLoaderCore, info.IsAddonLoaderAddon, info.IsNexus, info.IsNexusAddon, info.IsD3D11Shim, info.IsDXGIShim, info.FileVersion)
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

	// Get file version if available
	if winVersion, err := GetFileVersion(dllPath); err == nil {
		info.FileVersion = winVersion
	}

	// Get MD5 sum
	if md5sum, err := getMD5Sum(dllPath); err == nil {
		info.Md5sum = md5sum
	}

	// Check if the DLL is a D3D11 shim
	if isD3D11Shim(peFile) {
		info.IsD3D11Shim = true
	}

	// Check if the DLL is a DXGI shim
	if isDXGIShim(peFile) {
		info.IsDXGIShim = true
	}

	// Check if the DLL is arcdps
	if isArcdps(peFile) {
		info.IsArcdps = true
	}

	// Check if the DLL is an arcdps addon
	if isArcdpsAddon(peFile) {
		info.IsArcdpsAddon = true
	}

	// Check if the DLL is an addon loader shim
	if isAddonLoaderShim(peFile, dllPath) {
		info.IsAddonLoaderShim = true
	}

	// Check if the DLL is an addon loader core
	if isAddonLoaderCore(peFile, dllPath) {
		info.IsAddonLoaderCore = true
	}

	// Check if the DLL is an addon loader addon
	if isAddonLoaderAddon(peFile) {
		info.IsAddonLoaderAddon = true
	}

	// Check if the DLL is Nexus
	if isNexus(peFile, dllPath) {
		info.IsNexus = true
	}

	// Check if the DLL is a Nexus addon
	if isNexusAddon(peFile) {
		info.IsNexusAddon = true
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

func isNexus(peFile *peparser.File, dllPath string) bool {
	// Check if it is a shim
	if !isD3D11Shim(peFile) {
		return false
	}
	// Check for the Nexus API URL
	apiUrlPresent, err := searchBytesInLargeFile(dllPath, []byte(nexusApiUrl))
	if err != nil {
		return false
	}
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

func isAddonLoaderShim(peFile *peparser.File, dllPath string) bool {
	// The shim must be one of dxgi.dll or d3d11.dll
	if !isDXGIShim(peFile) && !isD3D11Shim(peFile) {
		return false
	}
	// Check if there's an addonLoader.dll string
	loaderStringPresent, err := searchBytesInLargeFile(dllPath, addonLoaderDllUtf16)
	if err != nil {
		return false
	}
	return loaderStringPresent
}

func isAddonLoaderCore(peFile *peparser.File, dllPath string) bool {
	if !isDXGIShim(peFile) || !isD3D11Shim(peFile) {
		return false
	}
	// Check if there's the description string
	loaderStringPresent, err := searchBytesInLargeFile(dllPath, addonLoaderCoreDescriptionUtf16)
	if err != nil {
		return false
	}
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
