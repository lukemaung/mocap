package backend

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"../util"
)

var Backend AnimationBackend

type Frame struct {
	Filename string
}

type AnimationBackend struct {
	Name string
	Frames []*Frame
}

func (f *AnimationBackend) Append(frame *Frame) {
	f.Frames = append(f.Frames, frame)
}

func (f *AnimationBackend) InsertAt(index int, frame *Frame) {
	log.Printf("will insert frame image %s at index %d", frame.Filename, index)
	if len(f.Frames) == index {
		f.Append(frame)
		return
	}
	f.Frames = append(f.Frames[:index+1], f.Frames[index:]...)
	f.Frames[index] = frame
}

func (f *AnimationBackend) RemoveAt(index int) {
	log.Printf("will delete backend frame %d/%d", index, len(f.Frames))

	//frames := make([]*Frame, 0)
	//frames = append(frames, f.Frames[:index]...)
	f.Frames = append(f.Frames[:index], f.Frames[index+1:]...)
}

func (f *AnimationBackend) RemoveAll() {
	log.Printf("clearing all frames from backend")
	frames := make([]*Frame, 0)
	f.Frames = frames
}

func (f *AnimationBackend) Save() error {
	log.Printf("saving %d frames into project %s", len(f.Frames), f.Name)

	bytes, err := json.Marshal(f)
	if err != nil {
		return err
	}

	err = util.MkRelativeDir(f.Name)
	if err != nil {
		return err
	}

	baseDir, err := util.GetMocapBaseDir()
	if err != nil {
		return err
	}

	fullPath := fmt.Sprintf(`%s\%s\animation.json`,baseDir, f.Name)
	log.Printf("about to write file %s", fullPath)
	return ioutil.WriteFile(fullPath, bytes, os.ModePerm)
}

func (f *AnimationBackend) Load(fileName string) error {
	log.Printf("will load project %s", fileName)
	baseDir, err := util.GetMocapBaseDir()
	if err != nil {
		return err
	}

	fullFileName := fmt.Sprintf(`%s\%s\animation.json`, baseDir, fileName)
	fileBytes, err := ioutil.ReadFile(fullFileName)
	if err != nil {
		return err
	}

	newAnimation := AnimationBackend{}
	err = json.Unmarshal(fileBytes, &newAnimation)
	if err != nil {
		return err
	}

	f.Name = newAnimation.Name
	f.Frames = newAnimation.Frames

	log.Printf("loaded %d frames into project %s", len(f.Frames), fileName)
	return nil
}