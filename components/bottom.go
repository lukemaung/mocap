package components

import (
	"fmt"
	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
	"gocv.io/x/gocv"
	"image/color"
	"log"
	"strconv"
	"time"

	_ "image/png"

	"../backend"
	"../config"
	"../util"
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
		AnimationBottomComponent.PreviewImage.SetMinSize(fyne.NewSize(config.WebcamDisplayWidth, config.WebcamDisplayHeight))
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
	baseDir, err := util.GetMocapBaseDir()
	if err != nil {
		log.Printf("error getting basedir: %s", err.Error())
		return
	}
	timestampSuffix := time.Now().Format("2006-01-02-15:04")
	absPath := fmt.Sprintf(`%s\%s\%s-%s.mp4`, baseDir, backend.Backend.Name, backend.Backend.Name, timestampSuffix)
	vw, err := gocv.VideoWriterFile(absPath, "mp4v", float64(f.Fps),config.WebcamCaptureWidth, config.WebcamCaptureHeight, true)
	if err != nil {
		log.Printf("error getting video writer: %s", err.Error())
		return
	}
	log.Printf("video file=%s", absPath)
	defer vw.Close()

	appWindow := *MocapApp.Window
	progressBar := dialog.NewProgress("Generating Video", "Please wait while generating video.", appWindow)
	for idx, frame := range backend.Backend.Frames {
		srcMat := gocv.IMRead(frame.Filename, gocv.IMReadColor)
		if srcMat.Empty() {
			log.Printf("couldn't read frame from %s", frame.Filename)
			continue
		}
		log.Printf("writing %dx%d frame %d", srcMat.Size()[1], srcMat.Size()[0], idx)
		err = vw.Write(srcMat)
		if err != nil {
			log.Printf("error writing frame: %s", err.Error())
		}
		err = srcMat.Close()
		if err != nil {
			log.Printf("error closing frame: %s", err.Error())
		}
		progressBar.SetValue(float64(idx)/float64(len(backend.Backend.Frames)))
	}
	progressBar.SetValue(1.0)
	progressBar.Hide()
	DisplayUserTip(fmt.Sprintf("Video file is saved at:\n%s", absPath))
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
	previewImage := canvas.NewRectangle(color.White)
	previewImage.SetMinSize(fyne.NewSize(config.WebcamDisplayWidth, config.WebcamDisplayHeight))
	previewImageContainer := fyne.NewContainerWithLayout(layout.NewMaxLayout(), previewImage)

	component := BottomComponent{
		PreviewImageContainer: previewImageContainer,
		PreviewImage:          nil,
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
		if backend.Backend.Name == "" {
			DisplayUserTip("Please create/open a project first.")
			return
		}
		log.Printf("play button clicked")
		component.Player.Start()
	})

	stopButton := widget.NewButton("Pause", func() {
		if backend.Backend.Name == "" {
			DisplayUserTip("Please create/open a project first.")
			return
		}
		log.Printf("pause button clicked")
		component.Player.Stop()
	})

	rewindButton := widget.NewButton("Rewind", func() {
		if backend.Backend.Name == "" {
			DisplayUserTip("Please create/open a project first.")
			return
		}
		log.Printf("rewind button clicked")
		component.Player.Rewind()
	})

	generateVideoButton := widget.NewButton("Generate", func() {
		if backend.Backend.Name == "" {
			DisplayUserTip("Please create/open a project first.")
			return
		}
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
