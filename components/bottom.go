package components

import (
	"fmt"
	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
	"log"
	"strconv"
	"time"

	_ "image/png"

	"../backend"
)

var AnimationBottomComponent *BottomComponent

type BottomComponent struct {
	// root container
	Container             *fyne.Container
	PreviewImageContainer *fyne.Container
	PreviewImage          *canvas.Image
	Player                *Player
}

const (
	defaultFps = 12
	maxFps     = 24
)

type Player struct {
	Ticker    *time.Ticker
	Fps       int
	IsPlaying bool
	sleepTime time.Duration
	frameNum  int
}

func (f *Player) On() {
	for true {
		if !f.IsPlaying {
			time.Sleep(time.Duration(1) * time.Second)
			continue
		}

		f.frameNum++
		if f.frameNum >= len(backend.Backend.Frames) {
			f.frameNum = 0
		}
		fileName := backend.Backend.Frames[f.frameNum].Filename
		AnimationBottomComponent.PreviewImage = canvas.NewImageFromFile(fileName)
		AnimationBottomComponent.PreviewImageContainer.Objects[0] = AnimationBottomComponent.PreviewImage
		AnimationBottomComponent.PreviewImage.SetMinSize(fyne.NewSize(webcamImageWidth, webcamImageHeight))
		AnimationBottomComponent.PreviewImageContainer.Refresh()
		time.Sleep(f.sleepTime)
	}
}

func (f *Player) Start() {
	f.IsPlaying = true
}

func (f *Player) Stop() {
	f.IsPlaying = false
}

func (f *Player) Rewind() {
	f.frameNum = 0
}

func (f *Player) GenerateVideo() {
	log.Printf("todo: generate video")
}

func (f *Player) SetFPS(fps int) {
	f.Fps = fps
	t := 1000000 / fps
	f.sleepTime = time.Duration(t) * time.Microsecond
}

func NewPlayer() *Player {
	player := Player{
	}
	player.SetFPS(defaultFps)
	return &player
}

func NewBottomComponent() *BottomComponent {
	previewImage := canvas.NewImageFromFile(`D:\Luke\Downloads\test.png`)
	previewImage.SetMinSize(fyne.NewSize(webcamImageWidth, webcamImageHeight))
	previewImageContainer := fyne.NewContainerWithLayout(layout.NewMaxLayout(), previewImage)

	component := BottomComponent{
		PreviewImageContainer: previewImageContainer,
		PreviewImage:          previewImage,
		Player:                NewPlayer(),
	}

	toolbarLayout := layout.NewVBoxLayout()

	fpsLabel := widget.NewLabel("FPS")

	fpsSlider := widget.NewSlider(1, 60)
	fpsSlider.Refresh()
	fpsSlider.OnChanged = func(value float64) {
		fpsLabel.Text = fmt.Sprintf("%.0f", value)
		fpsLabel.Refresh()
	}
	fpsSlider.Value = defaultFps
	fpsSlider.Orientation = widget.Horizontal
	fpsSliderContainer := fyne.NewContainerWithLayout(layout.NewCenterLayout(), fpsSlider)
	fpsSliderContainer.Resize(fyne.NewSize(60, 240))

	fpsSelectEntry := widget.NewSelect([]string{"1", "6", "12", "18", "24"}, func(choice string) {
		log.Printf("use selected choice %s", choice)
		fps, err := strconv.Atoi(choice)
		if err != nil {
			log.Printf("failed Atoi(%s) due to: %s", choice, err.Error())
			return
		}
		component.Player.SetFPS(fps)
	})
	fpsSelectEntry.PlaceHolder = "12"

	fpsLabelContainer := fyne.NewContainerWithLayout(layout.NewCenterLayout(), fpsLabel)

	playButton := widget.NewButton("Play", func() {
		log.Printf("play button clicked")
		component.Player.Start()
	})

	stopButton := widget.NewButton("Pause", func() {
		log.Printf("pause button clicked")
		component.Player.Stop()
	})

	rewindButton := widget.NewButton("Rewind", func() {
		log.Printf("rewind button clicked")
		component.Player.Rewind()
	})

	generateVideoButton := widget.NewButton("Generate", func() {
		log.Printf("generate button clicked")
		component.Player.GenerateVideo()
	})
	toolbarContainer := fyne.NewContainerWithLayout(toolbarLayout, playButton, stopButton, rewindButton, fpsSelectEntry, fpsLabelContainer, generateVideoButton)

	rootLayout := layout.NewHBoxLayout()

	rootContainer := fyne.NewContainerWithLayout(rootLayout, toolbarContainer, previewImageContainer)

	component.Container = rootContainer

	go component.Player.On()

	return &component
}
