package components

import (
	"bytes"
	"errors"
	"fmt"
	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/dialog"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/storage"
	"fyne.io/fyne/widget"
	"github.com/amarburg/go-fast-png"
	"github.com/google/uuid"
	"gocv.io/x/gocv"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"time"

	"../backend"
	"../config"
	"../util"
)

var AnimationTopComponent *TopComponent

type TopComponent struct {
	Toolbar *Toolbar

	CaptureMode CaptureMode

	// root container
	Container *fyne.Container

	// camera
	Webcam               *gocv.VideoCapture
	WebcamImageContainer *fyne.Container
	WebcamImage          *canvas.Image

	// contextual panel
	ContextPane *fyne.Container

	ProjectPanel    *Gallery
	ChromaPanel     *ChromaPanel
	BackgroundPanel *BackgroundPanel
}

type ChromaPanel struct {
	ChromaFilterToggle *widget.Check
	ColorPickerToggle  *widget.Check

	RedSlider   *widget.Slider
	GreenSlider *widget.Slider
	BlueSlider  *widget.Slider
	FuzzSlider  *widget.Slider

	PreviewColor *canvas.Rectangle

	Container *fyne.Container
}

func (c *ChromaPanel) GetChromaKey() color.Color {
	clr := color.RGBA{
		R: uint8(c.RedSlider.Value),
		G: uint8(c.GreenSlider.Value),
		B: uint8(c.BlueSlider.Value),
		A: 0,
	}
	return clr
}

type BackgroundPanel struct {
	Container            *fyne.Container
	BackgroundImageMat   *gocv.Mat
	BackgroundImage      *canvas.Image
	BackgroundResizedHsv *gocv.Mat
}

func (b *BackgroundPanel) RefreshDisplay() {
	if b.BackgroundImageMat != nil {
		img, err := b.BackgroundImageMat.ToImage()
		if err != nil {
			log.Printf("error: %s", err.Error())
			return
		}
		b.BackgroundImage.Image = img
		b.BackgroundImage.SetMinSize(fyne.NewSize(480, 270))
		canvas.Refresh(b.BackgroundImage)
	}
}

func (b *BackgroundPanel) GenerateResizedBackground() {
	backgroundResized := gocv.NewMat()
	backgroundResizedHsv := gocv.NewMat()
	gocv.Resize(*b.BackgroundImageMat, &backgroundResized, image.Pt(config.WebcamCaptureWidth, config.WebcamCaptureHeight), 0, 0, gocv.InterpolationLinear)
	gocv.CvtColor(backgroundResized, &backgroundResizedHsv, gocv.ColorBGRToHSV)
	b.BackgroundResizedHsv = &backgroundResizedHsv
}

func (b *BackgroundPanel) LoadFile(read fyne.URIReadCloser) {
	defer read.Close()
	fileName := read.URI().String()[len(read.URI().Scheme())+3:] // remove "file://"
	background := gocv.IMRead(fileName, gocv.IMReadColor)
	b.BackgroundImageMat = &background
}

func (b *BackgroundPanel) OpenFileDialog() {
	win := fyne.CurrentApp().Driver().AllWindows()[0]
	open := dialog.NewFileOpen(func(read fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		if read == nil {
			return
		}

		b.LoadFile(read)
		b.GenerateResizedBackground()
		b.RefreshDisplay()
	}, win)

	open.SetFilter(storage.NewExtensionFileFilter([]string{".png"}))
	open.Show()
}

