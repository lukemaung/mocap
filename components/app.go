package components

import (
	"fyne.io/fyne"
	"fyne.io/fyne/layout"
	"gocv.io/x/gocv"
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

func NewMocapAppWindow(mocapApp fyne.App, webcam *gocv.VideoCapture) *MocapAppWindow {
	mocapAppWindow := MocapAppWindow{}

	appWindow := mocapApp.NewWindow("Mocap Animation")

	AnimationTopComponent = NewTopComponent(webcam)
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
