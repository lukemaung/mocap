package components

import (
	"fmt"
	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
	"image/color"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"../backend"
	"../util"
)

type TappableIcon struct {
	widget.Icon
	ID               string
	IconTapHandler   func(id string) error
	ButtonTapHandler func()
}

func NewTappableIcon(res fyne.Resource, label string, iconTapHandler func(id string) error) *TappableIcon {
	icon := &TappableIcon{}
	icon.ID = label
	icon.ExtendBaseWidget(icon)
	icon.SetResource(res)
	icon.IconTapHandler = iconTapHandler
	return icon
}

func (t *TappableIcon) Tapped(pointEvent *fyne.PointEvent) {
	if t.IconTapHandler == nil {
		log.Printf("warning: tap handler is not registered")
		return
	}
	t.IconTapHandler(t.ID)
}

func (t *TappableIcon) TappedSecondary(_ *fyne.PointEvent) {
}

func (t *TappableIcon) MinSize() fyne.Size {
	t.ExtendBaseWidget(t)
	return fyne.NewSize(120, 90)
}

type Gallery struct {
	ParentContainer *fyne.Container // project tab container

	Type GalleryType

	ItemNames []string

	Container *fyne.Container // active container

	ThumbnailsPanel *fyne.Container // contains ThumbnailView
	ThumbnailView   *fyne.Container // contains Thumbnails
	NewEntryView    *fyne.Container // contains form to create new file/folder

	Thumbnails []fyne.Container // part of ThumbnailView

	IconTapHandler func(id string) error
}

func (s *Gallery) RegenerateThumbnails() {
	s.Thumbnails = make([]fyne.Container, 0)
	baseDir, err := util.GetMocapBaseDir()
	if err != nil {
		log.Printf("argh")
		return
	}

	for _, name := range s.ItemNames {
		absProjectDir := fmt.Sprintf(`%s\%s`, baseDir, name)

		//absThumbnailDir := fmt.Sprintf(`%s\snapshots\.thumbnails`, absProjectDir)
		absThumbnailDir := fmt.Sprintf(`%s\snapshots`, absProjectDir)
		outputDirRead, _ := os.Open(absThumbnailDir)
		outputDirFiles, _ := outputDirRead.Readdir(0)

		var image fyne.CanvasObject
		if len(outputDirFiles) > 0 {
			for _, fileOrdir := range outputDirFiles {
				if !fileOrdir.IsDir() {
					absThumbnailFilePath := fmt.Sprintf(`%s\%s`, absThumbnailDir, fileOrdir.Name())
					rsc, _ := fyne.LoadResourceFromPath(absThumbnailFilePath)
					image = NewTappableIcon(rsc, name, s.IconTapHandler)
					break
				}
			}
		}

		if image == nil {
			image = canvas.NewRectangle(color.White)
		}

		label := widget.NewLabel(name)
		obj := fyne.NewContainerWithLayout(layout.NewVBoxLayout(), image, label)

		s.Thumbnails = append(s.Thumbnails, *obj)
	}
}

func sliceContainsItem(slice []string, item string) bool {
	for _, test := range slice {
		if test == item {
			return true
		}
	}
	return false
}

//FIXME; sortBySecondCanvasObject

func (s *Gallery) Add(fileName string) {
	if sliceContainsItem(s.ItemNames, fileName) {
		log.Printf("%s already exists", fileName)
		return
	}
	s.ItemNames = append(s.ItemNames, fileName)
	sort.Strings(s.ItemNames)

	rsc, _ := fyne.LoadResourceFromPath(fmt.Sprintf(`D:\Luke\Downloads\%s.png`, fileName))
	image := NewTappableIcon(rsc, fileName, s.IconTapHandler)
	label := widget.NewLabel(fileName)
	obj := fyne.NewContainerWithLayout(layout.NewVBoxLayout(), image, label)
	s.Thumbnails = append(s.Thumbnails, *obj)
	s.ThumbnailsPanel.AddObject(obj)

	//s.RegenerateThumbnails()
}

