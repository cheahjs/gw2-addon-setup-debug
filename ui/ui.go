package ui

import (
	"gioui.org/app"
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
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			// This graphics context is used for managing the rendering state.
			gtx := app.NewContext(&ops, e)

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
