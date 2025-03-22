package admin

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/cheahjs/gw2-addon-setup-debug/utils"
	"go.uber.org/zap"
	"golang.org/x/sys/windows"
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
		cwd, err := os.Getwd()
		if err != nil {
			ac.logger.Errorw("Failed to get current working directory", "error", err)
			ac.errorMessage = fmt.Sprintf("Failed to get current working directory: %s", err.Error())
			ac.elevationStarted = false
			return false, false
		}
		args := strings.Join(os.Args[1:], " ")

		verbPtr, err := syscall.UTF16PtrFromString("runas")
		if err != nil {
			ac.logger.Errorw("Failed to convert verb to UTF-16", "error", err)
			ac.errorMessage = fmt.Sprintf("Failed to convert verb to UTF-16: %s", err.Error())
			ac.elevationStarted = false
			return false, false
		}
		exePtr, err := syscall.UTF16PtrFromString(exe)
		if err != nil {
			ac.logger.Errorw("Failed to convert executable path to UTF-16", "error", err)
			ac.errorMessage = fmt.Sprintf("Failed to convert executable path to UTF-16: %s", err.Error())
			ac.elevationStarted = false
			return false, false
		}
		cwdPtr, err := syscall.UTF16PtrFromString(cwd)
		if err != nil {
			ac.logger.Errorw("Failed to convert current working directory to UTF-16", "error", err)
			ac.errorMessage = fmt.Sprintf("Failed to convert current working directory to UTF-16: %s", err.Error())
			ac.elevationStarted = false
			return false, false
		}
		argPtr, err := syscall.UTF16PtrFromString(args)
		if err != nil {
			ac.logger.Errorw("Failed to convert arguments to UTF-16", "error", err)
			ac.errorMessage = fmt.Sprintf("Failed to convert arguments to UTF-16: %s", err.Error())
			ac.elevationStarted = false
			return false, false
		}

		// Run the elevated process
		err = windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, 0x1)
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
