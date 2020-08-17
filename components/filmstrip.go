package components

import (
	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
	"image/color"
	"log"

	"../backend"
	"../config"
)

const (
	thumbnailWidth  = 80
	thumbnailHeight = 45
	thumbnailCount  = 16
)

var (
	AnimationFilmStripComponent *FilmStrip
	FirstTimeFrameSelect = true
)

type FilmStrip struct {
	Container      *fyne.Container
	FrameContainer *fyne.Container
	VisibleFrames  []fyne.CanvasObject

	ViewSize   int
	ViewOffset int
	Cursor     int // -1 indicates cursor location is unset
}

func (f *FilmStrip) Left() {
	f.ViewOffset--
	if f.ViewOffset < 0 {
		f.ViewOffset = 0
	}
	f.ResetCursor()
}

func (f *FilmStrip) ResetCursor() {
	f.Cursor = -1
}

func (f *FilmStrip) Right() {
	maxAllowedLeftOffset := len(backend.Backend.Frames) - thumbnailCount
	if maxAllowedLeftOffset < 0 {
		maxAllowedLeftOffset = 0
	}
	if f.ViewOffset < maxAllowedLeftOffset {
		f.ViewOffset++
	}
	f.ResetCursor()
}

func (f *FilmStrip) Tail() {
	maxAllowedLeftOffset := len(backend.Backend.Frames) - thumbnailCount
	if maxAllowedLeftOffset < 0 {
		maxAllowedLeftOffset = 0
	}
	f.ViewOffset = maxAllowedLeftOffset
	f.ResetCursor()
}

func (f *FilmStrip) ExclusiveSelectFrame(fileName string) {
	//log.Printf("will look for backend frame with filename %s", fileName)
	for idx, frame := range backend.Backend.Frames {
		pinnedIdx := idx
		if fileName == frame.ThumbnailFilename {
			log.Printf("set cursor to backend frame %d, fileName: %s", idx, fileName)
			f.Cursor = pinnedIdx
			break
		}
	}
	for idx, frame := range f.VisibleFrames {
		switch v := frame.(type) {
		case *HotImage:
			if v.image.File == fileName {
				log.Printf("exclusively selecting visible frame %d", idx)
				v.Selected = true
			} else {
				v.Selected = false
			}
		default:
		}
	}

	if f.Cursor >= 0 {
		fileName := backend.Backend.Frames[f.Cursor].Filename
		AnimationBottomComponent.PreviewImage = canvas.NewImageFromFile(fileName)
		AnimationBottomComponent.PreviewImageContainer.Objects[0] = AnimationBottomComponent.PreviewImage
		AnimationBottomComponent.PreviewImage.SetMinSize(fyne.NewSize(config.WebcamDisplayWidth, config.WebcamDisplayHeight))
		AnimationBottomComponent.PreviewImageContainer.Refresh()

		if FirstTimeFrameSelect {
			DisplayUserTip("You can insert a new frame at this location by clicking Snapshot.\n You can remove this frame by right clicking on it.")
			FirstTimeFrameSelect = false
		}
	}
}

func (f *FilmStrip) DeleteFrame(fileName string) {
	for idx, frame := range backend.Backend.Frames {
		pinnedIdx := idx
		if fileName == frame.ThumbnailFilename {
			log.Printf("set cursor to backend frame %d, fileName: %s", idx, fileName)
			f.Cursor = pinnedIdx
			backend.Backend.RemoveAt(f.Cursor)
			break
		}
	}
}

func (f *FilmStrip) SyncToBackend() {
	log.Printf("syncing with backend")
	leftIndex := f.ViewOffset
	rightIndex := len(backend.Backend.Frames) - 1
	calculatedSize := f.ViewOffset + f.ViewSize
	if calculatedSize < rightIndex {
		rightIndex = calculatedSize
	}
	log.Printf("leftIndex=%d, calculatedSize=%d, rightIndex=%d", leftIndex, calculatedSize, rightIndex)
	if rightIndex > 0 {
		//log.Printf("will load visible frames from backend frames")
		for idx, frame := range backend.Backend.Frames[leftIndex:rightIndex] {
			pinnedFileName := frame.ThumbnailFilename
			pinnedThumbnailName := frame.ThumbnailFilename
			image := NewHotImageFromFile(pinnedThumbnailName, false, thumbnailWidth, thumbnailHeight,
				func(fileName string, event *fyne.PointEvent) {
					f.ExclusiveSelectFrame(pinnedFileName)
				},
				func(fileName string, event *fyne.PointEvent) {
					f.DeleteFrame(pinnedFileName)
					f.SyncToBackend()
				})
			if image == nil {
				log.Printf("error loading file %s", pinnedFileName)
				continue
			}
			f.VisibleFrames[idx] = image
		}
	}

	if rightIndex < thumbnailCount {
		//log.Printf("will insert white rects to unused visible frames")
		startIndex := rightIndex
		if startIndex < 0 {
			startIndex = 0
		}
		for idx := startIndex; idx < thumbnailCount; idx++ {
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
		ViewSize:      thumbnailCount,
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
	rootLayout.Layout(items, fyne.NewSize(config.WebcamCaptureWidth, config.WebcamDisplayHeight))
	rootContainer := fyne.NewContainerWithLayout(rootLayout, items...)
	filmstrip.Container = rootContainer

	return &filmstrip
}
