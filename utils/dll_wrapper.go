package utils

import (
	"go.uber.org/zap"
)

func ScanDll(logger *zap.SugaredLogger, dllPath string) (*DllInfo, error) {
	// Use the existing DLL parsing logic from dll.go
	info, err := parseDll(logger, dllPath)
	if err != nil {
		return nil, err
	}

	return info, nil
}
