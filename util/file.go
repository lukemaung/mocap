package util

import (
	"fmt"
	"os"

	"../config"
)

// Mkdir creates dirName relative to Mocap base dir
func MkRelativeDir(dirName string) error {
	absBaseDir, err := GetMocapBaseDir()
	if err != nil {
		return fmt.Errorf("can't get user home dir due to: %s", err)
	}
	absTargetDir := fmt.Sprintf(`%s\%s`, absBaseDir, dirName)
	err = os.MkdirAll(absTargetDir, os.ModeDir)
	if err != nil {
		return fmt.Errorf("can't create path %s dir due to: %s", absTargetDir, err)
	}
	return nil
}

func GetMocapBaseDir() (string, error){
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("can't get user home dir due to: %s", err)
	}
	absBaseDir := fmt.Sprintf(`%s\%s`, homeDir,  config.MocapDir)
	return absBaseDir, nil
}