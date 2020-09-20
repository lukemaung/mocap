package config

const (
	Version = "1.0.0"

	MocapDir = "Mocap Animation"

	WebcamCaptureWidth  = 1280
	WebcamCaptureHeight = 720

	WebcamDisplayWidth  = 640
	WebcamDisplayHeight = 360

	CaptureToDisplayWidthRatio  = WebcamCaptureWidth / WebcamDisplayWidth
	CaptureToDisplayHeightRatio = WebcamCaptureHeight / WebcamDisplayHeight

	MaxCameras = 7
)
