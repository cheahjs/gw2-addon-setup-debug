//go:build windows

package utils

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// Get8dot3NameStatus determines the 8.3 name creation status for a given volume path.
func Get8dot3NameStatus(volumePath string) (string, error) {
	// 1. Determine the root of the given volumePath.
	volumeRoot := filepath.VolumeName(volumePath)
	if volumeRoot == "" {
		// Handle UNC paths or other complex paths if necessary.
		// For now, fsutil primarily works well with local drive letters.
		// A simple check: if it doesn't look like "C:", it might be a UNC or invalid.
		if strings.HasPrefix(volumePath, `\\`) {
			return "", fmt.Errorf("UNC paths (%s) are not directly supported by fsutil 8dot3name query; manual check required", volumePath)
		}
		return "", fmt.Errorf("could not determine volume root for path: %s", volumePath)
	}
	// Ensure the path is in the format "C:\"
	if !strings.HasSuffix(volumeRoot, `\`) {
		volumeRoot += `\`
	}

	// 2. Read the NtfsDisable8dot3NameCreation registry value.
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\FileSystem`, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("failed to open registry key: %w", err)
	}
	defer key.Close()

	ntfsDisable8dot3NameCreation, _, err := key.GetIntegerValue("NtfsDisable8dot3NameCreation")
	if err != nil {
		return "", fmt.Errorf("failed to read NtfsDisable8dot3NameCreation registry value: %w", err)
	}

	// 3. Interpret the registry value.
	switch ntfsDisable8dot3NameCreation {
	case 0:
		return "8.3 name creation is enabled (globally)", nil
	case 1:
		return "8.3 name creation is disabled (globally)", nil
	case 3:
		return "8.3 name creation is disabled for all volumes except the system volume (globally)", nil
	case 2:
		// 4. If the registry value is 2, proceed to query fsutil.
		cmd := exec.Command("fsutil.exe", "8dot3name", "query", volumeRoot)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to execute fsutil for volume %s: %w. Output: %s", volumeRoot, err, string(output))
		}

		// Parse the output.
		// Example outputs:
		// "The volume state is: 0 (8dot3 name creation is enabled)."
		// "The volume state is: 1 (8dot3 name creation is disabled)."
		// "The registry state is: 2 (Per volume setting - the default)."
        // "The volume state is: 0 (8dot3 name creation is enabled)." is the critical part for per-volume
		outputStr := strings.TrimSpace(string(output))
		lines := strings.Split(outputStr, "\n")
		
		// We are looking for the line describing the volume state.
		// fsutil output can be verbose, e.g. also printing the global registry state.
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			if strings.Contains(trimmedLine, "The volume state is: 0") {
				return fmt.Sprintf("Enabled (per-volume setting for %s)", volumeRoot), nil
			}
			if strings.Contains(trimmedLine, "The volume state is: 1") {
				return fmt.Sprintf("Disabled (per-volume setting for %s)", volumeRoot), nil
			}
		}
		return "", fmt.Errorf("failed to parse fsutil output for volume %s. Output: %s", volumeRoot, outputStr)
	default:
		return fmt.Sprintf("Unknown global 8.3 name setting (value: %d)", ntfsDisable8dot3NameCreation), nil
	}
}