func (s *Gallery) Remove(fileName string) {
	//delete(s.ItemNames, fileName)
	//s.RegenerateThumbnails()
}

func (s *Gallery) OnTapNewButton() {
	s.ActivateNewEntryInputView()
}

func (s *Gallery) ActivateThumbnailView() {
	s.Container = s.ThumbnailView
	s.ParentContainer.Objects = []fyne.CanvasObject{s.Container}
	s.ParentContainer.Refresh()
}

func (s *Gallery) ActivateNewEntryInputView() {
	s.Container = s.NewEntryView
	s.ParentContainer.Objects = []fyne.CanvasObject{s.Container}
	s.ParentContainer.Refresh()
}

func (s *Gallery) OnSubmitNewItem() {

}
func (s *Gallery) OnCancelNewItem() {

}

type GalleryType int

const (
	Folder GalleryType = iota
	File
)

var fileExtensions = []string{".png", ".jpg", ".jpeg", ".gif", ".bmp"}

func hasAllowedExtension(fileName string) bool {
	for _, allowed := range fileExtensions {
		if strings.HasSuffix(fileName, allowed) {
			return true
		}
	}
	return false
}

func NewGallery(parentContainer *fyne.Container, galleryType GalleryType, baseDir string,
	existingProjectTapHandler func(id string) error, newProjectTapHandler func(name string) error) *Gallery {

	galleryContainer := Gallery{
		ParentContainer: parentContainer,
		Type:            galleryType,
		IconTapHandler:  existingProjectTapHandler,
		ItemNames:       make([]string, 0),
	}

	fileNames := make([]string, 0)
	switch galleryType {
	case Folder:
		dirs, err := ioutil.ReadDir(baseDir)
		if err != nil {
			log.Printf("no directories in %s", baseDir)
			break
		}
		for _, dir := range dirs {
			fileNames = append(fileNames, dir.Name())
			galleryContainer.ItemNames = append(galleryContainer.ItemNames, dir.Name())
		}

	case File:
		file, err := os.Open(baseDir)
		defer file.Close()
		if err != nil {
			log.Printf("no directories in %s", baseDir)
			break
		}
		fileInfo, err := file.Readdir(-1)
		if err != nil {
			log.Printf("no directories in %s", baseDir)
			break
		}
		for _, fInfo := range fileInfo {
			if hasAllowedExtension(fInfo.Name()) {
				fileNames = append(fileNames, fInfo.Name())
				galleryContainer.ItemNames = append(galleryContainer.ItemNames, fInfo.Name())
			}
		}
	}
	galleryContainer.RegenerateThumbnails()

	fileIconCellWidth := 160
	fileIconSize := 90
	fileTextSize := 20
	//verticalExtra := 4
	thumbnailsPanel := fyne.NewContainerWithLayout(layout.NewGridWrapLayout(fyne.NewSize(fileIconCellWidth,
		fileIconSize+theme.Padding()+fileTextSize)))

	for _, obj := range galleryContainer.Thumbnails {
		o := obj
		thumbnailsPanel.AddObject(&o)
	}
	galleryContainer.ThumbnailsPanel = thumbnailsPanel

	galleryContainer.IconTapHandler = existingProjectTapHandler

	scrollContainer := widget.NewVScrollContainer(thumbnailsPanel)
	scrollContainer.SetMinSize(fyne.NewSize(fileIconCellWidth*2+theme.Padding(),
		webcamImageHeight)) // NOTE: it's critical to call SetMinSize() on containers

	newButton := widget.NewButton("New", func() {
		galleryContainer.OnTapNewButton()
	})

	thumbnailViewContainer := fyne.NewContainerWithLayout(layout.NewVBoxLayout(), newButton, scrollContainer) // gridlayout makes sure the
	thumbnailViewContainer.Resize(fyne.NewSize(webcamImageWidth, webcamImageHeight))
	galleryContainer.ThumbnailView = thumbnailViewContainer

	projectEntry := widget.NewEntry()

	label := "New Project Name"
	if galleryContainer.Type != Folder {
		label = "New Image URL"
	}

	newThingForm := widget.Form{
		BaseWidget: widget.BaseWidget{},
		Items: []*widget.FormItem{
			{label, projectEntry},
		},
		OnSubmit: func() {
			log.Printf("creating new project %s", projectEntry.Text)
			backend.Backend.Name = projectEntry.Text
			backend.Backend.RemoveAll()
			err := backend.Backend.Save()
			if err != nil {
				log.Printf("there was an error saving creating project %s: %s", projectEntry.Text, err.Error())
			}
			newProjectTapHandler(projectEntry.Text)
			galleryContainer.Add(projectEntry.Text)
			galleryContainer.ActivateThumbnailView()

		},
		OnCancel: func() {
			log.Printf("canceling saving of new project")
			galleryContainer.ActivateThumbnailView()
		},
	}

	newThingContainer := fyne.NewContainerWithLayout(layout.NewGridLayout(1), &newThingForm)
	newThingContainer.Resize(fyne.NewSize(webcamImageWidth, webcamImageHeight))

	galleryContainer.NewEntryView = newThingContainer
	galleryContainer.Container = newThingContainer
	galleryContainer.Container.Hide()
	galleryContainer.Container.Show()
	galleryContainer.Container = galleryContainer.ThumbnailView

	return &galleryContainer
}

