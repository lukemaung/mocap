package components

import (
	"../backend"
	"../config"
	"fmt"
	"fyne.io/fyne"
	"fyne.io/fyne/layout"
)

/*

	MocapAppWindow
		TopComponent
			ProjectPanel
			ChromaPanel
			BackgroundPanel

*/

type MocapAppWindow struct {
	Window       *fyne.Window
	TopComponent *fyne.Container
	MidCompoment *fyne.Container
}


var MocapApp *MocapAppWindow

func UpdateMocapTitle() {
	appWindow := *MocapApp.Window
	projectName := backend.Backend.Name
	if projectName == "" {
		projectName = "<No Project>"
	}
	appWindow.SetTitle(fmt.Sprintf("%s - %s", config.MocapDir, projectName))
}

func NewMocapAppWindow(mocapApp fyne.App) *MocapAppWindow {
	mocapAppWindow := MocapAppWindow{}

	appWindow := mocapApp.NewWindow(config.MocapDir)

	AnimationTopComponent = NewTopComponent()
	AnimationFilmStripComponent = NewFilmStripComponent()
	AnimationBottomComponent = NewBottomComponent()

	rootLayout := layout.NewVBoxLayout()
	rootContainer := fyne.NewContainerWithLayout(rootLayout, AnimationTopComponent.Container, AnimationFilmStripComponent.Container, AnimationBottomComponent.Container)

	// start capturing
	go AnimationTopComponent.CaptureLoop()

	appWindow.SetContent(rootContainer)
	mocapAppWindow.Window = &appWindow

	return &mocapAppWindow
}
