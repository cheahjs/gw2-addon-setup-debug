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
	"github.com/cheahjs/gw2-addon-setup-debug/utils"
	"go.uber.org/zap"
)

type Monitor struct {
	logger              *zap.SugaredLogger
	continueButton      widget.Clickable
	confirmButton       widget.Clickable
	skipButton          widget.Clickable
	monitoringStarted   bool
	userConfirmed       bool
	userSkipped         bool
	status              string
	errorMessage        string
	processInfo         *utils.ProcessInfo
	startMonitoringTime time.Time
	gw2ProcessFound     bool
	window              *app.Window
	findProcessFunc     func() (*utils.ProcessInfo, error)
	tempProcessInfo     *utils.ProcessInfo
	ticker              *time.Ticker
}

func NewMonitor(logger *zap.SugaredLogger, window *app.Window) *Monitor {
	// Check every second for GW2 process
	return &Monitor{
		logger:         logger,
		continueButton: widget.Clickable{},
		confirmButton:  widget.Clickable{},
		skipButton:     widget.Clickable{},
		window:         window,
		ticker:         time.NewTicker(1 * time.Second),
	}
}

func (m *Monitor) Run(gtx layout.Context, e app.FrameEvent, findProcessFunc func() (*utils.ProcessInfo, error)) bool {
	// Store findProcessFunc for later use
	if m.findProcessFunc == nil {
		m.findProcessFunc = findProcessFunc
	}

	th := material.NewTheme()

	// Start monitoring if it hasn't started yet
	if !m.monitoringStarted {
		m.monitoringStarted = true
		m.startMonitoringTime = time.Now()
		go m.monitorProcess(m.ticker)
	}

	// User clicked confirm - get latest process info
	if m.confirmButton.Clicked(gtx) {
		if latestInfo, err := m.findProcessFunc(); err == nil {
			m.processInfo = latestInfo
			m.userConfirmed = true
		} else {
			m.errorMessage = "Failed to get latest process information"
		}
	}

	// User clicked skip
	if m.skipButton.Clicked(gtx) {
		m.userSkipped = true
		m.userConfirmed = false
		m.processInfo = nil
		m.status = "Process detection skipped"
		m.logger.Infow("User skipped GW2 process detection")
	}

	// User confirmed/skipped and continue button clicked
	if (m.userConfirmed || m.userSkipped) && m.continueButton.Clicked(gtx) {
		m.ticker.Stop()
		return true
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
			instructions := "Please launch Guild Wars 2 and navigate to the character selection screen. If the game crashes, then keep the crash report open and click the button below."
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
			if !m.userConfirmed && !m.userSkipped {
				flex := layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}

				children := []layout.FlexChild{
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, &m.skipButton, "Skip this step")
						return btn.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: 10}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if m.gw2ProcessFound {
							btn := material.Button(th, &m.confirmButton, "I'm at the character selection screen")
							return btn.Layout(gtx)
						}
						return layout.Dimensions{}
					}),
				}

				return flex.Layout(gtx, children...)
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
			} else if m.userSkipped {
				paragraph := material.Body1(th, "Guild Wars 2 process detection has been skipped.")
				paragraph.Alignment = text.Middle
				return paragraph.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if m.userConfirmed || m.userSkipped {
				btn := material.Button(th, &m.continueButton, "Continue")
				return btn.Layout(gtx)
			}
			return layout.Dimensions{}
		}))

	return false
}

func (m *Monitor) monitorProcess(ticker *time.Ticker) {
	for {
		select {
		case <-ticker.C:
			processInfo, err := m.findProcessFunc()

			// If we can't find the process or there's an error
			if err != nil || processInfo == nil {
				if m.gw2ProcessFound {
					// Process was found before but now it's gone
					m.gw2ProcessFound = false
					m.userConfirmed = false
					m.processInfo = nil
					m.tempProcessInfo = nil
					m.status = "Guild Wars 2 process no longer detected. Please launch the game."
					m.logger.Infow("GW2 process disappeared")
				} else {
					elapsedTime := time.Since(m.startMonitoringTime)
					m.status = fmt.Sprintf("Looking for Guild Wars 2 process... (%.0f seconds)", elapsedTime.Seconds())
				}
				m.window.Invalidate()
				if err != nil {
					m.logger.Debugw("Failed to find GW2 process", "error", err)
				}
				continue
			}

			// Process found
			m.tempProcessInfo = processInfo
			if !m.gw2ProcessFound {
				m.gw2ProcessFound = true
				m.status = "Guild Wars 2 process found!"
				m.logger.Infow("Found GW2 process",
					"pid", processInfo.ProcessID,
					"path", processInfo.ExecutablePath,
					"workingDir", processInfo.WorkingDir,
					"modules", len(processInfo.LoadedModules))
			}
			m.window.Invalidate()
		}
	}
}

func (m *Monitor) GetProcessInfo() *utils.ProcessInfo {
	// If the user skipped or we haven't confirmed yet, return nil
	if !m.userConfirmed || m.processInfo == nil {
		return nil
	}
	return m.processInfo
}
