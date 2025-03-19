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

	"golang.org/x/sys/windows"
)

type ModuleInfo struct {
	ModuleName  string
	BaseAddress uintptr
	ModuleSize  uint32
	EntryPoint  uintptr
}

type ProcessInfo struct {
	ProcessID      uint32
	ExecutablePath string
	LoadedModules  []ModuleInfo
	WorkingDir     string
	CommandLine    string
	Timestamp      time.Time
}

// MODULEINFO represents the structure returned by GetModuleInformation
type MODULEINFO struct {
	BaseOfDll   uintptr
	SizeOfImage uint32
	EntryPoint  uintptr
}

func FindGW2Process() (*ProcessInfo, error) {
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
			moduleList, err := listProcessModules(handle)
			if err != nil {
				return nil, fmt.Errorf("failed to list process modules: %w", err)
			}

			return &ProcessInfo{
				ProcessID:      pid,
				ExecutablePath: execPath,
				LoadedModules:  moduleList,
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

func listProcessModules(handle windows.Handle) ([]ModuleInfo, error) {
	var modules [1024]windows.Handle
	var cb = uint32(unsafe.Sizeof(modules))
	var needed uint32
	if err := windows.EnumProcessModulesEx(handle, &modules[0], cb, &needed, windows.LIST_MODULES_ALL); err != nil {
		return nil, fmt.Errorf("failed to enumerate process modules: %w", err)
	}
	count := needed / uint32(unsafe.Sizeof(modules[0]))

	var moduleInfos []ModuleInfo
	for i := uint32(0); i < count; i++ {
		var mi windows.ModuleInfo
		if err := windows.GetModuleInformation(handle, modules[i], &mi, uint32(unsafe.Sizeof(mi))); err != nil {
			return nil, fmt.Errorf("failed to get module information: %w", err)
		}

		var moduleName [windows.MAX_PATH]uint16
		if err := windows.GetModuleFileNameEx(handle, modules[i], &moduleName[0], uint32(len(moduleName))); err != nil {
			return nil, fmt.Errorf("failed to get module file name: %w", err)
		}

		moduleInfos = append(moduleInfos, ModuleInfo{
			BaseAddress: mi.BaseOfDll,
			ModuleSize:  mi.SizeOfImage,
			EntryPoint:  mi.EntryPoint,
			ModuleName:  syscall.UTF16ToString(moduleName[:]),
		})
	}

	return moduleInfos, nil
}

func IsRunningAsAdmin() (bool, error) {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		return false, fmt.Errorf("AllocateAndInitializeSid failed: %w", err)
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		return false, fmt.Errorf("IsMember failed: %w", err)
	}

	return member, nil
}
