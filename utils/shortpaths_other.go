//go:build !windows

package utils

// ShortPathsEnabled always returns false on non-Windows platforms.
func ShortPathsEnabled(string) (bool, error) {
	return false, nil
}
