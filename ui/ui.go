package ui

import (
	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"github.com/cheahjs/gw2-addon-setup-debug/ui/start"
	"go.uber.org/zap"
)

type UI struct {
	Logger *zap.SugaredLogger

	startMenu *start.Menu

	currentState uiState
}

func NewUI(logger *zap.SugaredLogger) *UI {
	return &UI{
		Logger:       logger,
		startMenu:    start.NewMenu(),
		currentState: startMenuState,
	}
}

func (ui *UI) Run(w *app.Window) error {
	var ops op.Ops
	for {
		switch e := w.NextEvent().(type) {
		case system.DestroyEvent:
			return e.Err
		case system.FrameEvent:
			// This graphics context is used for managing the rendering state.
			gtx := layout.NewContext(&ops, e)

			switch ui.currentState {
			case startMenuState:
				if ui.startMenu.Run(gtx, e) {
					ui.currentState = selectDirectoryState
				}
			case selectDirectoryState:
				break
			}
		}
	}
}
