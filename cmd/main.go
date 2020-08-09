package main

import (
	"../components"
	"fmt"
	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
	"gocv.io/x/gocv"
	_ "image/png"
	"log"

	"../util"
)


var (
	deviceID int
	err      error
)

type Mocap struct {
	webcam   *gocv.VideoCapture
	webcamImage *canvas.Image
	container *fyne.Container
}

func startApp() {
	// open webcam
	deviceID := 1
	webcam, err := gocv.OpenVideoCapture(deviceID)
	webcam.Set(gocv.VideoCaptureFrameWidth, 1280)
	webcam.Set(gocv.VideoCaptureFrameHeight, 720)
	if err != nil {
		fmt.Printf("Error opening capture device: %v\n", deviceID)
		return
	}

	defer webcam.Close()

	mocapApp := app.New()
	mocapAppWindow := components.NewMocapAppWindow(mocapApp, webcam)
	window := *mocapAppWindow.Window
	window.ShowAndRun()
}

func startFoo() {
	err := util.MkRelativeDir("") // empty -> just create base mocap dir
	if err != nil {
		log.Fatalf("error creating dir due to: %s", err)
	}

	myApp := app.New()
	myWindow := myApp.NewWindow("Entry Widget")

	input := widget.NewEntry()
	input.SetPlaceHolder("Enter text...")

	content := widget.NewVBox(input, widget.NewButton("Save", func() {
		log.Println("Content was:", input.Text)
	}))

	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}

func testForm() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Form Widget")

	entry := widget.NewEntry()
	textArea := widget.NewMultiLineEntry()

	form := &widget.Form{
		Items: []*widget.FormItem{ // we can specify items in the constructor
			{"Entry", entry}},
		OnSubmit: func() { // optional, handle form submission
			log.Println("Form submitted:", entry.Text)
			log.Println("multiline:", textArea.Text)
			myWindow.Close()
		},
	}

	// we can also append items
	form.Append("Text", textArea)

	f := fyne.NewContainerWithLayout(layout.NewVBoxLayout(), form)
	myWindow.SetContent(f)
	myWindow.ShowAndRun()
}
func main() {
	//startFoo()
	startApp()
	//testApp()
	//testForm()
}
