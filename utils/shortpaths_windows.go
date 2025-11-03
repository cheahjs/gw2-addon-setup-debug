//go:build windows

package utils

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// ShortPathsEnabled returns true if the volume containing the supplied path has 8.3 short paths enabled.
func ShortPathsEnabled(path string) (bool, error) {
	if path == "" {
		return false, fmt.Errorf("path is empty")
	}

	utf16Path, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false, fmt.Errorf("invalid path: %w", err)
	}

	volumePath := make([]uint16, windows.MAX_PATH)
	if err := windows.GetVolumePathName(utf16Path, &volumePath[0], uint32(len(volumePath))); err != nil {
		return false, fmt.Errorf("get volume path name: %w", err)
	}

	var fsFlags uint32
	if err := windows.GetVolumeInformation(&volumePath[0], nil, 0, nil, nil, &fsFlags, nil, 0); err != nil {
		return false, fmt.Errorf("get volume information: %w", err)
	}

	const fileSupportsShortNames = 0x00000040
	return fsFlags&fileSupportsShortNames != 0, nil
}
