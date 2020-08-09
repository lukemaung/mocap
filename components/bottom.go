package components

import (
	"fmt"
	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
	"log"
	"strconv"
)

var AnimationBottomComponent *BottomComponent

type BottomComponent struct {
	// root container
	Container *fyne.Container
	PreviewImage *canvas.Image
}

const (
	defaultFps = 12
)

func NewBottomComponent() *BottomComponent {
	previewImage := canvas.NewImageFromFile(`D:\Luke\Downloads\test.png`)
	previewImage.SetMinSize(fyne.NewSize(webcamImageWidth, webcamImageHeight))

	component := BottomComponent{
		PreviewImage: previewImage,
	}

	toolbarLayout := layout.NewVBoxLayout()

	fpsLabel := widget.NewLabel(strconv.Itoa(defaultFps))


	fpsSlider := widget.NewSlider(1, 60)
	fpsSlider.Refresh()
	fpsSlider.OnChanged = func(value float64) {
		fpsLabel.Text = fmt.Sprintf("%.0f", value)
		fpsLabel.Refresh()
	}
	fpsSlider.Value = defaultFps
	fpsSlider.Orientation = widget.Vertical
	fpsSliderContainer := fyne.NewContainerWithLayout(layout.NewGridLayout(1), fpsSlider)
	fpsSliderContainer.Resize(fyne.NewSize(60,240))

	playButton := widget.NewButton("Play", func() {
		log.Printf("play button clicked")
	})

	stopButton := widget.NewButton("Stop", func() {
		log.Printf("stop button clicked")
	})

	toolbarContainer := fyne.NewContainerWithLayout(toolbarLayout, playButton, stopButton, fpsSliderContainer, fpsLabel)

	rootLayout := layout.NewHBoxLayout()

	rootContainer := fyne.NewContainerWithLayout(rootLayout, toolbarContainer, previewImage)

	component.Container = rootContainer

	return &component
}
