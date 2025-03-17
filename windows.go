//go:build windows
// +build windows

package main

import (
	"fmt"
	"path/filepath"
	"time"
	"unsafe"

	"github.com/cheahjs/gw2-addon-setup-debug/ui/process_modules"
	"golang.org/x/sys/windows"
)

type ProcessEntry32 struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [windows.MAX_PATH]uint16
}

type ModuleEntry32 struct {
	Size         uint32
	ModuleID     uint32
	ProcessID    uint32
	GlblcntUsage uint32
	ProccntUsage uint32
	ModBaseAddr  *byte
	ModBaseSize  uint32
	HModule      windows.Handle
	Module       [windows.MAX_PATH]uint16
	ExePath      [windows.MAX_PATH]uint16
}

var (
	modkernel32                   = windows.NewLazySystemDLL("kernel32.dll")
	procCreateToolhelp32Snapshot  = modkernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First            = modkernel32.NewProc("Process32FirstW")
	procProcess32Next             = modkernel32.NewProc("Process32NextW")
	procModule32First             = modkernel32.NewProc("Module32FirstW")
	procModule32Next              = modkernel32.NewProc("Module32NextW")
	procGetProcessImageFileName   = modkernel32.NewProc("K32GetProcessImageFileNameW")
	procGetModuleFileNameEx       = modkernel32.NewProc("K32GetModuleFileNameExW")
	procQueryFullProcessImageName = modkernel32.NewProc("QueryFullProcessImageNameW")
	procGetCurrentProcess         = modkernel32.NewProc("GetCurrentProcess")
	procGetCommandLine            = modkernel32.NewProc("GetCommandLineW")
	procGetCurrentDirectory       = modkernel32.NewProc("GetCurrentDirectoryW")
)

const (
	TH32CS_SNAPPROCESS        = 0x00000002
	TH32CS_SNAPMODULE         = 0x00000008
	PROCESS_QUERY_INFORMATION = 0x0400
	PROCESS_VM_READ           = 0x0010
)

func CreateToolhelp32Snapshot(flags uint32, processID uint32) (windows.Handle, error) {
	ret, _, err := procCreateToolhelp32Snapshot.Call(
		uintptr(flags),
		uintptr(processID),
	)
	if ret == 0 {
		return 0, err
	}
	return windows.Handle(ret), nil
}

func Process32First(snapshot windows.Handle, pe *ProcessEntry32) error {
	pe.Size = uint32(unsafe.Sizeof(*pe))
	ret, _, err := procProcess32First.Call(
		uintptr(snapshot),
		uintptr(unsafe.Pointer(pe)),
	)
	if ret == 0 {
		return err
	}
	return nil
}

func Process32Next(snapshot windows.Handle, pe *ProcessEntry32) error {
	pe.Size = uint32(unsafe.Sizeof(*pe))
	ret, _, err := procProcess32Next.Call(
		uintptr(snapshot),
		uintptr(unsafe.Pointer(pe)),
	)
	if ret == 0 {
		return err
	}
	return nil
}

func Module32First(snapshot windows.Handle, me *ModuleEntry32) error {
	me.Size = uint32(unsafe.Sizeof(*me))
	ret, _, err := procModule32First.Call(
		uintptr(snapshot),
		uintptr(unsafe.Pointer(me)),
	)
	if ret == 0 {
		return err
	}
	return nil
}

func Module32Next(snapshot windows.Handle, me *ModuleEntry32) error {
	me.Size = uint32(unsafe.Sizeof(*me))
	ret, _, err := procModule32Next.Call(
		uintptr(snapshot),
		uintptr(unsafe.Pointer(me)),
	)
	if ret == 0 {
		return err
	}
	return nil
}

func GetProcessImageFileName(hProcess windows.Handle, filename *uint16, size uint32) (uint32, error) {
	ret, _, err := procGetProcessImageFileName.Call(
		uintptr(hProcess),
		uintptr(unsafe.Pointer(filename)),
		uintptr(size),
	)
	if ret == 0 {
		return 0, err
	}
	return uint32(ret), nil
}

func QueryFullProcessImageName(hProcess windows.Handle, flags uint32, filename *uint16, size *uint32) bool {
	ret, _, _ := procQueryFullProcessImageName.Call(
		uintptr(hProcess),
		uintptr(flags),
		uintptr(unsafe.Pointer(filename)),
		uintptr(unsafe.Pointer(size)),
	)
	return ret != 0
}

func FindGW2Process() (*process_modules.ProcessInfo, error) {
	snapshot, err := CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("could not create snapshot: %v", err)
	}
	defer windows.CloseHandle(snapshot)

	var pe ProcessEntry32
	err = Process32First(snapshot, &pe)
	if err != nil {
		return nil, fmt.Errorf("could not get first process: %v", err)
	}

	for {
		processName := windows.UTF16ToString(pe.ExeFile[:])
		if processName == "Gw2-64.exe" {
			// Found GW2 process, get details
			process, err := windows.OpenProcess(PROCESS_QUERY_INFORMATION|PROCESS_VM_READ, false, pe.ProcessID)
			if err != nil {
				return nil, fmt.Errorf("could not open process: %v", err)
			}
			defer windows.CloseHandle(process)

			// Get the full path to the executable
			var exePath [windows.MAX_PATH]uint16
			size := uint32(len(exePath))
			if !QueryFullProcessImageName(process, 0, &exePath[0], &size) {
				return nil, fmt.Errorf("could not get process image name")
			}
			executablePath := windows.UTF16ToString(exePath[:size])

			// Get working directory - use executable directory
			workingDir := filepath.Dir(executablePath)

			// Get the modules
			modulesSnapshot, err := CreateToolhelp32Snapshot(TH32CS_SNAPMODULE, pe.ProcessID)
			if err != nil {
				return nil, fmt.Errorf("could not create module snapshot: %v", err)
			}
			defer windows.CloseHandle(modulesSnapshot)

			var me ModuleEntry32
			err = Module32First(modulesSnapshot, &me)
			if err != nil {
				return nil, fmt.Errorf("could not get first module: %v", err)
			}

			var loadedModules []string
			for {
				modulePath := windows.UTF16ToString(me.ExePath[:])
				if modulePath != "" {
					loadedModules = append(loadedModules, modulePath)
				}

				err = Module32Next(modulesSnapshot, &me)
				if err != nil {
					break // No more modules
				}
			}

			return &process_modules.ProcessInfo{
				ProcessID:      int(pe.ProcessID),
				ExecutablePath: executablePath,
				WorkingDir:     workingDir,
				LoadedModules:  loadedModules,
				CommandLine:    "", // Not getting command line for now
				Timestamp:      time.Now(),
			}, nil
		}

		err = Process32Next(snapshot, &pe)
		if err != nil {
			break // No more processes
		}
	}

	return nil, fmt.Errorf("Guild Wars 2 process not found")
}
