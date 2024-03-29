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
	WebcamImageContainer *fyne.Container
	WebcamImage          *canvas.Image

	// contextual panel
	ContextPane *fyne.Container

	ProjectPanel    *Gallery
	ChromaPanel     *ChromaPanel
	ZoomPanel       *ZoomPanel
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

type ZoomPanel struct {
	Container  *fyne.Container
	ZoomSlider *widget.Slider
	ZoomLabel  *widget.Label
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
	gocv.Resize(*b.BackgroundImageMat, &backgroundResized, image.Pt(config.WebcamCaptureWidth, config.WebcamCaptureHeight), 0, 0, gocv.InterpolationLanczos4)
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
	ok := backend.CurrentCamera().Read(sourceMat)
	if !ok {
		return false
	}

	return !sourceMat.Empty()
}

func (c *TopComponent) captureLoopSleep() {
	time.Sleep(time.Duration(captureLoopSleepTime) * time.Millisecond)
}

func (c *TopComponent) applyZoom(sourceMat *gocv.Mat) ([]byte, error) {
	factor := c.ZoomPanel.ZoomSlider.Value
	xMax := factor * config.WebcamCaptureWidth
	yMax := factor * config.WebcamCaptureHeight
	xOffset := (xMax - config.WebcamCaptureWidth) / 2
	yOffset := (yMax - config.WebcamCaptureHeight) / 2
	gocv.Resize(*sourceMat, sourceMat, image.Pt(0, 0), factor, factor, gocv.InterpolationLanczos4)
	final := sourceMat.Region(image.Rect(int(xOffset), int(yOffset), int(xOffset+config.WebcamCaptureWidth), int(yOffset+config.WebcamCaptureHeight)))
	defer final.Close()
	return gocv.IMEncode(gocv.PNGFileExt, final)
}

func (c *TopComponent) CaptureLoop() {
	sourceMat := gocv.NewMat()
	if ok := backend.CurrentCamera().Read(&sourceMat); !ok {
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
			buf, err := c.applyZoom(&sourceMat)
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

			buf, err := c.applyZoom(&final)
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
		loopTiming := time.Since(startTime).Milliseconds()
		if loopTiming > 200 {
			log.Printf("warning. capture loop took too long: %d ms", loopTiming)
		}
		c.captureLoopSleep()
	} // infinite capture loop
}

type CaptureMode int

const (
	captureLoopSleepTime = 200

	CaptureModeDisable = iota
	CaptureModeNormal
	CaptureModeColorPick
	CaptureModeChromaKey
)

func ExistingProjectTapHandler(projName string) error {
	defer util.LogPerf("ExistingProjectTapHandler()", time.Now())
	log.Printf("will load existing project %s", projName)
	err := backend.Backend.Load(projName)
	AnimationFilmStripComponent.Tail()
	AnimationFilmStripComponent.SyncToBackend()
	UpdateMocapTitle()
	return err
}

func NewProjectTapHandler(name string) error {
	defer util.LogPerf("NewProjectTapHandler()", time.Now())
	log.Printf("will load new project %s", name)
	backend.Backend.RemoveAll()
	AnimationFilmStripComponent.Tail()
	AnimationFilmStripComponent.SyncToBackend()
	UpdateMocapTitle()
	return nil
}

func DisplayUserTip(text string) {
	appWindow := *MocapApp.Window
	dialog.NewInformation("Useful Tip", text, appWindow)
}

