package registry_check

import (
	"fmt"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"go.uber.org/zap"
	"golang.org/x/sys/windows/registry"
)

type RegistryInfo struct {
	SafeDllSearchMode *uint32
	KnownDlls         map[string]string
	Gw2RegistryPath   *string
}

type RegistryChecker struct {
	logger         *zap.SugaredLogger
	continueButton widget.Clickable
	registryInfo   *RegistryInfo
	status         string
	checkStarted   bool
	checkDone      bool
	window         *app.Window
}

func NewRegistryChecker(logger *zap.SugaredLogger, window *app.Window) *RegistryChecker {
	return &RegistryChecker{
		logger:         logger,
		continueButton: widget.Clickable{},
		window:         window,
	}
}

func (r *RegistryChecker) Run(gtx layout.Context, e app.FrameEvent) bool {
	th := material.NewTheme()

	if !r.checkStarted {
		r.checkStarted = true
		go r.checkRegistry()
	}

	if r.continueButton.Clicked(gtx) && r.checkDone {
		return true
	}

	layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			paragraph := material.Body1(th, "Checking Windows Registry Settings")
			paragraph.Alignment = text.Middle
			return paragraph.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			paragraph := material.Body1(th, r.status)
			return paragraph.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if r.checkDone && r.registryInfo != nil {
				var safeDllMode string
				if r.registryInfo.SafeDllSearchMode == nil {
					safeDllMode = "Not found"
				} else {
					safeDllMode = fmt.Sprintf("%d", *r.registryInfo.SafeDllSearchMode)
				}
				paragraph := material.Body1(th, fmt.Sprintf("SafeDllSearchMode: %s", safeDllMode))
				return paragraph.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if r.checkDone && r.registryInfo != nil {
				paragraph := material.Body1(th, fmt.Sprintf("Found %d KnownDLLs", len(r.registryInfo.KnownDlls)))
				return paragraph.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if r.checkDone && r.registryInfo != nil && r.registryInfo.Gw2RegistryPath != nil {
				paragraph := material.Body1(th, fmt.Sprintf("GW2 Registry Path: %s", *r.registryInfo.Gw2RegistryPath))
				return paragraph.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if r.checkDone {
				btn := material.Button(th, &r.continueButton, "Continue")
				return btn.Layout(gtx)
			}
			return layout.Dimensions{}
		}))

	return false
}

func (r *RegistryChecker) checkRegistry() {
	r.status = "Checking registry keys..."
	r.window.Invalidate()

	info := &RegistryInfo{
		KnownDlls: make(map[string]string),
	}

	// Check SafeDllSearchMode
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `System\CurrentControlSet\Control\Session Manager`, registry.READ)
	if err != nil {
		r.logger.Errorw("Failed to open Session Manager key", "error", err)
	} else {
		defer k.Close()
		if val, _, err := k.GetIntegerValue("SafeDllSearchMode"); err == nil {
			uval := uint32(val)
			info.SafeDllSearchMode = &uval
		} else {
			r.logger.Infow("SafeDllSearchMode not found", "error", err)
		}
	}

	// Check KnownDLLs
	k, err = registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\KnownDLLs`, registry.READ)
	if err != nil {
		r.logger.Errorw("Failed to open KnownDLLs key", "error", err)
	} else {
		defer k.Close()
		valueNames, err := k.ReadValueNames(0)
		if err != nil {
			r.logger.Errorw("Failed to read KnownDLLs values", "error", err)
		} else {
			for _, name := range valueNames {
				if val, _, err := k.GetStringValue(name); err == nil {
					info.KnownDlls[name] = val
				}
			}
		}
	}

	// Check GW2 Registry Path
	k, err = registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\ArenaNet\Guild Wars 2`, registry.READ)
	if err != nil {
		r.logger.Infow("Failed to open GW2 registry key", "error", err)
	} else {
		defer k.Close()
		if val, _, err := k.GetStringValue("Path"); err == nil {
			info.Gw2RegistryPath = &val
			r.logger.Infow("Found GW2 registry path", "path", val)
		} else {
			r.logger.Infow("GW2 Path not found in registry", "error", err)
		}
	}

	r.registryInfo = info
	r.status = "Registry check complete"
	r.checkDone = true
	r.window.Invalidate()
}

func (r *RegistryChecker) GetRegistryInfo() *RegistryInfo {
	return r.registryInfo
}