func NewBackgroundPanel() *BackgroundPanel {
	backgroundPanel := BackgroundPanel{}
	loadButton := widget.NewButton("Load Background Image", func() {
		backgroundPanel.OpenFileDialog()
	})
	dummyMat := gocv.NewMatWithSizeFromScalar(gocv.NewScalar(255.0, 255.0, 255.0, 255.0), 270, 480, gocv.MatTypeCV8UC3)
	backgroundPanel.BackgroundImageMat = &dummyMat
	img, err := backgroundPanel.BackgroundImageMat.ToImage()
	if err != nil {
		return nil
	}
	backgroundPanel.BackgroundImage = canvas.NewImageFromImage(img)
	panelLayout := layout.NewVBoxLayout()
	backgroundPanel.BackgroundImage.SetMinSize(fyne.NewSize(480, 270))
	backgroundPanel.GenerateResizedBackground()
	panelLayout.Layout([]fyne.CanvasObject{loadButton, backgroundPanel.BackgroundImage}, fyne.NewSize(480, 300))
	container := fyne.NewContainerWithLayout(panelLayout, loadButton, backgroundPanel.BackgroundImage)
	backgroundPanel.Container = container

	return &backgroundPanel
}

type ProjectPanel struct {
	ProjectNames *[]string

	Container *fyne.Container
}

func (s *ProjectPanel) GetThumbNails() []fyne.Container {
	defer util.LogPerf("ProjectPanel.Snapshot()", time.Now())
	objs := make([]fyne.Container, 0)
	for _, name := range *s.ProjectNames {
		nameFixed := name
		rsc, _ := fyne.LoadResourceFromPath(fmt.Sprintf(`D:\Luke\Downloads\%s.png`, name))
		img := NewTappableIcon(rsc, nameFixed, func(id string) error {
			log.Printf("tapped on %s", nameFixed)
			//TODO: load project

			return nil
		})

		label := widget.NewLabel(name)
		obj := fyne.NewContainerWithLayout(layout.NewVBoxLayout(), img, label)

		objs = append(objs, *obj)
	}
	return objs
}

func (c *TopComponent) saveCanvasImage(canvasImage *canvas.Image, absImageFilepath string) (*image.Image, error) {
	imageFile, err := os.Create(absImageFilepath)
	if err != nil {
		return nil, err
	}
	defer imageFile.Close()

	imageBytes := canvasImage.Resource.Content()
	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return nil, err
	}
	encoder := fastpng.Encoder{
		CompressionLevel: fastpng.BestSpeed,
	}
	err = encoder.Encode(imageFile, img)
	if err != nil {
		return nil, err
	}

	return &img, nil
}

func (c *TopComponent) saveImage(img *image.Image, absImageFilepath string) error {
	imageFile, err := os.Create(absImageFilepath)
	if err != nil {
		return err
	}
	defer imageFile.Close()

	encoder := fastpng.Encoder{
		CompressionLevel: fastpng.BestSpeed,
	}

	return encoder.Encode(imageFile, *img)
}

func (c *TopComponent) Snapshot() error {
	defer util.LogPerf("TopComponent.Snapshot()", time.Now())
	//c.DisableCapture()
	//defer c.EnableCapture()

	projectName := backend.Backend.Name
	if projectName == "" {
		return errors.New("create a project first before saving snapshots")
	}

	snapshotDir := fmt.Sprintf(`%s\snapshots`, projectName)
	err := util.MkRelativeDir(snapshotDir)
	if err != nil {
		return err
	}

	snapshotThumbnailDir := fmt.Sprintf(`%s\snapshots\.thumbnails`, projectName)
	err = util.MkRelativeDir(snapshotThumbnailDir)
	if err != nil {
		return err
	}

	newUUID, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	baseDir, err := util.GetMocapBaseDir()
	if err != nil {
		return err
	}

	fullAbsImageFilePath := fmt.Sprintf(`%s\%s\%s.png`, baseDir, snapshotDir, newUUID.String())
	fullThumbnailImageFilePath := fmt.Sprintf(`%s\%s\%s.png`, baseDir, snapshotThumbnailDir, newUUID.String())

	img, err := c.saveCanvasImage(c.WebcamImage, fullAbsImageFilePath)
	if err != nil {
		return err
	}
	srcMat, err := gocv.ImageToMatRGB(*img)
	if err != nil {
		return err
	}
	defer srcMat.Close()

	thumbnailMat := gocv.NewMat()
	defer thumbnailMat.Close()

	gocv.Resize(srcMat, &thumbnailMat, image.Pt(thumbnailWidth, thumbnailHeight), 0, 0, gocv.InterpolationLinear)
	thumbnailImage, err := thumbnailMat.ToImage()
	err = c.saveImage(&thumbnailImage, fullThumbnailImageFilePath)
	if err != nil {
		return err
	}

	cursor := AnimationFilmStripComponent.Cursor
	log.Printf("cursor=%d", cursor)
	if cursor == -1 || cursor == len(backend.Backend.Frames)-1 {
		backend.Backend.Append(&backend.Frame{Filename: fullAbsImageFilePath, ThumbnailFilename: fullThumbnailImageFilePath})
	} else {
		backend.Backend.InsertAt(cursor+1, &backend.Frame{Filename: fullAbsImageFilePath, ThumbnailFilename: fullThumbnailImageFilePath})
	}

	backend.Backend.Save()
	canvas.Refresh(c.WebcamImage)

	return nil
}

