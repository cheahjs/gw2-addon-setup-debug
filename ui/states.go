package ui

type uiState int

const (
	// Start menu
	startMenuState uiState = iota
	// Select GW2 directory
	selectDirectoryState
	// Scan DLLs in directory
	scanDllsState
	// Monitor GW2 process
	processMonitorState
	// Show results and generate report
	resultState
)
