package main

import (
	"C"
	"encoding/json"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/spf13/pflag"
)

var (
	lock sync.Mutex

	device *Device = nil
	manifests = []string{
		"./powerpulse.json",
		"/data/local/tmp/powerpulse.json",
		"/vendor/etc/powerpulse.json",
		"/system/vendor/etc/powerpulse.json",
		"/system/etc/powerpulse.json",
		"/etc/powerpulse.json",
	}
	profileNow = ""
	profileLast = ""
	debug = true
	verbose = true
	booted = false
	bootedProfile = false
)

//export PowerPulse_Stargaze
func PowerPulse_Stargaze() {
	go stargaze()
}
func stargaze() {
	Info("Stargazing for desired profile")
	found := false
	for profileName := range device.Profiles {
		if profileNow == profileName {
			found = true
			break
		}
	}
	if !found {
		//Could be risky, but we're being asked to choose here - next time it will be the user's last choice if paths/powerpulse/profile is set, or continue being this if user lets be
		//For any matter, it shouldn't hurt to start with the highest performing profile until the user limits it
		profileNow = device.ProfileOrder[len(device.ProfileOrder)-1]
	}
	Debug("Stargazed and found %s", profileNow)
}

//export PowerPulse_SetProfile
func PowerPulse_SetProfile(profile *C.char) {
	go setProfile(C.GoString(profile))
}
func setProfile(profile string) {
	PowerPulse_Init()
	if device == nil {
		return
	}
	lock.Lock()
	defer lock.Unlock()
	Debug("Got past lock for setProfile(%s)", profile)

	//When device.SetProfile is called but device.ProfileLock is true, profileNow is still updated with the new request
	profileLast = profileNow

	if !bootedProfile && device.ProfileBoot != "" && device.ProfileBootDuration.String() != "" {
		Debug("Applying boot profile %s", device.ProfileBoot)
		duration, err := device.ProfileBootDuration.Int64()
		if err != nil {
			Error("Error applying boot profile %s for duration %s: %v", device.ProfileBoot, device.ProfileBootDuration, err)
			return
		}
		if err := device.SetProfile(device.ProfileBoot); err != nil {
			Error("Error applying boot profile %s: %v", device.ProfileBoot, err)
			return
		}
		bootedProfile = true
		if profile == "" {
			PowerPulse_Stargaze()
			profile = profileNow
		}
		if duration > 0 {
			device.ProfileLock = true
			Debug("Deferring profile %s for %d seconds", profile, duration)
			time.Sleep(time.Second * time.Duration(duration))
			device.ProfileLock = false
		}
	}

	//Check if we need to change course on which profile to apply
	if profileNow != profileLast {
		profile = profileNow
	}

	Info("Applying profile %s", profile)
	if err := device.SetProfile(profile); err != nil {
		Error("Error applying profile %s: %v", profile, err)
		return
	}
	profileNow = profile
	if err := device.CacheProfile(profile); err != nil {
		Warn("Failed to cache profile %s for reboot: %v", profile, err)
	}
}

//export PowerPulse_ResetProfile
func PowerPulse_ResetProfile() {
	go resetProfile()
}
func resetProfile() {
	PowerPulse_Init()
	if device == nil {
		return
	}
	lock.Lock()
	defer lock.Unlock()
	Debug("Got past lock for resetProfile()")

	if profileLast != "" {
		if err := device.SetProfile(profileLast); err != nil {
			Error("Error resetting to profile %s: %v", profileLast, err)
			return
		}
		profileTmp := profileNow
		profileNow = profileLast
		profileLast = profileTmp
	}
}

//export PowerPulse_SetInteractive
func PowerPulse_SetInteractive(interactive bool) {
	go setInteractive(interactive)
}
func setInteractive(interactive bool) {
	for inputName, input := range device.Paths.Inputs {
		if input.Path != "" {
			switch input.Type {
			case "touchkey", "touchscreen":
				if input.Enabled != "" {
					enabledPath := pathJoin(input.Path, input.Enabled)
					if interactive {
						Debug("Interacting with %s (%s)", inputName, input.Type)
						device.write(enabledPath, "1")
					} else {
						Debug("Not interacting with %s (%s)", inputName, input.Type)
						device.write(enabledPath, "0")
					}
				}
			}
		}
	}

	if profile := device.GetProfile("screen_off"); profile != nil {
		if interactive {
			Debug("Turning off screen off profile")
			resetProfile()
		} else {
			Debug("Turning on screen off profile")
			setProfile("screen_off")
		}
	}
}