func (c *TopComponent) SetCaptureMode(mode CaptureMode) {
	c.CaptureMode = mode
}

func (c *TopComponent) ReadWebCam(sourceMat *gocv.Mat) bool {
	ok := c.Webcam.Read(sourceMat)
	if !ok {
		return false
	}

	return !sourceMat.Empty()
}

func (c *TopComponent) captureLoopSleep() {
	time.Sleep(time.Duration(captureLoopSleepTime) * time.Millisecond)
}

func (c *TopComponent) CaptureLoop() {
	sourceMat := gocv.NewMat()
	if ok := c.Webcam.Read(&sourceMat); !ok {
		log.Printf("Device closed")
		return
	}

	sourceHsv := gocv.NewMat()
	chromaKey := gocv.NewMat()
	mask := gocv.NewMat()
	inverseMask := gocv.NewMat()

	final := gocv.NewMat()

	defer sourceHsv.Close()
	defer chromaKey.Close()
	defer mask.Close()
	defer inverseMask.Close()

	defer final.Close()
	defer sourceMat.Close()

	for { // start infinite capture loops
		startTime := time.Now()
		switch c.CaptureMode {
		case CaptureModeDisable:
			// do nothing
			//log.Printf("mode=CaptureModeDisable")
			c.captureLoopSleep()
			continue
		case CaptureModeNormal:
			// normal capture mode. no filter
			//log.Printf("mode=CaptureModeNormal")
			if !c.ReadWebCam(&sourceMat) {
				log.Printf("Device closed or empty read from webcam")
				c.captureLoopSleep()
				continue
			}
			//newStartTime := time.Now()
			buf, err := gocv.IMEncode(gocv.PNGFileExt, sourceMat)
			if err != nil {
				log.Printf("error: %s", err.Error())
				c.captureLoopSleep()
				continue
			}
			//log.Printf("IMEncode took %d ms", time.Since(newStartTime).Milliseconds())
			c.WebcamImage.Resource = fyne.NewStaticResource("webcam", buf)
			c.WebcamImageContainer.Objects[0] = c.WebcamImage
		case CaptureModeColorPick:
			// do nothing
		case CaptureModeChromaKey:
			// chroma key mode - apply chroma key filter and background image, if any
			//log.Printf("mode=CaptureModeChromaKey")
			if !c.ReadWebCam(&sourceMat) {
				log.Printf("Device closed or empty read from webcam")
				c.captureLoopSleep()
				continue
			}
			// image processing should use HSV
			gocv.CvtColor(sourceMat, &sourceHsv, gocv.ColorBGRToHSV)
			nowR := c.ChromaPanel.RedSlider.Value
			nowG := c.ChromaPanel.RedSlider.Value
			nowB := c.ChromaPanel.BlueSlider.Value
			nowF := c.ChromaPanel.FuzzSlider.Value

			chromaKey = gocv.NewMatFromScalar(gocv.NewScalar(nowB, nowG, nowR, 255.0), gocv.MatTypeCV8UC3)
			gocv.CvtColor(chromaKey, &chromaKey, gocv.ColorBGRToHSV)
			keys := gocv.Split(chromaKey)
			h := float64(keys[0].GetUCharAt(0, 0))
			h1 := math.Max(h-nowF, 0)
			h2 := math.Min(h+nowF, 179.0)

			// split HSV lower bounds into H, S, V channels
			lower := gocv.NewScalar(h1, 50.0, 100.0, 255.0)
			upper := gocv.NewScalar(h2, 255.0, 255.0, 255.0)

			gocv.InRangeWithScalar(sourceHsv, lower, upper, &mask)
			gocv.BitwiseNot(mask, &inverseMask)

			captureResult := gocv.NewMat()
			backgroundResult := gocv.NewMat()

			gocv.BitwiseAndWithMask(sourceHsv, sourceHsv, &captureResult, inverseMask)                                                         // green screened region deleted
			gocv.BitwiseAndWithMask(*c.BackgroundPanel.BackgroundResizedHsv, *c.BackgroundPanel.BackgroundResizedHsv, &backgroundResult, mask) // green screened region remains
			gocv.Add(backgroundResult, captureResult, &final)

			// displayable image should be in BGR
			gocv.CvtColor(final, &final, gocv.ColorHSVToBGR)

			// encode the final into png
			buf, err := gocv.IMEncode(gocv.PNGFileExt, final)
			if err != nil {
				log.Printf("error: %s", err.Error())
				c.captureLoopSleep()
				continue
			}

			c.WebcamImage.Resource = fyne.NewStaticResource("webcam", buf)
			c.WebcamImageContainer.Objects[0] = c.WebcamImage

			err = backgroundResult.Close()
			if err != nil {
				log.Printf("error closing backgroundResult due to %s", err.Error())
			}
			err = captureResult.Close()
			if err != nil {
				log.Printf("error closing captureResult due to %s", err.Error())
			}
			err = chromaKey.Close()
			if err != nil {
				log.Printf("error closing chromaKey due to %s", err.Error())
			}
		}

		canvas.Refresh(c.WebcamImageContainer)
		//		c.WebcamImageContainer.Refresh()
		//canvas.Refresh(c.WebcamImage)
		loopTiming := time.Since(startTime).Milliseconds()
		if loopTiming > 50 {
			log.Printf("warning. capture loop took too long: %d ms", loopTiming)
		}
		c.captureLoopSleep()
	} // infinite capture loop
}