type HotImage struct {
	widget.BaseWidget

	min      fyne.Size
	image    *canvas.Image
	Selected bool

	OnTap          func(fileName string, ev *fyne.PointEvent)
	OnSecondaryTap func(fileName string, ev *fyne.PointEvent)
}

func (r *HotImage) SetMinSize(size fyne.Size) {
	r.Resize(size)
}

func (r *HotImage) MinSize() fyne.Size {
	return r.min
}

func (r *HotImage) CreateRenderer() fyne.WidgetRenderer {
	return &HotImageWidgetRenderer{hotImage: r}
}

func (r *HotImage) Tapped(ev *fyne.PointEvent) {
	if r.OnTap != nil {
		r.OnTap(r.image.File, ev)
	}
}

func (r *HotImage) TappedSecondary(ev *fyne.PointEvent) {
	if r.OnSecondaryTap != nil {
		r.OnSecondaryTap(r.image.File, ev)
	}
}

func NewHotImage(fileName string, forceResize bool, width int, height int, onTap func(string, *fyne.PointEvent), onSecondaryTap func(string, *fyne.PointEvent)) *HotImage {
	img := canvas.NewImageFromFile(fileName)
	size := fyne.NewSize(width, height)
	if forceResize {
		img.Resize(size)
	}
	r := &HotImage{image: img, min: size, OnTap: onTap, OnSecondaryTap: onSecondaryTap}
	r.ExtendBaseWidget(r)
	return r
}

type HotImageWidgetRenderer struct {
	hotImage *HotImage
}

func (r *HotImageWidgetRenderer) Layout(size fyne.Size) {
	r.hotImage.image.Resize(size)
}

func (r *HotImageWidgetRenderer) MinSize() fyne.Size {
	return r.MinSize()
}

func (r *HotImageWidgetRenderer) Refresh() {
	canvas.Refresh(r.hotImage)
}

func (r *HotImageWidgetRenderer) BackgroundColor() color.Color {
	return theme.BackgroundColor()
}

func (r *HotImageWidgetRenderer) Objects() []fyne.CanvasObject {
	if r.hotImage.Selected {
		rect := canvas.NewRectangle(color.White)
		rect.StrokeColor = color.White
		rect.StrokeWidth = 2.0
		rect.FillColor = color.Transparent
		rect.Resize(r.hotImage.MinSize())
		return []fyne.CanvasObject{r.hotImage.image, rect}
	}
	return []fyne.CanvasObject{r.hotImage.image}
}

func (r *HotImageWidgetRenderer) Destroy() {
}
