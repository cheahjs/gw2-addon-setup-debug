package start

import (
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type Menu struct {
	startButton widget.Clickable
}

func NewMenu() *Menu {
	return &Menu{
		startButton: widget.Clickable{},
	}
}

func (sm *Menu) Run(gtx layout.Context, e system.FrameEvent) bool {
	th := material.NewTheme()

	clicked := sm.startButton.Clicked(gtx)

	layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			paragraph := material.Body1(th, "This tool will guide you step by step in debugging your addon setup for Guild Wars 2. It will also help you generate a report that you can share for other people to help you debug your setup.")
			paragraph.Alignment = text.Middle
			return paragraph.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 25}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			paragraph := material.Body1(th, "Click the button below to start.")
			paragraph.Alignment = text.Middle
			return paragraph.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 25}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th, &sm.startButton, "Start")
			return btn.Layout(gtx)
		}))

	// Pass the drawing operations to the GPU.
	e.Frame(gtx.Ops)

	return clicked
}
