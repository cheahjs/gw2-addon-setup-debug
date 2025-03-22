package ui

type uiState int

const (
	// Start menu
	startMenuState uiState = iota
	// Admin check screen
	adminCheckState
	// Select GW2 directory
	selectDirectoryState
	// Scan DLLs in directory
	scanDllsState
	// Monitor GW2 process
	processMonitorState
	// Check registry settings
	registryCheckState
	// Show results and generate report
	resultState
)
