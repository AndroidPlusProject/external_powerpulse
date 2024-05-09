package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
)

type Device struct {
	//Locks
	BoostMutex   sync.Mutex //Delays and prevents boosts based on time deltas
	ProfileLock  bool       //Protects the live profile
	ProfileMutex sync.Mutex //Prevents race conditions when toggling quickly between profiles

	Buffered            []BufferedWrite `json:"-"`                 //Buffered string values ready to be synced to each path
	Paths               *Paths                                     //Manifest of paths to device settings
	ProfileBoot         string      `json:"profile_boot"`          //Default profile, used permanently without a profile manager
	ProfileBootDuration json.Number `json:"profile_boot_duration"` //Force sets the boot profile for X seconds (no decimals) before setting the first requested profile after init
	ProfileInheritance  []string    `json:"profile_inheritance"`   //Profile order for inheritance of configurations
	ProfileOrder        []string    `json:"profile_order"`         //Profile order for stargazing
	Profiles            map[string]*Profile                        //Manifest of device settings per profile
	ProfilesInherited   map[string]*Profile `json:"-"`             //Manifest of device settings per profile, with inheritance expansion caching
	Profile             string `json:"-"`                          //The currently loaded profile
}

type BufferedWrite struct {
	Path string
	Data string
}

func (dev *Device) ReadBool(path string) (bool, error) {
	buffer, err := ioutil.ReadFile(path)
	if err != nil {
		return false, err
	} else {
		if buffer[len(buffer)-1] == '\n' { buffer = buffer[:len(buffer)-1] }
	}
	switch string(buffer) {
	case "1", "t", "T", "true", "True", "TRUE", "y", "Y", "yes", "Yes", "YES", "enabled", "Enabled", "ENABLED":
		return true, nil
	case "0", "f", "F", "false", "False", "FALSE", "n", "N", "no", "No", "NO", "disabled", "Disabled", "DISABLED":
		return false, nil
	}
	return false, fmt.Errorf("Unknown bool interface for %s", path)
}

func (dev *Device) BufferWriteBool(path string, data bool) error {
	buffer, err := ioutil.ReadFile(path)
	if err != nil {
		Warn("Failed to read from path %s: %v", path, err)
	} else {
		if buffer[len(buffer)-1] == '\n' { buffer = buffer[:len(buffer)-1] }
	}
	val1, val0 := "", ""
	switch string(buffer) {
	case "1", "0": val1, val0 = "1", "0"
	case "t", "f": val1, val0 = "t", "f"
	case "T", "F": val1, val0 = "T", "F"
	case "true", "false": val1, val0 = "true", "false"
	case "True", "False": val1, val0 = "True", "False"
	case "TRUE", "FALSE": val1, val0 = "TRUE", "FALSE"
	case "y", "n": val1, val0 = "y", "n"
	case "Y", "N": val1, val0 = "Y", "N"
	case "yes", "no": val1, val0 = "yes", "no"
	case "Yes", "No": val1, val0 = "Yes", "No"
	case "YES", "NO": val1, val0 = "YES", "NO"
	default:
		Warn("Using default handler for bool interface on %s", path)
		//Use 1 and 0 as a last resort
		val1, val0 = "1", "0"
	}
	if data {
		if string(buffer) == val1 {
			Debug("Skipping reset !> %s", path)
			return nil
		}
		dev.BufferWrite(path, val1)
		return nil
	}
	if string(buffer) == val0 {
		Debug("Skipping reset !> %s", path)
		return nil
	}
	dev.BufferWrite(path, val0)
	return nil
}

func (dev *Device) BufferWriteNumber(path string, data interface{}) {
	switch v := data.(type) {
	case float64, float32:
		dev.BufferWrite(path, fmt.Sprintf("%.0F", v))
		return
	}
	dev.BufferWrite(path, fmt.Sprintf("%d", data))
}

func (dev *Device) BufferWrite(path string, data string) {
	if path == "" || data == "" {
		return
	}

	if dev.Buffered == nil {
		dev.Buffered = make([]BufferedWrite, 0)
	}
	found := false
	for i := 0; i < len(dev.Buffered); i++ {
		if dev.Buffered[i].Path == path {
			dev.Buffered[i].Data = data
			found = true
			break
		}
	}
	if !found {
		dev.Buffered = append(dev.Buffered, BufferedWrite{Path: path, Data: data})
	}
}

func (dev *Device) write(path, data string) error {
	if data == "" {
		return nil //Skip empty config options
	}

	dataBytes := make([]byte, 0)
	if data != "-" {
		dataBytes = []byte(data)
	}

	/*buffer, err := ioutil.ReadFile(path)
	if err == nil && len(buffer) > 0 && buffer[len(buffer)-1] == '\n' {
		buffer = buffer[:len(buffer)-1]
	}

	if string(buffer) == string(dataBytes) {
		Debug("Skipping reset !> %s", path)
		return nil
	}

	if len(dataBytes) > 0 {
		Debug("Writing '%s' > %s", string(dataBytes), path)
	} else {
		Debug("Clearing %s", path)
	}*/

	//To prevent edge cases with partial applications due to invalid values or inheritance, gracefully log the error and move on
	err := ioutil.WriteFile(path, dataBytes, 0664)
	if err != nil {
		Error("Failed writing '%s' > %s: %v", string(dataBytes), path, err)
	}
	return nil
}