func NewTopComponent() *TopComponent {
	webcamImage := canvas.Image{}
	webcamImage.SetMinSize(fyne.NewSize(config.WebcamDisplayWidth, config.WebcamDisplayHeight))
	webcamImageContainer := fyne.NewContainerWithLayout(layout.NewMaxLayout(), &webcamImage)

	component := TopComponent{
		WebcamImage: &webcamImage,
	}
	component.WebcamImageContainer = webcamImageContainer

	leftLayout := layout.NewVBoxLayout()
	snapshotButton := widget.NewButton("Snapshot", func() {
		err := component.Snapshot()
		if err != nil {
			DisplayUserTip("Please create/open a project before taking snapshots.")
		}
		AnimationFilmStripComponent.Tail()
		AnimationFilmStripComponent.SyncToBackend()
	})

	cameraButtons := make([]*widget.Button, 0)
	for camID := 0; camID < config.MaxCameras; camID ++ {
		pinnedCamID := camID
		cameraButtons = append(cameraButtons, widget.NewButton(fmt.Sprintf("Camera %d", pinnedCamID +1), func() {
			currentCaptureMode := component.CaptureMode
			component.SetCaptureMode(CaptureModeDisable)
			_, _, err := backend.SwitchCamera(pinnedCamID)
			if err != nil {
				DisplayUserTip("There is no camera at this slot. Will continue using previous camera. \nPlease connect a camera to this slot if you wish to use it.")
			}
			component.SetCaptureMode(currentCaptureMode)
		}))
	}
	cameraButtonContainer := fyne.NewContainerWithLayout(layout.NewHBoxLayout())
	for _, cb := range cameraButtons {
		cameraButtonContainer.AddObject(cb)
	}
	cameraButtonContainer.AddObject(widget.NewButton("?", func() {
		DisplayUserTip(fmt.Sprintf(`Mocap Animation
Mocap Animation v%s

Powered by opensource software:
- https://github.com/hybridgroup/gocv
- https://developer.fyne.io

Icons made by https://www.flaticon.com/authors/pixel-perfect

Copyright (c) Luke Maung 2020`, config.Version))
	}))

	leftContainer := fyne.NewContainerWithLayout(leftLayout, webcamImageContainer, snapshotButton, cameraButtonContainer)

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
		ChromaFilterToggle: widget.NewCheck("", func(flag bool) {
			if flag {
				component.ChromaPanel.ColorPickerToggle.Checked = false
				component.ChromaPanel.ColorPickerToggle.Refresh()
				component.SetCaptureMode(CaptureModeChromaKey)
			} else {
				component.SetCaptureMode(CaptureModeNormal)
			}
		}),
		ColorPickerToggle: widget.NewCheck("", func(flag bool) {
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
						tempMat, err := gocv.ImageToMatRGBA(img)
						if err != nil {
							log.Printf("error decoding webcam image: %s", err.Error())
							return
						}
						buf, err := component.applyZoom(&tempMat)
						tempMat.Close()
						finalImg, err := fastpng.Decode(bytes.NewReader(buf))
						clr := finalImg.At(x*config.CaptureToDisplayWidthRatio, y*config.CaptureToDisplayWidthRatio) // clicks are on canvas image, where the dimensions are smaller than the underlying capture image
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

	redLabel := widget.NewLabel("R (0)")
	greenLabel := widget.NewLabel("G (0)")
	blueLabel := widget.NewLabel("B (0)")
	fuzzLabel := widget.NewLabel("Fuzz (0)")
	chromaPanel.RedSlider.OnChanged = func(value float64) {
		redLabel.SetText(fmt.Sprintf("R (%d)", int(value)))
		chromaPanel.PreviewColor.FillColor = chromaPanel.GetChromaKey()
		canvas.Refresh(chromaPanel.PreviewColor)
	}
	chromaPanel.GreenSlider.OnChanged = func(value float64) {
		greenLabel.SetText(fmt.Sprintf("G (%d)", int(value)))
		chromaPanel.PreviewColor.FillColor = chromaPanel.GetChromaKey()
		canvas.Refresh(chromaPanel.PreviewColor)
	}
	chromaPanel.BlueSlider.OnChanged = func(value float64) {
		blueLabel.SetText(fmt.Sprintf("B (%d)", int(value)))
		chromaPanel.PreviewColor.FillColor = chromaPanel.GetChromaKey()
		canvas.Refresh(chromaPanel.PreviewColor)
	}
	chromaPanel.FuzzSlider.OnChanged = func(value float64) {
		fuzzLabel.SetText(fmt.Sprintf("Fuzz (%d)", int(value)))
		chromaPanel.PreviewColor.FillColor = chromaPanel.GetChromaKey()
		canvas.Refresh(chromaPanel.PreviewColor)
	}
	chromaToggleGroup := fyne.NewContainerWithLayout(layout.NewFormLayout(), widget.NewLabel("Apply Chroma Key Filter") , chromaPanel.ChromaFilterToggle)
	pickerToggleGroup := fyne.NewContainerWithLayout(layout.NewFormLayout(), widget.NewLabel("Color Picker Mode"), chromaPanel.ColorPickerToggle)
	redGroup := fyne.NewContainerWithLayout(layout.NewFormLayout(), redLabel, chromaPanel.RedSlider)
	greenGroup := fyne.NewContainerWithLayout(layout.NewFormLayout(), greenLabel, chromaPanel.GreenSlider)
	blueGroup := fyne.NewContainerWithLayout(layout.NewFormLayout(), blueLabel, chromaPanel.BlueSlider)
	fuzzGroup := fyne.NewContainerWithLayout(layout.NewFormLayout(), fuzzLabel, chromaPanel.FuzzSlider)
	chromaTabContent := fyne.NewContainerWithLayout(layout.NewVBoxLayout(), chromaToggleGroup, pickerToggleGroup, redGroup, greenGroup, blueGroup, fuzzGroup, chromaPanel.PreviewColor)

	chromaPanel.Container = chromaTabContent
	component.ChromaPanel = &chromaPanel
	component.ContextPane = component.ChromaPanel.Container

	// zoom panel
	zoomLabel := widget.NewLabel("1.0")
	zoomSlider := widget.NewSlider(1.0, 5.0)
	zoomSlider.Step = 0.1
	zoomSlider.OnChanged = func(value float64) {
		text := fmt.Sprintf("%.1f", value)
		zoomLabel.SetText(text)
	}
	zoomContainer := fyne.NewContainerWithLayout(layout.NewVBoxLayout(), fyne.NewContainerWithLayout(layout.NewFormLayout(), zoomLabel, zoomSlider))
	zoomPanel := ZoomPanel{
		Container:  zoomContainer,
		ZoomSlider: zoomSlider,
	}
	component.ZoomPanel = &zoomPanel

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
	//TODO: implement this some day
	//tabContainer.Append(&widget.TabItem{
	//	Text:    "Filter",
	//	Icon:    nil,
	//	Content: fyne.NewContainer(),
	//})
	tabContainer.Append(&widget.TabItem{
		Text:    "Zoom",
		Icon:    nil,
		Content: zoomContainer,
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