//export PowerPulse_SetPowerHint
func PowerPulse_SetPowerHint(hint, data int32) {
	go setPowerHint(hint, data)
}
func setPowerHint(hint, data int32) {
	switch PowerHint(hint) {
	case HINT_VSYNC:
		if data > 0 {
			Debug("PowerHint: VSYNC: on")
			//TODO: device.Boosting = true; go device.BoostIf()
			device.Boost(16666) //1 frame @60Hz
		} else {
			Debug("PowerHint: VSYNC: off")
			//TODO: device.Boosting = false
		}
		return

	case HINT_INTERACTION:
		device.Boost(data * 1000)
		return

	case HINT_VIDEO_ENCODE:
		Debug("PowerHint: DEPRECATED: VIDEO_ENCODE: %d", data)
		return

	case HINT_VIDEO_DECODE:
		Debug("PowerHint: DEPRECATED: VIDEO_DECODE: %d", data)
		return

	case HINT_LOW_POWER:
		if profile := device.GetProfile("battery_saver"); profile != nil {
			if data > 0 {
				Debug("Turning on battery saver profile")
				setProfile("battery_saver")
			} else {
				Debug("Turning off battery saver profile")
				resetProfile()
			}
		}
		return

	case HINT_LAUNCH:
		if data > 0 {
			//TODO: device.Boosting = true; go device.BoostIf()
			device.Boost(3000000) //3 seconds
		} else {
			//TODO: device.Boosting = false
		}
		return

	case HINT_SUSTAINED_PERFORMANCE, HINT_VR_MODE,
		HINT_AUDIO_STREAMING, HINT_AUDIO_LOW_LATENCY,
		HINT_CAMERA_LAUNCH, HINT_CAMERA_STREAMING, HINT_CAMERA_SHOT,
		HINT_EXPENSIVE_RENDERING, HINT_LINEAGE_CPU_BOOST:
		if profile := device.GetProfile("performance"); profile != nil {
			if data > 0 {
				Debug("Turning on performance profile")
				setProfile("performance")
			} else {
				Debug("Turning off performance profile")
				resetProfile()
			}
		}
		return

	case HINT_LINEAGE_SET_PROFILE:
		switch data {
		case -1:
			setProfile("screen_off")
		case 0:
			setProfile("battery_saver")
		case 3:
			setProfile("efficiency")
		case 1:
			setProfile("balanced")
		case 4:
			setProfile("quick")
		case 2:
			setProfile("performance")
		}
		return
	}

	Debug("PowerHint: %d: %d (not supported)", hint, data)
}

//export PowerPulse_SetFeature
func PowerPulse_SetFeature(feature int32, activate bool) {
	switch PowerFeature(feature) {
	case FEATURE_DOUBLE_TAP_TO_WAKE: //POWER_FEATURE_DOUBLE_TAP_TO_WAKE
		Debug("SetFeature: POWER_FEATURE_DOUBLE_TAP_TO_WAKE: %t", activate)
		return
	}

	Debug("SetFeature: %d: %t (not supported)", feature, activate)
}

//export PowerPulse_GetFeature
func PowerPulse_GetFeature(feature int32) uint32 {
	switch PowerFeature(feature) {
	case FEATURE_DOUBLE_TAP_TO_WAKE:
		Debug("GetFeature: DOUBLE_TAP_TO_WAKE: true")
		return 1 //TODO: return from config

	case FEATURE_SUPPORTED_PROFILES:
		if len(device.ProfileOrder) > 0 {
			Debug("GetFeature: SUPPORTED_PROFILES: true")
			return uint32(len(device.ProfileOrder))
		}
		Debug("GetFeature: SUPPORTED_PROFILES: false")
		return 0
	}

	Debug("GetFeature: %d: (not supported)", feature)
	return 0
}

//export PowerPulse_Init
func PowerPulse_Init() {
	go initialize()
}
func initialize() {
	if booted {
		return
	}
	booted = true
	startTime := time.Now()

	Info("Need to boot PowerPulse first, just a blip...")
	reloadConfig()

	deltaTime := time.Now().Sub(startTime).Milliseconds()
	Info("PowerPulse finished init in %dms", deltaTime)
}

