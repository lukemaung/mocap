package components

import (
	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
	"image/color"
	"log"

	"../backend"
)

const (
	thumbnailWidth = 80
	thumbnailHeight = 45
	thumbnailCount = 16
)

var AnimationFilmStripComponent *FilmStrip

type FilmStrip struct {
	Container     *fyne.Container
	FrameContainer *fyne.Container
	VisibleFrames []fyne.CanvasObject
	ViewSize int
	ViewOffset int
	Cursor int
}

func (f *FilmStrip) Left() {
	f.ViewOffset--
	if f.ViewOffset < 0 {
		f.ViewOffset = 0
	}
}

func (f *FilmStrip) Right() {
	maxAllowedLeftOffset := len(backend.AnimationBackend.Frames) - thumbnailCount
	if maxAllowedLeftOffset < 0 {
		maxAllowedLeftOffset = 0
	}
	if f.ViewOffset < maxAllowedLeftOffset {
		f.ViewOffset++
	}
}

func (f *FilmStrip) Tail() {
	maxAllowedLeftOffset := len(backend.AnimationBackend.Frames) - thumbnailCount
	if maxAllowedLeftOffset < 0 {
		maxAllowedLeftOffset = 0
	}
	f.ViewOffset = maxAllowedLeftOffset
}

func (f *FilmStrip) SyncToBackend() {
	log.Printf("syncing with backend")
	leftIndex := f.ViewOffset
	rightIndex := len(backend.AnimationBackend.Frames) - 1
	calculatedSize := f.ViewOffset+f.ViewSize
	if calculatedSize < rightIndex {
		rightIndex = calculatedSize
	}
	log.Printf("leftIndex=%d, calculatedSize=%d, rightIndex=%d", leftIndex, calculatedSize, rightIndex)
	if rightIndex > 0 {
		log.Printf("will load visible frames from backend frames")
		for idx, frame := range backend.AnimationBackend.Frames[leftIndex:rightIndex] {
			image := canvas.NewImageFromFile(frame.Filename)
			if image == nil {
				log.Printf("error loading file %s due to: %s", frame.Filename)
				continue
			}

			log.Printf("image.image=%#v", image.Image)

			image.SetMinSize(fyne.Size{
				Width:  thumbnailWidth,
				Height: thumbnailHeight,
			})
			f.VisibleFrames[idx] = image
		}
	}

	if rightIndex < thumbnailCount {
		log.Printf("will insert white rects to unused visible frames")
		startIndex := rightIndex
		if startIndex < 0 {
			startIndex = 0
		}
		for idx := startIndex; idx < thumbnailCount; idx ++ {
			if idx >= 0 {
				rect := canvas.NewRectangle(color.White)
				rect.SetMinSize(fyne.Size{
					Width:  thumbnailWidth,
					Height: thumbnailHeight,
				})
				f.VisibleFrames[idx] = rect
			}
		}
	}

	f.FrameContainer.Objects = f.VisibleFrames
	f.FrameContainer.Refresh()
}

func NewFilmStripComponent() *FilmStrip {
	frames := make([]fyne.CanvasObject, 0)
	for i := 0; i < thumbnailCount; i++ {
		rect := canvas.NewRectangle(color.White)
		rect.SetMinSize(fyne.Size{
			Width:  thumbnailWidth,
			Height: thumbnailHeight,
		})
		frames = append(frames, rect)
	}

	filmstrip := FilmStrip{
		VisibleFrames: frames,
		ViewSize: thumbnailCount,
	}

	leftButton := widget.NewButton("<", func() {
		filmstrip.Left()
		filmstrip.SyncToBackend()
	})

	rightButton := widget.NewButton(">", func() {
		filmstrip.Right()
		filmstrip.SyncToBackend()
	})

	frameContainer := fyne.NewContainerWithLayout(layout.NewHBoxLayout(), frames...)
	filmstrip.FrameContainer = frameContainer
	rootLayout := layout.NewHBoxLayout()
	items := make([]fyne.CanvasObject, 0)
	items = append(items, leftButton)
	items = append(items, frameContainer)
	items = append(items, rightButton)
	rootLayout.Layout(items, fyne.NewSize(1280,360))
	rootContainer := fyne.NewContainerWithLayout(rootLayout, items...)
	filmstrip.Container = rootContainer

	return &filmstrip
}