type CaptureMode int

const (
	captureLoopSleepTime = 100

	CaptureModeDisable = iota
	CaptureModeNormal
	CaptureModeColorPick
	CaptureModeChromaKey
)

func ExistingProjectTapHandler(id string) error {
	defer util.LogPerf("ExistingProjectTapHandler()", time.Now())
	log.Printf("will load existing project %s", id)
	err := backend.Backend.Load(id)
	AnimationFilmStripComponent.Tail()
	AnimationFilmStripComponent.SyncToBackend()
	return err
}

func NewProjectTapHandler(name string) error {
	defer util.LogPerf("NewProjectTapHandler()", time.Now())
	log.Printf("will load new project %s", name)
	backend.Backend.RemoveAll()
	AnimationFilmStripComponent.Tail()
	AnimationFilmStripComponent.SyncToBackend()
	return nil
}

func NewTopComponent(webcam *gocv.VideoCapture) *TopComponent {
	webcamImage := canvas.Image{}
	webcamImage.SetMinSize(fyne.NewSize(config.WebcamDisplayWidth, config.WebcamDisplayHeight))
	webcamImageContainer := fyne.NewContainerWithLayout(layout.NewMaxLayout(), &webcamImage)

	component := TopComponent{
		Webcam:      webcam,
		WebcamImage: &webcamImage,
	}
	component.WebcamImageContainer = webcamImageContainer

	leftLayout := layout.NewVBoxLayout()
	snapshotButton := widget.NewButton("Snapshot", func() {
		err := component.Snapshot()
		if err != nil {
			log.Printf("error snapshot: %s", err)
		}
		AnimationFilmStripComponent.Tail()
		AnimationFilmStripComponent.SyncToBackend()
	})
	leftContainer := fyne.NewContainerWithLayout(leftLayout, webcamImageContainer, snapshotButton)

	absBaseDir, err := util.GetMocapBaseDir()
	if err != nil {
		log.Fatal(err)
	}

	/**
		.------------------------
	   	| projectTabContent
		|   .----------------------
		|   | projectPanel.Container
	    |   |                     |--- ThumbnailView, or
		|	|					  |--- NewItemInputForm
	*/
	projectTabContent := fyne.NewContainer()
	projectPanel := NewGallery(projectTabContent, Folder, absBaseDir, ExistingProjectTapHandler, NewProjectTapHandler)
	projectTabContent.AddObject(projectPanel.Container)
	component.ProjectPanel = projectPanel

	// chroma key tab content
	rightLayout := layout.NewCenterLayout()
	component.ContextPane = fyne.NewContainerWithLayout(rightLayout)

	chromaPanel := ChromaPanel{
		ChromaFilterToggle: widget.NewCheck("Apply Chroma Key Filter", func(flag bool) {
			if flag {
				component.ChromaPanel.ColorPickerToggle.Checked = false
				component.ChromaPanel.ColorPickerToggle.Refresh()
				component.SetCaptureMode(CaptureModeChromaKey)
			} else {
				component.SetCaptureMode(CaptureModeNormal)
			}
		}),
		ColorPickerToggle: widget.NewCheck("Color Picker Mode", func(flag bool) {
			if flag {
				component.ChromaPanel.ChromaFilterToggle.Checked = false
				component.ChromaPanel.ChromaFilterToggle.Refresh()
				component.SetCaptureMode(CaptureModeColorPick)
				// convert webcam image to a hotimage
				hotImage := NewHotImageFromCanvasImage(component.WebcamImage, true, config.WebcamDisplayWidth, config.WebcamDisplayHeight,
					func(s string, event *fyne.PointEvent) {
						x, y := event.Position.X, event.Position.Y
						bufReader := bytes.NewReader(component.WebcamImage.Resource.Content())
						img, err := fastpng.Decode(bufReader)
						if err != nil {
							log.Printf("error decoding webcam image: %s", err.Error())
							return
						}
						log.Printf("")
						clr := img.At(x*config.CaptureToDisplayWidthRatio, y*config.CaptureToDisplayWidthRatio) // clicks are on canvas image, where the dimensions are smaller than the underlying capture image
						r, g, b, a := clr.RGBA()
						log.Printf("left-clicked on webcam at %#v. color=(%d, %d, %d, %d)", event.Position, r/0x101, g/0x101, b/0x101, a/0x101)

						component.ChromaPanel.RedSlider.Value = float64(r / 0x101)
						component.ChromaPanel.GreenSlider.Value = float64(g / 0x101)
						component.ChromaPanel.BlueSlider.Value = float64(b / 0x101)
						component.ChromaPanel.FuzzSlider.Value = 20
						component.ChromaPanel.ColorPickerToggle.Checked = false
						component.ChromaPanel.ChromaFilterToggle.Checked = true
						component.ChromaPanel.PreviewColor.FillColor = clr
						component.ChromaPanel.ChromaFilterToggle.Checked = true
						component.ChromaPanel.ColorPickerToggle.Checked = false

						component.ChromaPanel.RedSlider.Refresh()
						component.ChromaPanel.GreenSlider.Refresh()
						component.ChromaPanel.BlueSlider.Refresh()
						component.ChromaPanel.FuzzSlider.Refresh()
						component.ChromaPanel.ColorPickerToggle.Refresh()
						component.ChromaPanel.ChromaFilterToggle.Refresh()
						component.ChromaPanel.PreviewColor.Refresh()
						component.ChromaPanel.ChromaFilterToggle.Refresh()
						component.ChromaPanel.ColorPickerToggle.Refresh()

						component.SetCaptureMode(CaptureModeChromaKey)
					}, func(s string, event *fyne.PointEvent) {
						log.Printf("right-clicked on webcam at %#v", event.Position)
					})
				component.WebcamImageContainer.Objects[0] = hotImage
			} else {
				component.SetCaptureMode(CaptureModeNormal)
				component.WebcamImageContainer.Objects[0] = component.WebcamImage
			}
		}),
		RedSlider:   widget.NewSlider(0, 255),
		GreenSlider: widget.NewSlider(0, 255),
		BlueSlider:  widget.NewSlider(0, 255),
		FuzzSlider:  widget.NewSlider(0, 255),
	}

	chromaPanel.PreviewColor = canvas.NewRectangle(chromaPanel.GetChromaKey())
	chromaPanel.PreviewColor.SetMinSize(fyne.NewSize(320, 36))

	chromaPanel.RedSlider.OnChanged = func(value float64) {
		chromaPanel.PreviewColor.FillColor = chromaPanel.GetChromaKey()
		canvas.Refresh(chromaPanel.PreviewColor)
	}
	chromaPanel.GreenSlider.OnChanged = func(value float64) {
		chromaPanel.PreviewColor.FillColor = chromaPanel.GetChromaKey()
		canvas.Refresh(chromaPanel.PreviewColor)
	}
	chromaPanel.BlueSlider.OnChanged = func(value float64) {
		chromaPanel.PreviewColor.FillColor = chromaPanel.GetChromaKey()
		canvas.Refresh(chromaPanel.PreviewColor)
	}

	chromaGroup := widget.NewGroup("Chroma Key Setup", chromaPanel.ChromaFilterToggle, chromaPanel.ColorPickerToggle, chromaPanel.RedSlider, chromaPanel.GreenSlider, chromaPanel.BlueSlider, chromaPanel.FuzzSlider, chromaPanel.PreviewColor)
	chromaTabContent := fyne.NewContainerWithLayout(layout.NewVBoxLayout(), chromaGroup)

	chromaPanel.Container = chromaTabContent
	component.ChromaPanel = &chromaPanel
	component.ContextPane = component.ChromaPanel.Container

	// background tab contents
	backgroundPanel := NewBackgroundPanel()
	backgroundTabContent := fyne.NewContainer(backgroundPanel.Container)
	component.BackgroundPanel = backgroundPanel

	// add all the tabs to tab container
	tabContainer := widget.NewTabContainer()
	tabContainer.Append(&widget.TabItem{
		Text:    "Project",
		Icon:    nil,
		Content: projectTabContent,
	})
	tabContainer.Append(&widget.TabItem{
		Text:    "Chroma Key",
		Icon:    nil,
		Content: chromaTabContent,
	})
	tabContainer.Append(&widget.TabItem{
		Text:    "Filter",
		Icon:    nil,
		Content: fyne.NewContainer(),
	})
	tabContainer.Append(&widget.TabItem{
		Text:    "Zoom",
		Icon:    nil,
		Content: fyne.NewContainer(),
	})
	tabContainer.Append(&widget.TabItem{
		Text:    "Background",
		Icon:    nil,
		Content: backgroundTabContent,
	})

	rootLayout := layout.NewHBoxLayout()
	rootLayout.Layout([]fyne.CanvasObject{leftContainer, tabContainer}, fyne.NewSize(config.WebcamCaptureWidth, config.WebcamDisplayHeight))
	rootContainer := fyne.NewContainerWithLayout(rootLayout, leftContainer, tabContainer)

	component.Container = rootContainer

	component.SetCaptureMode(CaptureModeNormal)

	return &component
}
