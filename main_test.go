//go:build windows
// +build windows

package main

import (
	"flag"
	"os"
	"testing"
)

// TestMainFlags simulates command-line arguments and checks if flags are parsed correctly.
func TestMainFlags(t *testing.T) {
	// Store original os.Args and restore it after the test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	testCases := []struct {
		name             string
		args             []string
		expectedDetect   bool
		expectedGw2Path  string
		expectedReportPath string
	}{
		{
			name:             "No flags",
			args:             []string{"cmd"},
			expectedDetect:   false,
			expectedGw2Path:  "",
			expectedReportPath: "",
		},
		{
			name:             "Detect process flag",
			args:             []string{"cmd", "--detect-process"},
			expectedDetect:   true,
			expectedGw2Path:  "",
			expectedReportPath: "",
		},
		{
			name:             "GW2 path flag",
			args:             []string{"cmd", "--gw2-path", "/path/to/gw2"},
			expectedDetect:   false,
			expectedGw2Path:  "/path/to/gw2",
			expectedReportPath: "",
		},
		{
			name:             "Report output path flag",
			args:             []string{"cmd", "--report-output-path", "/path/to/report.txt"},
			expectedDetect:   false,
			expectedGw2Path:  "",
			expectedReportPath: "/path/to/report.txt",
		},
		{
			name:             "All flags",
			args:             []string{"cmd", "--detect-process", "--gw2-path", "/path/to/gw2", "--report-output-path", "/path/to/report.txt"},
			expectedDetect:   true,
			expectedGw2Path:  "/path/to/gw2",
			expectedReportPath: "/path/to/report.txt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset flags for each test case
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			detectProcess = flag.Bool("detect-process", false, "")
			gw2Path = flag.String("gw2-path", "", "")
			reportOutputPath = flag.String("report-output-path", "", "")

			os.Args = tc.args
			flag.Parse()

			if *detectProcess != tc.expectedDetect {
				t.Errorf("Expected detectProcess to be %v, got %v", tc.expectedDetect, *detectProcess)
			}
			if *gw2Path != tc.expectedGw2Path {
				t.Errorf("Expected gw2Path to be %s, got %s", tc.expectedGw2Path, *gw2Path)
			}
			if *reportOutputPath != tc.expectedReportPath {
				t.Errorf("Expected reportOutputPath to be %s, got %s", tc.expectedReportPath, *reportOutputPath)
			}
		})
	}
}

// Note: Testing the full non-interactive workflow would require more extensive mocking
// or an integration test setup, as it involves file system operations, process interactions,
// and registry access. This test focuses on the flag parsing aspect.
//
// To test the non-interactive mode logic itself (the if block in main()),
// you would typically refactor that logic into a separate function that can be called
// with mocked dependencies (logger, utility functions, etc.).
// For this example, we're keeping it simple and focusing on flags.