//export PowerPulse_ReloadConfig
func PowerPulse_ReloadConfig() {
	go reloadConfig()
}
func reloadConfig() {
	deviceJSON := make([]byte, 0)
	for i := 0; i < len(manifests); i++ {
		tmpJSON, err := ioutil.ReadFile(manifests[i])
		if err == nil && len(tmpJSON) > 0 {
			Info("Found manifest at %s", manifests[i])
			deviceJSON = tmpJSON
			break
		}
	}

	device = &Device{}
	if err := json.Unmarshal(deviceJSON, device); err != nil {
		Error("Error parsing manifest: %v", err)
		return
	}

	if device.Paths == nil {
		device.Paths = &Paths{}
	}
	if err := device.Paths.Init(); err != nil {
		Error("Error parsing paths from device manifest: %v", err)
		return
	}

	pathsJSON, err := json.Marshal(device.Paths)
	if err != nil {
		Debug("DEBUG: Error marshalling paths for print: %v", err)
		return
	}
	Debug(string(pathsJSON))

	if len(device.Profiles) < 1 {
		Error("Error reading device manifest: No profiles were found")
		return
	}

	for profileName := range device.Profiles {
		adjustedName := strings.ReplaceAll(strings.ToLower(profileName), " ", "_")
		if adjustedName != profileName {
			device.Profiles[adjustedName] = device.Profiles[profileName]
			delete(device.Profiles, profileName)
			Debug("Found profile %s as %s", profileName, adjustedName)
		} else {
			Debug("Found profile %s", adjustedName)
		}
	}

	if device.ProfileBoot != "" {
		device.ProfileBoot = strings.ReplaceAll(strings.ToLower(device.ProfileBoot), " ", "_")
	}

	if profileNow == "" {
		if device.ProfileBoot != "" {
			profileNow = strings.ReplaceAll(strings.ToLower(device.ProfileBoot), " ", "_")
		}
		if device.Paths.PowerPulse != nil && device.Paths.PowerPulse.Profile != "" {
			buffer, err := ioutil.ReadFile(device.Paths.PowerPulse.Profile)
			if err == nil && len(buffer) > 0 {
				if buffer[len(buffer)-1] == '\n' { buffer = buffer[:len(buffer)-1] }
				profileNow = strings.ReplaceAll(strings.ToLower(string(buffer)), " ", "_")
			}
		}
	} else {
		profileNow = strings.ReplaceAll(strings.ToLower(profileNow), " ", "_")
	}

	if device.ProfileInheritance == nil || len(device.ProfileInheritance) == 0 {
		Debug("No profile inheritance was specified")
		//Try to add any recognizable profiles
		pi := make([]string, 0)
		try := []string{"screen_off", "battery_saver", "efficiency", "balanced", "quick", "performance", "bootpulse"}
		for i := 0; i < len(try); i++ {
			if p := device.GetProfile(try[i]); p != nil {
				Debug("Found profile %s", try[i])
				pi = append(pi, try[i])
			}
		}
		if profileNow != "" {
			found := false
			for i := 0; i < len(pi); i++ {
				if pi[i] == profileNow {
					found = true
					break
				}
			}
			if !found {
				//Start with the configured boot profile, in case we inherit special settings (better to be safe than sorry!)
				pi = append([]string{profileNow}, pi...)
			}
		}
		device.ProfileInheritance = pi
	}
	Debug("Profile inheritance: %s", device.ProfileInheritance)

	if len(device.ProfileInheritance) > 0 {
		Debug("Expanding profile inheritance cache...")
		device.ProfilesInherited = device.GetInheritedProfiles()
	}

	if device.ProfileOrder == nil || len(device.ProfileOrder) == 0 {
		Debug("No profile order was specified")
		//Try to add any recognizable profiles
		po := make([]string, 0)
		try := []string{"battery_saver", "efficiency", "balanced", "quick", "performance"}
		for i := 0; i < len(try); i++ {
			if p := device.GetProfile(try[i]); p != nil {
				Debug("Found profile %s", try[i])
				po = append(po, try[i])
			}
		}
		if profileNow != "" {
			found := false
			for i := 0; i < len(po); i++ {
				if po[i] == profileNow {
					found = true
					break
				}
			}
			if !found {
				//Start with the configured boot profile, in case we inherit special settings (better to be safe than sorry!)
				po = append([]string{profileNow}, po...)
			}
		}
		device.ProfileOrder = po
	}
	if len(device.ProfileOrder) == 0 {
		Debug("No identifiable boot profile, please set your profile order and/or your boot profile!")
		Error("Error reading device manifest!")
		return
	}
	Debug("Profile order: %s", device.ProfileOrder)
}

func main() {
	debug = false
	verbose = false
	pflag.StringArrayVarP(&manifests, "manifest", "m", manifests, "path to device manifest(s),comma-separated")
	pflag.StringVarP(&profileNow, "profile", "p", profileNow, "profile override")
	pflag.BoolVarP(&debug, "debug", "d", debug, "debug mode")
	pflag.BoolVarP(&verbose, "verbose", "v", verbose, "verbose mode")
	pflag.Parse()

	initialize()
	stargaze()

	Info("Applying profile %s", profileNow)
	setProfile(profileNow)
}
