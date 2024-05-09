package main

import (
	"encoding/json"
	"fmt"
	"time"

	jsone "github.com/go-json-experiment/json"
)

type Profile struct {
	Clusters map[string]*Cluster
	CPUSets map[string]*CPUSet
	GPU *GPU
	Kernel *Kernel
	IPA *IPA
	InputBooster *InputBooster
	SecSlow *SecSlow
}

type Cluster struct {
	CPUFreq *CPUFreq
}

type CPUFreq struct {
	Max json.Number
	Min json.Number
	Speed json.Number
	Governor string
	Governors map[string]map[string]interface{} //"interactive":{"arg":0,"arg2":"val"},"performance":{"arg":true}
}

type CPUSet struct {
	CPUs string
	CPUExclusive *bool `json:"cpu_exclusive"`
}

type GPU struct {
	DVFS *DVFS
	Highspeed *GPUHighspeed
}

type DVFS struct {
	Governor string
	Max json.Number
	Min json.Number
}

type GPUHighspeed struct {
	Clock json.Number
	Load json.Number
}

type Kernel struct {
	DynamicHotplug *bool
	PowerEfficient *bool
	HMP *KernelHMP
}

type KernelHMP struct {
	Boost *bool
	Semiboost *bool
	ActiveDownMigration *bool
	AggressiveUpMigration *bool
	Threshold *KernelHMPThreshold
	SbThreshold *KernelHMPThreshold
}

type KernelHMPThreshold struct {
	Down json.Number
	Up json.Number
}

type IPA struct {
	Enabled *bool
	ControlTemp json.Number
}

type InputBooster struct {
	Head string
	Tail string
}

type SecSlow struct {
	Enabled *bool
	Enforced *bool
	TimerRate json.Number
}

func (dev *Device) CacheProfile(name string) error {
	found := false
	for i := 0; i < len(dev.ProfileOrder); i++ {
		if dev.ProfileOrder[i] == name {
			found = true
			break
		}
	}
	if !found {
		return nil //Silently fail, we only want to cache profiles we can pick from
	}
	if dev.Paths.PowerPulse != nil && dev.Paths.PowerPulse.Profile != "" {
		return dev.write(dev.Paths.PowerPulse.Profile, name)
	}
	return nil
}

func (dev *Device) SyncProfile() error {
	Debug("Syncing profile")
	if dev.Buffered == nil {return nil}
	for i := 0; i < len(dev.Buffered); i++ {
		bw := dev.Buffered[i]
		if err := dev.write(bw.Path, bw.Data); err != nil {return err}
	}

	//Reset the buffer for the next profile chain
	dev.Buffered = make([]BufferedWrite, 0)

	//Start threads for any services that we control
	profile := dev.GetProfileNow()
	if profile == nil {
		return fmt.Errorf("failed to find current profile after syncing writes")
	}
	for clusterName, cluster := range profile.Clusters {
		if cluster.CPUFreq.Governor == "powerpulse" {
			go dev.GovernCPU(clusterName)
		}
	}

	return nil
}

func (dev *Device) GetProfile(name string) *Profile {
	if _, exists := dev.ProfilesInherited[name]; exists {
		return dev.ProfilesInherited[name]
	}
	if _, exists := dev.Profiles[name]; exists {
		return dev.Profiles[name]
	}
	return nil
}

func (dev *Device) GetInheritedProfiles() map[string]*Profile {
	profiles := make(map[string]*Profile)

	//Store a copy of each profile after inheritance from the previous one
	for i := 0; i < len(dev.ProfileInheritance); i++ {
		Debug("Creating inherited profile %d: %s", i, dev.ProfileInheritance[i])
		var profile Profile //Start with a blank profile structure
		if i > 0 {
			Debug("Copying previous profile %d: %s", i-1, dev.ProfileInheritance[i-1])
			tmp := profiles[dev.ProfileInheritance[i-1]]
			bytes, err := jsone.Marshal(tmp)
			if err != nil {
				Error("Marshal: %v", err)
				return nil
			}
			if err := jsone.Unmarshal(bytes, &profile); err != nil {
				Error("Unmarshal: %v", err)
				return nil
			}
		}
		dev.getProfile(dev.ProfileInheritance[i], &profile) //Install the new profile inheritance overlay
		profiles[dev.ProfileInheritance[i]] = &profile //Store the expanded profile with a new reference
	}

	return profiles
}

