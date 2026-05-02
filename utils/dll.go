package utils

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

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
	IsQuarantined      bool
	IsReshade          bool
	FileVersion        WinVersion
	FileDescription    string
	ProductName        string
	ProductVersion     string
	Error              string
}

func (info *DllInfo) String() string {
	return fmt.Sprintf(
		"md5sum: %v, fileDesc: %v, productName: %v, productVer: %v, isArcdps: %v, isArcdpsAddon: %v, isAddonLoaderShim: %v, isAddonLoaderCore: %v, isAddonLoaderAddon: %v, isNexus: %v, isNexusAddon: %v, isD3D11Shim: %v, isDXGIShim: %v, isGw2Load: %v, isGw2LoadAddon: %v, isQuarantined: %v, isReshade: %v, fileVersion: %v",
		info.Md5sum,
		info.FileDescription,
		info.ProductName,
		info.ProductVersion,
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
		info.IsQuarantined,
		info.IsReshade,
		info.FileVersion,
	)
}

func (info *DllInfo) Flags() string {
	var flags strings.Builder
	if info.IsNexus {
		flags.WriteString("[Nexus] ")
	}
	if info.IsArcdps {
		flags.WriteString("[Arcdps] ")
	}
	if info.IsAddonLoaderShim {
		flags.WriteString("[AddonLoaderShim] ")
	}
	if info.IsD3D11Shim {
		flags.WriteString("[D3D11Shim] ")
	}
	if info.IsDXGIShim {
		flags.WriteString("[DXGIShim] ")
	}
	if info.IsAddonLoaderCore {
		flags.WriteString("[AddonLoaderCore] ")
	}
	if info.IsAddonLoaderAddon {
		flags.WriteString("[AddonLoaderAddon] ")
	}
	if info.IsNexusAddon {
		flags.WriteString("[NexusAddon] ")
	}
	if info.IsArcdpsAddon {
		flags.WriteString("[ArcdpsAddon] ")
	}
	if info.IsGw2Load {
		flags.WriteString("[GW2Load] ")
	}
	if info.IsGw2LoadAddon {
		flags.WriteString("[GW2LoadAddon] ")
	}
	if info.IsReshade {
		flags.WriteString("[Reshade] ")
	}
	if info.IsQuarantined {
		flags.WriteString("[Quarantined] ")
	}
	return flags.String()
}

// ParseDll parses a DLL and returns information about the DLL
func ParseDll(logger *zap.SugaredLogger, dllPath string) (*DllInfo, error) {
	info := &DllInfo{
		FilePath: dllPath,
	}

	// Check if file is quarantined by Windows
	if isQuarantined, err := checkFileQuarantined(dllPath); err == nil {
		info.IsQuarantined = isQuarantined
	}

	// Get file version info (numeric + strings) in a single read
	if winVer, fileDesc, prodName, prodVer, err := GetFileVersionAll(dllPath); err == nil {
		info.FileVersion = winVer
		info.FileDescription = fileDesc
		info.ProductName = prodName
		info.ProductVersion = prodVer
	}

	// Read the entire DLL into memory once. This buffer is reused for:
	// - MD5 computation
	// - Byte pattern searching (Nexus API URL, addonLoader strings, etc.)
	// This avoids re-opening and re-reading the file multiple times.
	fileData, err := os.ReadFile(dllPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", dllPath)
	}

	// Compute MD5 from the in-memory buffer (no extra file read)
	info.Md5sum = getMD5SumFromData(fileData)

	// Parse PE file
	peFile, err := peparser.New(dllPath, &peparser.Options{
		OmitImportDirectory:       true,
		OmitSecurityDirectory:     true,
		OmitRelocDirectory:        true,
		OmitDebugDirectory:        true,
		OmitArchitectureDirectory: true,
		OmitGlobalPtrDirectory:    true,
		OmitTLSDirectory:          true,
		OmitLoadConfigDirectory:   true,
		OmitBoundImportDirectory:  true,
		OmitIATDirectory:          true,
		OmitDelayImportDirectory:  true,
		OmitCLRHeaderDirectory:    true,
		OmitCLRMetadata:           true,
	})
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

	// Check if the DLL is an addon loader shim (uses in-memory search)
	if isAddonLoaderShim(exports, fileData) {
		info.IsAddonLoaderShim = true
	}

	// Check if the DLL is an addon loader core (uses in-memory search)
	if isAddonLoaderCore(exports, fileData) {
		info.IsAddonLoaderCore = true
	}

	// Check if the DLL is an addon loader addon
	if isAddonLoaderAddon(exports) {
		info.IsAddonLoaderAddon = true
	}

	// Check if the DLL is Nexus (uses in-memory search)
	if isNexus(exports, fileData) {
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

	// Check if the DLL is Reshade
	if isReshade(exports) {
		info.IsReshade = true
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

func isNexus(exports map[string]struct{}, fileData []byte) bool {
	// Check if it is a shim
	if !isD3D11Shim(exports) {
		return false
	}
	// Check for the Nexus API URL in already-loaded file data
	return bytes.Contains(fileData, []byte(nexusApiUrl))
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

func isAddonLoaderShim(exports map[string]struct{}, fileData []byte) bool {
	// The shim must be one of dxgi.dll or d3d11.dll
	if !isDXGIShim(exports) && !isD3D11Shim(exports) {
		return false
	}
	// Check if there's an addonLoader.dll string in already-loaded file data
	return bytes.Contains(fileData, addonLoaderDllUtf16)
}

func isAddonLoaderCore(exports map[string]struct{}, fileData []byte) bool {
	if !isDXGIShim(exports) || !isD3D11Shim(exports) {
		return false
	}
	// Check if there's the description string in already-loaded file data
	return bytes.Contains(fileData, addonLoaderCoreDescriptionUtf16)
}

func isGw2Load(exports map[string]struct{}) bool {
	_, exists := exports["GW2Load_CheckIfAddon"]
	return exists
}

func isGw2LoadAddon(exports map[string]struct{}) bool {
	_, exists := exports["GW2Load_GetAddonAPIVersion"]
	return exists
}

func isReshade(exports map[string]struct{}) bool {
	_, exists := exports["ReShadeVersion"]
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

// getMD5SumFromData calculates the MD5 checksum from an in-memory byte slice.
func getMD5SumFromData(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

func checkFileQuarantined(filePath string) (bool, error) {
	// Windows stores Zone.Identifier as an alternate data stream
	zoneIdentifierPath := filePath + ":Zone.Identifier"

	// Read the contents to check for ZoneId=3 (Internet) or ZoneId=4 (Restricted)
	content, err := os.ReadFile(zoneIdentifierPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // File is not quarantined
		}
		return false, err
	}

	contentStr := string(content)
	return strings.Contains(contentStr, "ZoneId=3") || strings.Contains(contentStr, "ZoneId=4"), nil
}
