//go:build windows
// +build windows

package utils

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/cheahjs/gw2-addon-setup-debug/ui/process_modules"
	"golang.org/x/sys/windows"
)

var (
	psapi    = windows.NewLazySystemDLL("psapi.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	enumProcessModules   = psapi.NewProc("EnumProcessModules")
	getModuleFileNameEx  = psapi.NewProc("GetModuleFileNameExW")
	getModuleInformation = psapi.NewProc("GetModuleInformation")
)

type ModuleInfo struct {
	ModuleName  string
	BaseAddress uintptr
	ModuleSize  uint32
}

// MODULEINFO represents the structure returned by GetModuleInformation
type MODULEINFO struct {
	BaseOfDll   uintptr
	SizeOfImage uint32
	EntryPoint  uintptr
}

func FindGW2Process() (*process_modules.ProcessInfo, error) {
	// Create a snapshot of running processes
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot failed: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	var processEntry windows.ProcessEntry32
	processEntry.Size = uint32(unsafe.Sizeof(processEntry))

	// Get the first process
	err = windows.Process32First(snapshot, &processEntry)
	if err != nil {
		return nil, fmt.Errorf("Process32First failed: %w", err)
	}

	// Iterate through processes
	for {
		// Convert the process name from UTF16 to a Go string
		name := windows.UTF16ToString(processEntry.ExeFile[:])

		// Check if this is GW2
		if strings.EqualFold(name, "gw2-64.exe") {
			pid := processEntry.ProcessID

			// Open the process to get a handle
			handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, pid)
			if err != nil {
				return nil, fmt.Errorf("failed to open process: %w", err)
			}
			defer windows.CloseHandle(handle)

			// Get the executable path
			execPath, err := getProcessPath(handle)
			if err != nil {
				return nil, fmt.Errorf("failed to get process path: %w", err)
			}

			// Get loaded modules
			moduleList, err := listProcessModules(pid)
			if err != nil {
				// If ToolHelp32 approach failed, try PSAPI approach
				moduleList, err = listProcessModulesPSAPI(handle)
				if err != nil {
					return nil, fmt.Errorf("failed to list process modules: %w", err)
				}
			}

			// Convert ModuleInfo to string paths for compatibility
			loadedModules := make([]string, len(moduleList))
			for i, module := range moduleList {
				loadedModules[i] = module.ModuleName
			}

			return &process_modules.ProcessInfo{
				ProcessID:      pid,
				ExecutablePath: execPath,
				LoadedModules:  loadedModules,
				WorkingDir:     filepath.Dir(execPath),
				CommandLine:    execPath,
				Timestamp:      time.Now(),
			}, nil
		}

		err = windows.Process32Next(snapshot, &processEntry)
		if err != nil {
			if err == syscall.ERROR_NO_MORE_FILES {
				break
			}
			return nil, fmt.Errorf("Process32Next failed: %w", err)
		}
	}

	return nil, fmt.Errorf("no GW2 process found")
}

func getProcessPath(handle windows.Handle) (string, error) {
	var pathLen uint32 = windows.MAX_PATH
	var pathBuf []uint16 = make([]uint16, pathLen)

	err := windows.QueryFullProcessImageName(handle, 0, &pathBuf[0], &pathLen)
	if err != nil {
		return "", fmt.Errorf("QueryFullProcessImageName failed: %w", err)
	}

	return windows.UTF16ToString(pathBuf[:pathLen]), nil
}