func (dev *Device) getProfile(name string, dst *Profile) {
	Debug("Inheriting profile %s", name)
	profile := dev.Profiles[name]
	if profile == nil {
		Error("Nil profile %s", name)
		return
	}

	if dst.Clusters == nil {
		dst.Clusters = make(map[string]*Cluster)
	}
	for clusterName, cluster := range profile.Clusters {
		//("%s: Working on cluster %s", name, clusterName)
		if _, exists := dst.Clusters[clusterName]; !exists {
			//Debug("%s: Cluster data missing, copying and continuing", name)
			dst.Clusters[clusterName] = cluster
			continue
		}
		if dst.Clusters[clusterName].CPUFreq == nil {
			//Debug("%s: CPUFreq data missing, copying and continuing", name)
			dst.Clusters[clusterName].CPUFreq = cluster.CPUFreq
			continue
		}
		if cluster.CPUFreq != nil {
			if cluster.CPUFreq.Max.String() != "" {
				dst.Clusters[clusterName].CPUFreq.Max = cluster.CPUFreq.Max
			}
			if cluster.CPUFreq.Min.String() != "" {
				dst.Clusters[clusterName].CPUFreq.Min = cluster.CPUFreq.Min
			}
			if cluster.CPUFreq.Speed.String() != "" {
				dst.Clusters[clusterName].CPUFreq.Speed = cluster.CPUFreq.Speed
			}
			if cluster.CPUFreq.Governor != "" {
				dst.Clusters[clusterName].CPUFreq.Governor = cluster.CPUFreq.Governor
			}
			govs := dst.Clusters[clusterName].CPUFreq.Governors
			if govs == nil {
				//Debug("%s: Using blank governors", name)
				govs = make(map[string]map[string]interface{})
			}
			for governorName, governorData := range cluster.CPUFreq.Governors {
				govData := make(map[string]interface{})
				if _, exists := govs[governorName]; exists {
					//Debug("%s: Reusing inherited data for %s", name, governorName)
					govData = govs[governorName]
				}
				for arg, val := range governorData {
					//Debug("%s: %s -> %s = %v", name, governorName, arg, val)
					govData[arg] = val.(interface{})
				}
				govs[governorName] = govData
			}
			dst.Clusters[clusterName].CPUFreq.Governors = govs
		}
	}

	if dst.CPUSets == nil {
		dst.CPUSets = make(map[string]*CPUSet)
	}
	for setName, set := range profile.CPUSets {
		if _, exists := dst.CPUSets[setName]; exists {
			if set.CPUs != "" {
				dst.CPUSets[setName].CPUs = set.CPUs
			}
			if set.CPUExclusive != nil {
				dst.CPUSets[setName].CPUExclusive = set.CPUExclusive
			}
		} else {
			dst.CPUSets[setName] = set
		}
	}

	if dst.GPU == nil {
		dst.GPU = profile.GPU
	} else if profile.GPU != nil {
		if profile.GPU.DVFS != nil {
			if dst.GPU.DVFS == nil {
				dst.GPU.DVFS = profile.GPU.DVFS
			} else {
				if profile.GPU.DVFS.Governor != "" {
					dst.GPU.DVFS.Governor = profile.GPU.DVFS.Governor
				}
				if profile.GPU.DVFS.Max.String() != "" {
					dst.GPU.DVFS.Max = profile.GPU.DVFS.Max
				}
				if profile.GPU.DVFS.Min.String() != "" {
					dst.GPU.DVFS.Min = profile.GPU.DVFS.Min
				}
			}
		}
		if profile.GPU.Highspeed != nil {
			if dst.GPU.Highspeed == nil {
				dst.GPU.Highspeed = profile.GPU.Highspeed
			} else {
				if profile.GPU.Highspeed.Clock.String() != "" {
					dst.GPU.Highspeed.Clock = profile.GPU.Highspeed.Clock
				}
				if profile.GPU.Highspeed.Load.String() != "" {
					dst.GPU.Highspeed.Load = profile.GPU.Highspeed.Load
				}
			}
		}
	}

	if dst.Kernel == nil {
		dst.Kernel = profile.Kernel
	} else if profile.Kernel != nil {
		if profile.Kernel.DynamicHotplug != nil {
			dst.Kernel.DynamicHotplug = profile.Kernel.DynamicHotplug
		}
		if profile.Kernel.PowerEfficient != nil {
			dst.Kernel.PowerEfficient = profile.Kernel.PowerEfficient
		}
		if profile.Kernel.HMP != nil {
			if dst.Kernel.HMP == nil {
				dst.Kernel.HMP = profile.Kernel.HMP
			} else {
				if profile.Kernel.HMP.Boost != nil {
					dst.Kernel.HMP.Boost = profile.Kernel.HMP.Boost
				}
				if profile.Kernel.HMP.Semiboost != nil {
					dst.Kernel.HMP.Semiboost = profile.Kernel.HMP.Semiboost
				}
				if profile.Kernel.HMP.ActiveDownMigration != nil {
					dst.Kernel.HMP.ActiveDownMigration = profile.Kernel.HMP.ActiveDownMigration
				}
				if profile.Kernel.HMP.AggressiveUpMigration != nil {
					dst.Kernel.HMP.AggressiveUpMigration = profile.Kernel.HMP.AggressiveUpMigration
				}
				if profile.Kernel.HMP.Threshold != nil {
					if dst.Kernel.HMP.Threshold == nil {
						dst.Kernel.HMP.Threshold = profile.Kernel.HMP.Threshold
					} else {
						if profile.Kernel.HMP.Threshold.Down.String() != "" {
							dst.Kernel.HMP.Threshold.Down = profile.Kernel.HMP.Threshold.Down
						}
						if profile.Kernel.HMP.Threshold.Up.String() != "" {
							dst.Kernel.HMP.Threshold.Up = profile.Kernel.HMP.Threshold.Up
						}
					}
				}
				if profile.Kernel.HMP.SbThreshold != nil {
					if dst.Kernel.HMP.SbThreshold == nil {
						dst.Kernel.HMP.SbThreshold = profile.Kernel.HMP.SbThreshold
					} else {
						if profile.Kernel.HMP.SbThreshold.Down.String() != "" {
							dst.Kernel.HMP.SbThreshold.Down = profile.Kernel.HMP.SbThreshold.Down
						}
						if profile.Kernel.HMP.SbThreshold.Up.String() != "" {
							dst.Kernel.HMP.SbThreshold.Up = profile.Kernel.HMP.SbThreshold.Up
						}
					}
				}
			}
		}
	}

	if dst.IPA == nil {
		dst.IPA = profile.IPA
	} else if profile.IPA != nil {
		if profile.IPA.Enabled != nil {
			dst.IPA.Enabled = profile.IPA.Enabled
		}
		if profile.IPA.ControlTemp.String() != "" {
			dst.IPA.ControlTemp = profile.IPA.ControlTemp
		}
	}

	if dst.InputBooster == nil {
		dst.InputBooster = profile.InputBooster
	} else if profile.InputBooster != nil {
		if profile.InputBooster.Head != "" {
			dst.InputBooster.Head = profile.InputBooster.Head
		}
		if profile.InputBooster.Tail != "" {
			dst.InputBooster.Tail = profile.InputBooster.Tail
		}
	}

	if dst.SecSlow == nil {
		dst.SecSlow = profile.SecSlow
	} else if profile.SecSlow != nil {
		if profile.SecSlow.Enabled != nil {
			dst.SecSlow.Enabled = profile.SecSlow.Enabled
		}
		if profile.SecSlow.Enforced != nil {
			dst.SecSlow.Enforced = profile.SecSlow.Enforced
		}
		if profile.SecSlow.TimerRate.String() != "" {
			dst.SecSlow.TimerRate = profile.SecSlow.TimerRate
		}
	}
}

