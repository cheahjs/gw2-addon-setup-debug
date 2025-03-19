package admin

import (
	"fmt"
	"os"
	"os/exec"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/cheahjs/gw2-addon-setup-debug/utils"
	"go.uber.org/zap"
)

type AdminCheck struct {
	logger           *zap.SugaredLogger
	continueButton   widget.Clickable
	elevateButton    widget.Clickable
	isAdmin          bool
	adminCheckDone   bool
	errorMessage     string
	elevationStarted bool
}

func NewAdminCheck(logger *zap.SugaredLogger) *AdminCheck {
	return &AdminCheck{
		logger:         logger,
		continueButton: widget.Clickable{},
		elevateButton:  widget.Clickable{},
	}
}

func (ac *AdminCheck) Run(gtx layout.Context, e app.FrameEvent) (bool, bool) {
	th := material.NewTheme()

	// Check admin status if not done yet
	if !ac.adminCheckDone {
		isAdmin, err := utils.IsRunningAsAdmin()
		if err != nil {
			ac.logger.Errorw("Failed to check admin status", "error", err)
			ac.errorMessage = fmt.Sprintf("Failed to check administrator status: %s", err.Error())
		} else {
			ac.isAdmin = isAdmin
		}
		ac.adminCheckDone = true
	}

	// Handle button clicks
	if ac.continueButton.Clicked(gtx) {
		return true, false
	}

	if ac.elevateButton.Clicked(gtx) && !ac.elevationStarted {
		ac.elevationStarted = true
		// Get the current executable path
		exe, err := os.Executable()
		if err != nil {
			ac.logger.Errorw("Failed to get executable path", "error", err)
			ac.errorMessage = fmt.Sprintf("Failed to get executable path: %s", err.Error())
			ac.elevationStarted = false
			return false, false
		}

		// Prepare runas command
		cmd := exec.Command("powershell", "Start-Process", "-Verb", "RunAs", exe)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Run the elevated process
		err = cmd.Start()
		if err != nil {
			ac.logger.Errorw("Failed to start elevated process", "error", err)
			ac.errorMessage = fmt.Sprintf("Failed to start elevated process: %s", err.Error())
			ac.elevationStarted = false
			return false, false
		}

		// Return true and signal that we're elevating
		return true, true
	}

	layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			title := "Administrator Privileges"
			if ac.isAdmin {
				title = "Running as Administrator"
			}
			paragraph := material.Body1(th, title)
			paragraph.Alignment = text.Middle
			return paragraph.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			var message string
			if ac.isAdmin {
				message = "This application is running with administrator privileges."
			} else {
				message = "This application is not running with administrator privileges. Some features may not work correctly without administrator access, especially if you start Guild Wars 2 as administrator."
			}
			paragraph := material.Body1(th, message)
			paragraph.Alignment = text.Middle
			return paragraph.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if ac.errorMessage != "" {
				paragraph := material.Body1(th, ac.errorMessage)
				paragraph.Alignment = text.Middle
				return paragraph.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !ac.isAdmin {
				flex := layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}
				children := []layout.FlexChild{
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, &ac.continueButton, "Continue without admin")
						return btn.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: 10}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, &ac.elevateButton, "Restart as administrator")
						return btn.Layout(gtx)
					}),
				}
				return flex.Layout(gtx, children...)
			} else {
				btn := material.Button(th, &ac.continueButton, "Continue")
				return btn.Layout(gtx)
			}
		}))

	return false, false
}
