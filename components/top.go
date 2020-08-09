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
	"github.com/google/uuid"
	"gocv.io/x/gocv"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"time"

	"../backend"
	"../util"
)

var AnimationTopComponent *TopComponent

type TopComponent struct {
	Toolbar *Toolbar

	// root container
	Container *fyne.Container

	// camera
	Webcam   *gocv.VideoCapture
	WebcamImage *canvas.Image

	// contextual panel
	ContextPane *fyne.Container

	ProjectPanel *Gallery
	ChromaPanel *ChromaPanel
	BackgroundPanel *BackgroundPanel
}

type ChromaPanel struct {
	Check *widget.Check

	RedSlider *widget.Slider
	GreenSlider *widget.Slider
	BlueSlider *widget.Slider
	FuzzSlider *widget.Slider

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
	Container *fyne.Container
	BackgroundImageMat *gocv.Mat
	BackgroundImage *canvas.Image
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
	gocv.Resize(*b.BackgroundImageMat, &backgroundResized, image.Pt(1280, 720), 0, 0, gocv.InterpolationLinear)
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
	panelLayout.Layout([]fyne.CanvasObject{loadButton, backgroundPanel.BackgroundImage}, fyne.NewSize(480,300))
	container := fyne.NewContainerWithLayout(panelLayout, loadButton, backgroundPanel.BackgroundImage)
	backgroundPanel.Container = container

	return &backgroundPanel
}

type SuperDuperProjectPanel struct {
	ProjectNames *[]string

	Container *fyne.Container
}

func (s *SuperDuperProjectPanel) GetThumbNails() []fyne.Container {
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

func (c *TopComponent) Snapshot() error {
	projectName := backend.AnimationBackend.Name
	if projectName == "" {
		return errors.New("create a project first before saving snapshots")
	}
	snapshotDir := fmt.Sprintf(`%s\snapshots`, projectName)
	err := util.MkRelativeDir(snapshotDir)
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
	log.Printf("about to create file %s", fullAbsImageFilePath)
	imageFile, err := os.Create(fullAbsImageFilePath)
	if err != nil {
		return err
	}
	defer imageFile.Close()

	imageBytes := c.WebcamImage.Resource.Content()
	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return err
	}
	err = png.Encode(imageFile, img)
	if err != nil {
		return err
	}

	backend.AnimationBackend.Append(&backend.Frame{Filename: fullAbsImageFilePath})
	backend.AnimationBackend.Save()
	canvas.Refresh(c.WebcamImage)

	return nil
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
	//defer backgroundResized.Close()

	defer final.Close()
	defer sourceMat.Close()

	for {
		if ok := c.Webcam.Read(&sourceMat); !ok {
			log.Printf("Device closed")
			return
		}
		if sourceMat.Empty() {
			continue
		}

		if !c.ChromaPanel.Check.Checked {
			buf, err := gocv.IMEncode(".png", sourceMat)
			if err != nil {
				log.Printf("error: %s", err.Error())
			}

			c.WebcamImage.Resource = fyne.NewStaticResource("webcam", buf)
		} else {
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

			gocv.BitwiseAndWithMask(sourceHsv, sourceHsv, &captureResult, inverseMask) // green screened region deleted
			gocv.BitwiseAndWithMask(*c.BackgroundPanel.BackgroundResizedHsv, *c.BackgroundPanel.BackgroundResizedHsv, &backgroundResult, mask) // green screened region remains
			gocv.Add(backgroundResult, captureResult, &final)

			// displayable image should be in BGR
			gocv.CvtColor(final, &final, gocv.ColorHSVToBGR)

			// encode the final into png
			buf, err := gocv.IMEncode(".png", final)
			if err != nil {
				log.Printf("error: %s", err.Error())
			}

			c.WebcamImage.Resource = fyne.NewStaticResource("webcam", buf)

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

		canvas.Refresh(c.WebcamImage)
		time.Sleep(time.Duration(100) * time.Millisecond)
	}
}

const (
	webcamImageWidth = 640
	webcamImageHeight = 360
)

func ExistingProjectTapHandler(id string) error {
	log.Printf("will load existing project %s", id)
	err := backend.AnimationBackend.Load(id)
	AnimationFilmStripComponent.Tail()
	AnimationFilmStripComponent.SyncToBackend()
	return err
}

func NewProjectTapHandler(name string) error {
	log.Printf("will load new project %s", name)
	backend.AnimationBackend.RemoveAll()
	AnimationFilmStripComponent.Tail()
	AnimationFilmStripComponent.SyncToBackend()
	return nil
}

func NewTopComponent(webcam *gocv.VideoCapture) *TopComponent {
	webcamImage := canvas.Image{}
	webcamImage.SetMinSize(fyne.NewSize(webcamImageWidth, webcamImageHeight))

	component := TopComponent{
		Webcam: webcam,
		WebcamImage: &webcamImage,
	}

	leftLayout := layout.NewVBoxLayout()
	snapshotButton := widget.NewButton("Snapshot", func() {
		err := component.Snapshot()
		if err != nil {
			log.Printf("error snapshot: %s", err)
		}
		AnimationFilmStripComponent.Tail()
		AnimationFilmStripComponent.SyncToBackend()
		//TODO: update the filmstrip using c.WebcamImage
	})
	leftContainer := fyne.NewContainerWithLayout(leftLayout, &webcamImage, snapshotButton)

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
		Check:       widget.NewCheck("Enable", func(flag bool) {
			log.Printf("flag=%v", flag)
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

	chromaGroup := widget.NewGroup("Chroma Key Setup", chromaPanel.Check, chromaPanel.RedSlider, chromaPanel.GreenSlider, chromaPanel.BlueSlider, chromaPanel.FuzzSlider, chromaPanel.PreviewColor)
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
	rootLayout.Layout([]fyne.CanvasObject{leftContainer, tabContainer}, fyne.NewSize(1280,webcamImageHeight))
	rootContainer := fyne.NewContainerWithLayout(rootLayout, leftContainer, tabContainer)

	component.Container = rootContainer

	return &component
}