func (dev *Device) GetProfileNow() *Profile {
	return dev.GetProfile(dev.Profile)
}

func (dev *Device) SetProfile(name string) error {
	//Save the requested profile if we're locked out
	if dev.ProfileLock {
		profileNow = name
		return fmt.Errorf("not allowed to set %s yet, locked to %s", name, dev.Profile)
	}

	dev.ProfileMutex.Lock()
	dev.ProfileLock = true
	defer func() {
		dev.ProfileLock = false
		dev.ProfileMutex.Unlock()
	}()

	startTime := time.Now()
	profile := dev.GetProfile(name)
	if profile == nil {
		return fmt.Errorf("profile %s does not exist", name)
	}
	dev.Profile = name

	//Set the new profile and sync it live
	if err := dev.setProfile(profile, name); err != nil {return err}
	if err := dev.SyncProfile(); err != nil {return err}

	//Handle cpusets separately for safety reasons
	if err := dev.setCpusets(profile); err != nil {return err}

	deltaTime := time.Now().Sub(startTime).Milliseconds()
	Info("PowerPulse finished applying %s in %dms", name, deltaTime)
	return nil
}

func (dev *Device) setProfile(profile *Profile, name string) error {
	for clusterName, cluster := range profile.Clusters {
		pathCluster := dev.Paths.Clusters[clusterName]
		clusterPath := pathCluster.Path
		Debug("Loading CPU cluster %s", clusterName)
		cpuPath := pathJoin(clusterPath, pathCluster.CPU)
		Debug("Loading CPU node %s", pathCluster.CPU)

		if cluster.CPUFreq != nil {
			freq := cluster.CPUFreq
			pathFreq := dev.Paths.Clusters[clusterName].CPUFreq
			freqPath := pathJoin(cpuPath, pathFreq.Path)
			Debug("Loading cpufreq %s", freqPath)
			if freq.Governor != "" {
				governorPath := pathJoin(freqPath, pathFreq.Governor)
				Debug("> CPUFreq > Governor = %s", freq.Governor)
				if freq.Governor == "powerpulse" {
					dev.BufferWrite(governorPath, "userspace")
				} else {
					dev.BufferWrite(governorPath, freq.Governor)
				}
				if governor, exists := freq.Governors[freq.Governor]; exists && governor != nil {
					governorPath := pathJoin(freqPath, freq.Governor)
					Debug("Loading cpufreq governor %s", freq.Governor)
					for arg, val := range governor {
						argPath := pathJoin(governorPath, arg)
						switch v := val.(type) {
						case bool:
							Debug("> %s > %s = %t", freq.Governor, arg, v)
							if err := dev.BufferWriteBool(argPath, v); err != nil {return err}
						case float64:
							Debug("> %s > %s = %.0F", freq.Governor, arg, v)
							dev.BufferWriteNumber(argPath, v)
						case string:
							Debug("> %s > %s = %s", freq.Governor, arg, v)
							dev.BufferWrite(argPath, v)
						default:
							return fmt.Errorf("governor %s has invalid value type '%T' for arg %s", freq.Governor, v, arg)
						}
					}
				}
			}
			max := freq.Max.String()
			if max != "" {
				maxPath := pathJoin(freqPath, pathFreq.Max)
				Debug("> CPUFreq > Max = %s", max)
				dev.BufferWrite(maxPath, max)
			}
			min := freq.Min.String()
			if min != "" {
				minPath := pathJoin(freqPath, pathFreq.Min)
				Debug("> CPUFreq > Min = %s", min)
				dev.BufferWrite(minPath, min)
			}
			speed := freq.Speed.String()
			if speed != "" {
				speedPath := pathJoin(freqPath, pathFreq.Speed)
				Debug("> CPUFreq > Speed = %s", speed)
				dev.BufferWrite(speedPath, speed)
			}
		}
	}

	if profile.GPU != nil {
		gpu := profile.GPU
		gpuPath := dev.Paths.GPU.Path
		Debug("Loading GPU")
		if gpu.DVFS != nil {
			dvfs := gpu.DVFS
			Debug("Loading GPU DVFS")
			governor := dvfs.Governor
			if governor != "" {
				governorPath := pathJoin(gpuPath, dev.Paths.GPU.DVFS.Governor)
				Debug("> GPU > DVFS > Governor = %s", governor)
				dev.BufferWrite(governorPath, governor)
			}
			max := dvfs.Max.String()
			if max != "" {
				maxPath := pathJoin(gpuPath, dev.Paths.GPU.DVFS.Max)
				Debug("> GPU > DVFS > Max = %s", max)
				dev.BufferWrite(maxPath, max)
			}
			min := dvfs.Min.String()
			if min != "" {
				minPath := pathJoin(gpuPath, dev.Paths.GPU.DVFS.Min)
				Debug("> GPU > DVFS > Min = %s", min)
				dev.BufferWrite(minPath, min)
			}
		}
		if gpu.Highspeed != nil {
			hs := gpu.Highspeed
			Debug("Loading GPU highspeed")
			clock := hs.Clock.String()
			if clock != "" {
				clockPath := pathJoin(gpuPath, dev.Paths.GPU.Highspeed.Clock)
				Debug("> GPU > Highspeed > Clock = %s", clock)
				dev.BufferWrite(clockPath, clock)
			}
			load := hs.Load.String()
			if load != "" {
				loadPath := pathJoin(gpuPath, dev.Paths.GPU.Highspeed.Load)
				Debug("> GPU > Highspeed > Load = %s", load)
				dev.BufferWrite(loadPath, load)
			}
		}
	}

	if profile.Kernel != nil {
		krnl := profile.Kernel
		Debug("Loading kernel")
		if krnl.DynamicHotplug != nil {
			dynamicHotplugPath := dev.Paths.Kernel.DynamicHotplug
			Debug("> Kernel > Dynamic Hotplug = %t", krnl.DynamicHotplug)
			if err := dev.BufferWriteBool(dynamicHotplugPath, *krnl.DynamicHotplug); err != nil {return err}
		}
		if krnl.PowerEfficient != nil {
			powerEfficientPath := dev.Paths.Kernel.PowerEfficient
			Debug("> Kernel > Power Efficient = %t", krnl.PowerEfficient)
			if err := dev.BufferWriteBool(powerEfficientPath, *krnl.PowerEfficient); err != nil {return err}
		}
		if krnl.HMP != nil {
			hmp := krnl.HMP
			hmpPath := dev.Paths.Kernel.HMP.Path
			Debug("Loading kernel HMP")
			hmpPaths := dev.Paths.Kernel.HMP
			if hmp.Boost != nil {
				boostPath := pathJoin(hmpPath, hmpPaths.Boost)
				Debug("> Kernel > HMP > Boost = %t", hmp.Boost)
				if err := dev.BufferWriteBool(boostPath, *hmp.Boost); err != nil {return err}
			}
			if hmp.Semiboost != nil {
				semiboostPath := pathJoin(hmpPath, hmpPaths.Semiboost)
				Debug("> Kernel > HMP > Semiboost = %t", hmp.Semiboost)
				if err := dev.BufferWriteBool(semiboostPath, *hmp.Semiboost); err != nil {return err}
			}
			if hmp.ActiveDownMigration != nil {
				activeDownMigrationPath := pathJoin(hmpPath, hmpPaths.ActiveDownMigration)
				Debug("> Kernel > HMP > Active Down Migration = %t", hmp.ActiveDownMigration)
				if err := dev.BufferWriteBool(activeDownMigrationPath, *hmp.ActiveDownMigration); err != nil {return err}
			}
			if hmp.AggressiveUpMigration != nil {
				aggressiveUpMigrationPath := pathJoin(hmpPath, hmpPaths.AggressiveUpMigration)
				Debug("> Kernel > HMP > Aggressive Up Migration = %t", hmp.AggressiveUpMigration)
				if err := dev.BufferWriteBool(aggressiveUpMigrationPath, *hmp.AggressiveUpMigration); err != nil {return err}
			}
			if hmp.Threshold != nil {
				thld := hmp.Threshold
				down := thld.Down.String()
				if down != "" {
					downPath := pathJoin(hmpPath, hmpPaths.Threshold.Down)
					Debug("> Kernel > HMP > Threshold > Down = %s", down)
					dev.BufferWrite(downPath, down)
				}
				up := thld.Up.String()
				if up != "" {
					upPath := pathJoin(hmpPath, hmpPaths.Threshold.Up)
					Debug("> Kernel > HMP > Threshold > Up = %s", up)
					dev.BufferWrite(upPath, up)
				}
			}
			if hmp.SbThreshold != nil {
				thld := hmp.SbThreshold
				down := thld.Down.String()
				if down != "" {
					downPath := pathJoin(hmpPath, hmpPaths.SbThreshold.Down)
					Debug("> Kernel > HMP > Semiboost Threshold > Down = %s", down)
					dev.BufferWrite(downPath, down)
				}
				up := thld.Up.String()
				if up != "" {
					upPath := pathJoin(hmpPath, hmpPaths.SbThreshold.Up)
					Debug("> Kernel > HMP > Semiboost Threshold > Up = %s", up)
					dev.BufferWrite(upPath, up)
				}
			}
		}
	}

	if profile.IPA != nil {
		Debug("Loading IPA")
		ipa := profile.IPA
		ipaPaths := dev.Paths.IPA
		ipaPath := ipaPaths.Path
		if ipa.Enabled != nil {
			Debug("> IPA > Enabled = %t", *ipa.Enabled)
			enabledPath := pathJoin(ipaPath, ipaPaths.Enabled)
			if err := dev.BufferWriteBool(enabledPath, *ipa.Enabled); err != nil {return err}
			if *ipa.Enabled {
				controlTemp := ipa.ControlTemp.String()
				if controlTemp != "" {
					ctPath := pathJoin(ipaPath, ipaPaths.ControlTemp)
					Debug("> IPA > Control Temp = %s", controlTemp)
					dev.BufferWrite(ctPath, controlTemp)
				}
			}
		}
	}

	if profile.InputBooster != nil {
		ib := profile.InputBooster
		ibPaths := dev.Paths.InputBooster
		Debug("Loading input booster")
		if ib.Head != "" {
			headPath := ibPaths.Head
			Debug("> Input Booster > Head = %s", ib.Head)
			dev.BufferWrite(headPath, ib.Head)
		}
		if ib.Tail != "" {
			tailPath := ibPaths.Tail
			Debug("> Input Booster > Tail = %s", ib.Tail)
			dev.BufferWrite(tailPath, ib.Tail)
		}
	}

	if profile.SecSlow != nil {
		slow := profile.SecSlow
		slowPaths := dev.Paths.SecSlow
		if slow.Enabled != nil {
			enabledPath := slowPaths.Enabled
			Debug("Loading sec_slow")
			Debug("> sec_slow > Enabled = %t", slow.Enabled)
			if err := dev.BufferWriteBool(enabledPath, *slow.Enabled); err != nil {return err}
			if *slow.Enabled {
				if slow.Enforced != nil {
					enforcedPath := slowPaths.Enforced
					Debug("> sec_slow > Enforced = %t", slow.Enforced)
					if err := dev.BufferWriteBool(enforcedPath, *slow.Enforced); err != nil {return err}
				}
				timerRate := slow.TimerRate.String()
				if timerRate != "" {
					timerRatePath := slowPaths.TimerRate
					Debug("> sec_slow > Timer Rate = %s", timerRate)
					dev.BufferWrite(timerRatePath, timerRate)
				}
			}
		}
	}

	return nil
}