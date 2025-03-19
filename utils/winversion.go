// https://github.com/keybase/client/blob/04dfb34b132e041b13e4584012a75f5008dfa516/go/install/winversion.go/ Copyright 2018 Keybase Inc. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE_thirdparty file.
// Adapted mainly from github.com/gonutz/w32

//go:build windows
// +build windows

package utils

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	version                = windows.NewLazySystemDLL("version.dll")
	getFileVersionInfoSize = version.NewProc("GetFileVersionInfoSizeW")
	getFileVersionInfo     = version.NewProc("GetFileVersionInfoW")
	verQueryValue          = version.NewProc("VerQueryValueW")
)

type VS_FIXEDFILEINFO struct {
	Signature        uint32
	StrucVersion     uint32
	FileVersionMS    uint32
	FileVersionLS    uint32
	ProductVersionMS uint32
	ProductVersionLS uint32
	FileFlagsMask    uint32
	FileFlags        uint32
	FileOS           uint32
	FileType         uint32
	FileSubtype      uint32
	FileDateMS       uint32
	FileDateLS       uint32
}

type WinVersion struct {
	Major uint32
	Minor uint32
	Patch uint32
	Build uint32
}

// FileVersion concatenates FileVersionMS and FileVersionLS to a uint64 value.
func (fi VS_FIXEDFILEINFO) FileVersion() uint64 {
	return uint64(fi.FileVersionMS)<<32 | uint64(fi.FileVersionLS)
}

func GetFileVersionInfoSize(path string) uint32 {
	ret, _, _ := getFileVersionInfoSize.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path))),
		0,
	)
	return uint32(ret)
}

func GetFileVersionInfo(path string, data []byte) bool {
	ret, _, _ := getFileVersionInfo.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path))),
		0,
		uintptr(len(data)),
		uintptr(unsafe.Pointer(&data[0])),
	)
	return ret != 0
}

// VerQueryValueRoot calls VerQueryValue
// (https://msdn.microsoft.com/en-us/library/windows/desktop/ms647464(v=vs.85).aspx)
// with `\` (root) to retieve the VS_FIXEDFILEINFO.
func VerQueryValueRoot(block []byte) (VS_FIXEDFILEINFO, error) {
	var offset uintptr
	var length uint
	blockStart := unsafe.Pointer(&block[0])
	ret, _, _ := verQueryValue.Call(
		uintptr(blockStart),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(`\`))),
		uintptr(unsafe.Pointer(&offset)),
		uintptr(unsafe.Pointer(&length)),
	)
	if ret == 0 {
		return VS_FIXEDFILEINFO{}, errors.New("VerQueryValueRoot: verQueryValue failed")
	}
	start := int(offset) - int(uintptr(blockStart))
	end := start + int(length)
	if start < 0 || start >= len(block) || end < start || end > len(block) {
		return VS_FIXEDFILEINFO{}, errors.New("VerQueryValueRoot: find failed")
	}
	data := block[start:end]
	info := *((*VS_FIXEDFILEINFO)(unsafe.Pointer(&data[0])))
	return info, nil
}

func GetFileVersion(path string) (WinVersion, error) {
	var result WinVersion
	size := GetFileVersionInfoSize(path)
	if size <= 0 {
		return result, errors.New("GetFileVersionInfoSize failed")
	}

	info := make([]byte, size)
	ok := GetFileVersionInfo(path, info)
	if !ok {
		return result, errors.New("GetFileVersionInfo failed")
	}

	fixed, err := VerQueryValueRoot(info)
	if err != nil {
		return result, err
	}
	version := fixed.FileVersion()

	result.Major = uint32(version & 0xFFFF000000000000 >> 48)
	result.Minor = uint32(version & 0x0000FFFF00000000 >> 32)
	result.Patch = uint32(version & 0x00000000FFFF0000 >> 16)
	result.Build = uint32(version & 0x000000000000FFFF)

	return result, nil
}

// GetFileVersionStrings retrieves string information from the version resource
func GetFileVersionStrings(path string) (fileDescription, productName, productVersion string, err error) {
	size := GetFileVersionInfoSize(path)
	if size <= 0 {
		return "", "", "", errors.New("GetFileVersionInfoSize failed")
	}

	info := make([]byte, size)
	ok := GetFileVersionInfo(path, info)
	if !ok {
		return "", "", "", errors.New("GetFileVersionInfo failed")
	}

	// Query the translation table
	var offset uintptr
	var length uint
	blockStart := unsafe.Pointer(&info[0])
	ret, _, _ := verQueryValue.Call(
		uintptr(blockStart),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(`\VarFileInfo\Translation`))),
		uintptr(unsafe.Pointer(&offset)),
		uintptr(unsafe.Pointer(&length)),
	)
	if ret == 0 {
		return "", "", "", errors.New("failed to query translation table")
	}

	start := int(offset) - int(uintptr(blockStart))
	end := start + int(length)
	if start < 0 || start >= len(info) || end < start || end > len(info) {
		return "", "", "", errors.New("invalid translation table offset")
	}

	// First translation entry
	langID := uint16(info[start])
	langID |= uint16(info[start+1]) << 8
	charsetID := uint16(info[start+2])
	charsetID |= uint16(info[start+3]) << 8

	// Build the version strings path with the language ID
	fileDescPath := fmt.Sprintf("\\StringFileInfo\\%04x%04x\\FileDescription", langID, charsetID)
	productNamePath := fmt.Sprintf("\\StringFileInfo\\%04x%04x\\ProductName", langID, charsetID)
	productVersionPath := fmt.Sprintf("\\StringFileInfo\\%04x%04x\\ProductVersion", langID, charsetID)

	// Query each string
	fileDescription, _ = queryVersionString(info, fileDescPath)
	productName, _ = queryVersionString(info, productNamePath)
	productVersion, _ = queryVersionString(info, productVersionPath)

	return fileDescription, productName, productVersion, nil
}

func queryVersionString(block []byte, path string) (string, error) {
	var offset uintptr
	var length uint
	blockStart := unsafe.Pointer(&block[0])
	ret, _, _ := verQueryValue.Call(
		uintptr(blockStart),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path))),
		uintptr(unsafe.Pointer(&offset)),
		uintptr(unsafe.Pointer(&length)),
	)
	if ret == 0 {
		return "", errors.New("failed to query version string")
	}

	start := int(offset) - int(uintptr(blockStart))
	end := start + int(length)
	if start < 0 || start >= len(block) || end < start || end > len(block) {
		return "", errors.New("invalid string offset")
	}

	// Convert UTF-16 to string
	data := block[start:end]
	ptr := (*uint16)(unsafe.Pointer(&data[0]))
	return windows.UTF16PtrToString(ptr), nil
}
