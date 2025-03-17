package process_modules

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"go.uber.org/zap"
)

type ProcessInfo struct {
	ProcessID      int
	ExecutablePath string
	LoadedModules  []string
	WorkingDir     string
	CommandLine    string
	Timestamp      time.Time
}

type Monitor struct {
	logger              *zap.SugaredLogger
	continueButton      widget.Clickable
	confirmButton       widget.Clickable
	monitoringStarted   bool
	monitoringDone      bool
	userConfirmed       bool
	status              string
	errorMessage        string
	processInfo         *ProcessInfo
	startMonitoringTime time.Time
	gw2ProcessFound     bool
}

func NewMonitor(logger *zap.SugaredLogger) *Monitor {
	return &Monitor{
		logger:         logger,
		continueButton: widget.Clickable{},
		confirmButton:  widget.Clickable{},
	}
}

func (m *Monitor) Run(gtx layout.Context, e app.FrameEvent, findProcessFunc func() (*ProcessInfo, error)) bool {
	th := material.NewTheme()

	// Start monitoring if it hasn't started yet
	if !m.monitoringStarted {
		m.monitoringStarted = true
		m.startMonitoringTime = time.Now()
		go m.monitorProcess(findProcessFunc)
	}

	// User confirmed and continue button clicked
	if m.userConfirmed && m.continueButton.Clicked(gtx) {
		return true
	}

	// User clicked confirm
	if m.confirmButton.Clicked(gtx) {
		m.userConfirmed = true
	}

	layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			paragraph := material.Body1(th, "Guild Wars 2 Process Detection")
			paragraph.Alignment = text.Middle
			return paragraph.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			instructions := "Please launch Guild Wars 2 and navigate to the character selection screen."
			if m.gw2ProcessFound {
				instructions = "Guild Wars 2 process detected! Please confirm when you've reached the character selection screen."
			}
			paragraph := material.Body1(th, instructions)
			return paragraph.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			paragraph := material.Body1(th, m.status)
			return paragraph.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 10}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if m.errorMessage != "" {
				paragraph := material.Body1(th, m.errorMessage)
				paragraph.Color = color.NRGBA{R: 200, A: 255} // Red color for error
				return paragraph.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if m.gw2ProcessFound && !m.userConfirmed {
				btn := material.Button(th, &m.confirmButton, "I'm at the character selection screen")
				return btn.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if m.userConfirmed {
				var moduleText strings.Builder
				moduleText.WriteString("Process information captured:\n\n")
				moduleText.WriteString(fmt.Sprintf("Executable: %s\n", m.processInfo.ExecutablePath))
				moduleText.WriteString(fmt.Sprintf("Process ID: %d\n", m.processInfo.ProcessID))
				moduleText.WriteString(fmt.Sprintf("Working Directory: %s\n", m.processInfo.WorkingDir))
				moduleText.WriteString(fmt.Sprintf("Command Line: %s\n\n", m.processInfo.CommandLine))
				moduleText.WriteString(fmt.Sprintf("Number of loaded modules: %d\n", len(m.processInfo.LoadedModules)))

				paragraph := material.Body1(th, moduleText.String())
				paragraph.Alignment = text.Start
				return paragraph.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if m.userConfirmed {
				btn := material.Button(th, &m.continueButton, "Continue")
				return btn.Layout(gtx)
			}
			return layout.Dimensions{}
		}))

	// Pass the drawing operations to the GPU.
	e.Frame(gtx.Ops)

	return false
}

func (m *Monitor) monitorProcess(findProcessFunc func() (*ProcessInfo, error)) {
	// Check every second for GW2 process
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			elapsedTime := time.Since(m.startMonitoringTime)
			m.status = fmt.Sprintf("Looking for Guild Wars 2 process... (%.0f seconds)", elapsedTime.Seconds())

			processInfo, err := findProcessFunc()
			if err != nil {
				m.logger.Debugw("Failed to find GW2 process", "error", err)
				continue
			}

			if processInfo != nil {
				m.processInfo = processInfo
				m.gw2ProcessFound = true
				m.status = "Guild Wars 2 process found!"
				m.logger.Infow("Found GW2 process",
					"pid", processInfo.ProcessID,
					"path", processInfo.ExecutablePath,
					"workingDir", processInfo.WorkingDir,
					"modules", len(processInfo.LoadedModules))
				return
			}
		}
	}
}

func (m *Monitor) GetProcessInfo() *ProcessInfo {
	return m.processInfo
}