func listProcessModules(pid uint32) ([]ModuleInfo, error) {
	var modules []ModuleInfo

	// First try with both 32 and 64 bit modules
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPMODULE|windows.TH32CS_SNAPMODULE32, pid)
	if err != nil {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot failed: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	var moduleEntry windows.ModuleEntry32
	moduleEntry.Size = uint32(unsafe.Sizeof(moduleEntry))

	// Get the first module
	err = windows.Module32First(snapshot, &moduleEntry)
	if err != nil {
		// Return a more descriptive error for debugging
		return nil, fmt.Errorf("Module32First failed with pid %d: %w", pid, err)
	}

	for {
		moduleName := windows.UTF16ToString(moduleEntry.Module[:])
		modulePath := windows.UTF16ToString(moduleEntry.ExePath[:])

		// Use full path if available, otherwise use module name
		moduleNameToUse := modulePath
		if moduleNameToUse == "" {
			moduleNameToUse = moduleName
		}

		modules = append(modules, ModuleInfo{
			ModuleName:  moduleNameToUse,
			BaseAddress: uintptr(moduleEntry.ModBaseAddr),
			ModuleSize:  moduleEntry.ModBaseSize,
		})

		err = windows.Module32Next(snapshot, &moduleEntry)
		if err != nil {
			if err == syscall.ERROR_NO_MORE_FILES {
				break
			}
			return nil, fmt.Errorf("Module32Next failed: %w", err)
		}
	}

	return modules, nil
}

func listProcessModulesPSAPI(handle windows.Handle) ([]ModuleInfo, error) {
	var modules []ModuleInfo

	// Start with a reasonable buffer size
	const initialSize = 1024
	moduleHandles := make([]windows.Handle, initialSize)
	var needed uint32

	// First call to get the actual number of modules
	r, _, err := enumProcessModules.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&moduleHandles[0])),
		uintptr(initialSize*int(unsafe.Sizeof(windows.Handle(0)))),
		uintptr(unsafe.Pointer(&needed)))

	if r == 0 {
		return nil, fmt.Errorf("EnumProcessModules failed: %v", err)
	}

	// Calculate number of modules and resize the buffer if necessary
	numModules := needed / uint32(unsafe.Sizeof(windows.Handle(0)))
	if numModules > initialSize {
		moduleHandles = make([]windows.Handle, numModules)
		r, _, err = enumProcessModules.Call(
			uintptr(handle),
			uintptr(unsafe.Pointer(&moduleHandles[0])),
			uintptr(numModules*uint32(unsafe.Sizeof(windows.Handle(0)))),
			uintptr(unsafe.Pointer(&needed)))

		if r == 0 {
			return nil, fmt.Errorf("EnumProcessModules (resized) failed: %v", err)
		}
	}

	// Reset numModules based on what we got back
	numModules = needed / uint32(unsafe.Sizeof(windows.Handle(0)))
	if numModules == 0 {
		// Try to at least get the main module
		moduleHandles = moduleHandles[:1]
		r, _, _ = enumProcessModules.Call(
			uintptr(handle),
			uintptr(unsafe.Pointer(&moduleHandles[0])),
			uintptr(unsafe.Sizeof(moduleHandles[0])),
			uintptr(unsafe.Pointer(&needed)))

		if r == 0 {
			return modules, nil // Return empty list if we can't even get the main module
		}
		numModules = 1
	}

	// Process each module
	for i := uint32(0); i < numModules; i++ {
		if moduleHandles[i] == 0 {
			continue
		}

		// Get module file name
		var modulePath [windows.MAX_PATH]uint16
		r, _, _ := getModuleFileNameEx.Call(
			uintptr(handle),
			uintptr(moduleHandles[i]),
			uintptr(unsafe.Pointer(&modulePath[0])),
			uintptr(len(modulePath)))

		// If we got a path, add the module
		if r > 0 {
			// Try to get module information
			var moduleInfo MODULEINFO
			r2, _, _ := getModuleInformation.Call(
				uintptr(handle),
				uintptr(moduleHandles[i]),
				uintptr(unsafe.Pointer(&moduleInfo)),
				uintptr(unsafe.Sizeof(moduleInfo)))

			modInfo := ModuleInfo{
				ModuleName:  windows.UTF16ToString(modulePath[:r]),
				BaseAddress: uintptr(moduleHandles[i]),
				ModuleSize:  0,
			}

			// If we got module info successfully, use that
			if r2 != 0 {
				modInfo.BaseAddress = moduleInfo.BaseOfDll
				modInfo.ModuleSize = moduleInfo.SizeOfImage
			}

			modules = append(modules, modInfo)
		}
	}

	// Return what we have, even if empty
	return modules, nil
}

func isGW2Process(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), "gw2-64.exe")
}
