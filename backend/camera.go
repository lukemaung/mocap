package backend

import (
	"gocv.io/x/gocv"
	"log"

	"../config"
)

var (
	Cameras = map[int]*gocv.VideoCapture{}
)

var CurrentWebcamID = -1

func CurrentCamera() *gocv.VideoCapture {
	return Cameras[CurrentWebcamID]
}

func SwitchCamera(deviceID int) (*gocv.VideoCapture, int) {
	if deviceID == CurrentWebcamID {
		return Cameras[CurrentWebcamID], CurrentWebcamID
	}
	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		log.Printf("error opening new webcam %d. reopening previous %d", deviceID, CurrentWebcamID)
		return Cameras[CurrentWebcamID], CurrentWebcamID
	}
	log.Printf("opened cam %d", deviceID)
	CurrentWebcamID = deviceID
	Cameras[CurrentWebcamID] = webcam
	webcam.Set(gocv.VideoCaptureFrameWidth, config.WebcamCaptureWidth)
	webcam.Set(gocv.VideoCaptureFrameHeight, config.WebcamCaptureHeight)
	return webcam, CurrentWebcamID
}
