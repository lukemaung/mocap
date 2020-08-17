package main

import (
	"flag"
	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
	"gocv.io/x/gocv"
	_ "image/png"
	"log"
	"os"
	"runtime/pprof"

	"./backend"
	"./components"
	"./config"
	"./util"
)

type Mocap struct {
	webcam      *gocv.VideoCapture
	webcamImage *canvas.Image
	container   *fyne.Container
}

func closeAllWebcams() {
	components.AnimationTopComponent.SetCaptureMode(components.CaptureModeDisable)
	for idx, cam := range backend.Cameras {
		err := cam.Close()
		if err != nil {
			log.Printf("error closing webcam %d: %s", idx, err.Error())
			continue
		}
		log.Printf("closed cam %d", idx)
	}
}

func startApp() {
	firstCamera := -1
	for deviceID := 0; deviceID < config.MaxCameras; deviceID++ {
		webcam, _, _ := backend.SwitchCamera(deviceID)
		if webcam != nil && firstCamera < 0 {
			firstCamera = deviceID
		}
	}
	backend.SwitchCamera(firstCamera)
	defer closeAllWebcams()

	mocapApp := app.New()
	mocapAppWindow := components.NewMocapAppWindow(mocapApp)
	window := *mocapAppWindow.Window
	window.SetFixedSize(true)
	components.MocapApp = mocapAppWindow
	components.UpdateMocapTitle()

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

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	//startFoo()
	startApp()
	//testApp()
	//testForm()
}
