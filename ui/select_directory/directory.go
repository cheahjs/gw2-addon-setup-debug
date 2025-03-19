package select_directory

import (
	"image/color"
	"os"
	"path"
	"path/filepath"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/explorer"
	"go.uber.org/zap"
)

type Directory struct {
	logger            *zap.SugaredLogger
	directory         string
	selectButton      widget.Clickable
	continueButton    widget.Clickable
	errorMessage      string
	isValid           bool
	fileExplorer      *explorer.Explorer
	includeDirListing widget.Bool
	includeLogs       widget.Bool
	hasArcdpsLogs     bool
}

func NewDirectory(logger *zap.SugaredLogger) *Directory {
	return &Directory{
		logger:            logger,
		selectButton:      widget.Clickable{},
		continueButton:    widget.Clickable{},
		includeDirListing: widget.Bool{Value: true}, // Include directory listing by default
		includeLogs:       widget.Bool{Value: true}, // Include logs by default if they exist
	}
}

func (d *Directory) Run(w *app.Window, gtx layout.Context, e app.FrameEvent) (bool, string, bool, bool) {
	th := material.NewTheme()

	if d.fileExplorer == nil {
		d.fileExplorer = explorer.NewExplorer(w)
	}

	// Handle explorer events
	d.fileExplorer.ListenEvents(e)

	if d.selectButton.Clicked(gtx) {
		go func() {
			readCloser, err := d.fileExplorer.ChooseFile("*Gw2-64.exe")
			if err != nil {
				d.logger.Errorw("Failed to choose file", "error", err)
				return
			}
			defer readCloser.Close()

			// Coerce the readCloser to a os.File
			file, ok := readCloser.(*os.File)
			if !ok {
				d.logger.Errorw("Failed to coerce readCloser to os.File", "error", err)
				return
			}

			// Get the path of the file
			filePath := file.Name()
			dirPath := filepath.Dir(filePath)

			// Set the directory and validate
			d.directory = dirPath
			d.validateDirectory()

			// Trigger a re-render
			w.Invalidate()
		}()
	}

	// Continue button clicked and directory is valid
	if d.continueButton.Clicked(gtx) && d.isValid {
		return true, d.directory, d.includeDirListing.Value, d.hasArcdpsLogs && d.includeLogs.Value
	}

	layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			paragraph := material.Body1(th, "Please select your Guild Wars 2 executable (Gw2-64.exe).")
			paragraph.Alignment = text.Middle
			return paragraph.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th, &d.selectButton, "Select Gw2-64.exe")
			return btn.Layout(gtx)
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if d.directory != "" {
				paragraph := material.Body1(th, "Selected directory: "+d.directory)
				return paragraph.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(
			layout.Spacer{Height: 10}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if d.errorMessage != "" {
				paragraph := material.Body1(th, d.errorMessage)
				paragraph.Color = color.NRGBA{R: 200, A: 255} // Red color for error
				return paragraph.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(
			layout.Spacer{Height: 20}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if d.isValid {
				// Add checkbox for directory listing
				checkbox := material.CheckBox(th, &d.includeDirListing, "Include directory listing in report")
				return checkbox.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(
			layout.Spacer{Height: 10}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if d.isValid && d.hasArcdpsLogs {
				// Add checkbox for arcdps logs
				checkbox := material.CheckBox(th, &d.includeLogs, "Include arcdps logs in report")
				return checkbox.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(
			layout.Spacer{Height: 10}.Layout,
		),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if d.isValid {
				btn := material.Button(th, &d.continueButton, "Continue")
				return btn.Layout(gtx)
			}
			return layout.Dimensions{}
		}))

	return false, "", false, false
}

func (d *Directory) validateDirectory() {
	// Check if Gw2-64.exe exists
	_, err := os.Stat(path.Join(d.directory, "Gw2-64.exe"))
	if err != nil {
		d.logger.Errorw("Gw2-64.exe not found", "error", err)
		d.errorMessage = "Gw2-64.exe not found. Please select the correct Guild Wars 2 directory."
		d.isValid = false
		return
	}

	// Check if Gw2.dat exists
	_, err = os.Stat(path.Join(d.directory, "Gw2.dat"))
	if err != nil {
		d.logger.Errorw("Gw2.dat not found", "error", err)
		d.errorMessage = "Gw2.dat not found. Please select the correct Guild Wars 2 directory."
		d.isValid = false
		return
	}

	// Check for arcdps logs
	_, err1 := os.Stat(path.Join(d.directory, "addons/arcdps/arcdps.log"))
	_, err2 := os.Stat(path.Join(d.directory, "addons/arcdps/arcdps_lastcrash.log"))
	d.hasArcdpsLogs = err1 == nil || err2 == nil
	if d.hasArcdpsLogs {
		d.logger.Infow("Found arcdps logs", "path", d.directory)
	}

	d.errorMessage = ""
	d.isValid = true
	d.logger.Infow("Valid GW2 directory", "path", d.directory)
}
