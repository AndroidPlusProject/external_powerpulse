package main

import (
	"encoding/json"
	"fmt"
	"time"
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
	profile := &Profile{}

	index := -1
	for i := 0; i < len(dev.ProfileInheritance); i++ {
		if dev.ProfileInheritance[i] == name {
			index = i
			break
		}
	}

	//Inherit any parent profiles if available
	if index > 0 {
		for i := 0; i < index; i++ {
			dev.getProfile(dev.ProfileInheritance[i], profile)
		}
	}
	dev.getProfile(name, profile)

	return profile
}

func (dev *Device) getProfile(name string, dst *Profile) {
	profile := dev.Profiles[name]
	if profile == nil {
		return
	}

	if dst.Clusters == nil {
		dst.Clusters = make(map[string]*Cluster)
	}
	for clusterName, cluster := range profile.Clusters {
		if _, exists := dst.Clusters[clusterName]; !exists {
			dst.Clusters[clusterName] = cluster
			continue
		}
		if dst.Clusters[clusterName].CPUFreq == nil {
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
			if dst.Clusters[clusterName].CPUFreq.Governors == nil {
				dst.Clusters[clusterName].CPUFreq.Governors = cluster.CPUFreq.Governors
			} else if cluster.CPUFreq.Governors != nil {
				for data, value := range cluster.CPUFreq.Governors {
					dst.Clusters[clusterName].CPUFreq.Governors[data] = value
				}
			}
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
		if debug {
			Debug("Loading CPU cluster %s", clusterName)
			Debug(clusterPath)
		}

		if cluster.CPUFreq != nil {
			freq := cluster.CPUFreq
			pathFreq := dev.Paths.Clusters[clusterName].CPUFreq
			freqPath := pathJoin(clusterPath, pathFreq.Path)
			if debug {
				Debug("Loading cpufreq %s", freqPath)
				Debug(freqPath)
			}
			if freq.Governor != "" {
				governorPath := pathJoin(freqPath, pathFreq.Governor)
				if debug {
					Debug("> CPUFreq > Governor = %s", freq.Governor)
					Debug(governorPath)
				}
				if freq.Governor == "powerpulse" {
					dev.BufferWrite(governorPath, "userspace")
				} else {
					dev.BufferWrite(governorPath, freq.Governor)
				}
			}
			max := freq.Max.String()
			if max != "" {
				Debug("> CPUFreq > Max = %s", max)
				maxPath := pathJoin(freqPath, pathFreq.Max)
				Debug(maxPath)
				dev.BufferWrite(maxPath, max)
			}
			min := freq.Min.String()
			if min != "" {
				minPath := pathJoin(freqPath, pathFreq.Min)
				if debug {
					Debug("> CPUFreq > Min = %s", min)
					Debug(minPath)
				}
				dev.BufferWrite(minPath, min)
			}
			speed := freq.Speed.String()
			if speed != "" {
				speedPath := pathJoin(freqPath, pathFreq.Speed)
				if debug {
					Debug("> CPUFreq > Speed = %s", speed)
					Debug(speedPath)
				}
				dev.BufferWrite(speedPath, speed)
			}
			for governorName, governor := range freq.Governors {
				governorPath := pathJoin(freqPath, governorName)
				if debug {
					Debug("Loading cpufreq governor %s", governorName)
					Debug(governorPath)
				}
				for arg, val := range governor {
					argPath := pathJoin(governorPath, arg)
					Debug(argPath)
					switch v := val.(type) {
					case bool:
						Debug("> %s > %s = %t", governorName, arg, v)
						if err := dev.BufferWriteBool(argPath, v); err != nil {return err}
					case float64:
						Debug("> %s > %s = %.0F", governorName, arg, v)
						dev.BufferWriteNumber(argPath, v)
					case string:
						Debug("> %s > %s = %s", governorName, arg, v)
						dev.BufferWrite(argPath, v)
					default:
						return fmt.Errorf("governor %s has invalid value type '%T' for arg %s", governorName, v, arg)
					}
				}
			}
		}
	}

	if profile.GPU != nil {
		gpu := profile.GPU
		gpuPath := dev.Paths.GPU.Path
		if debug {
			Debug("Loading GPU")
			Debug(gpuPath)
		}
		if gpu.DVFS != nil {
			dvfs := gpu.DVFS
			Debug("Loading GPU DVFS")
			governor := dvfs.Governor
			if governor != "" {
				governorPath := pathJoin(gpuPath, dev.Paths.GPU.DVFS.Governor)
				if debug {
					Debug("> GPU > DVFS > Governor = %s", governor)
					Debug(governorPath)
				}
				dev.BufferWrite(governorPath, governor)
			}
			max := dvfs.Max.String()
			if max != "" {
				maxPath := pathJoin(gpuPath, dev.Paths.GPU.DVFS.Max)
				if debug {
					Debug("> GPU > DVFS > Max = %s", max)
					Debug(maxPath)
				}
				dev.BufferWrite(maxPath, max)
			}
			min := dvfs.Min.String()
			if min != "" {
				minPath := pathJoin(gpuPath, dev.Paths.GPU.DVFS.Min)
				if debug {
					Debug("> GPU > DVFS > Min = %s", min)
					Debug(minPath)
				}
				dev.BufferWrite(minPath, min)
			}
		}
		if gpu.Highspeed != nil {
			hs := gpu.Highspeed
			Debug("Loading GPU highspeed")
			clock := hs.Clock.String()
			if clock != "" {
				clockPath := pathJoin(gpuPath, dev.Paths.GPU.Highspeed.Clock)
				if debug {
					Debug("> GPU > Highspeed > Clock = %s", clock)
					Debug(clockPath)
				}
				dev.BufferWrite(clockPath, clock)
			}
			load := hs.Load.String()
			if load != "" {
				loadPath := pathJoin(gpuPath, dev.Paths.GPU.Highspeed.Load)
				if debug {
					Debug("> GPU > Highspeed > Load = %s", load)
					Debug(loadPath)
				}
				dev.BufferWrite(loadPath, load)
			}
		}
	}

	if profile.Kernel != nil {
		krnl := profile.Kernel
		Debug("Loading kernel")
		if krnl.DynamicHotplug != nil {
			dynamicHotplugPath := dev.Paths.Kernel.DynamicHotplug
			if debug {
				Debug("> Kernel > Dynamic Hotplug = %t", krnl.DynamicHotplug)
				Debug(dynamicHotplugPath)
			}
			if err := dev.BufferWriteBool(dynamicHotplugPath, *krnl.DynamicHotplug); err != nil {return err}
		}
		if krnl.PowerEfficient != nil {
			powerEfficientPath := dev.Paths.Kernel.PowerEfficient
			if debug {
				Debug("> Kernel > Power Efficient = %t", krnl.PowerEfficient)
				Debug(powerEfficientPath)
			}
			if err := dev.BufferWriteBool(powerEfficientPath, *krnl.PowerEfficient); err != nil {return err}
		}
		if krnl.HMP != nil {
			hmp := krnl.HMP
			hmpPath := dev.Paths.Kernel.HMP.Path
			if debug {
				Debug("Loading kernel HMP")
				Debug(hmpPath)
			}
			hmpPaths := dev.Paths.Kernel.HMP
			if hmp.Boost != nil {
				boostPath := pathJoin(hmpPath, hmpPaths.Boost)
				if debug {
					Debug("> Kernel > HMP > Boost = %t", hmp.Boost)
					Debug(boostPath)
				}
				if err := dev.BufferWriteBool(boostPath, *hmp.Boost); err != nil {return err}
			}
			if hmp.Semiboost != nil {
				semiboostPath := pathJoin(hmpPath, hmpPaths.Semiboost)
				if debug {
					Debug("> Kernel > HMP > Semiboost = %t", hmp.Semiboost)
					Debug(semiboostPath)
				}
				if err := dev.BufferWriteBool(semiboostPath, *hmp.Semiboost); err != nil {return err}
			}
			if hmp.ActiveDownMigration != nil {
				activeDownMigrationPath := pathJoin(hmpPath, hmpPaths.ActiveDownMigration)
				if debug {
					Debug("> Kernel > HMP > Active Down Migration = %t", hmp.ActiveDownMigration)
					Debug(activeDownMigrationPath)
				}
				if err := dev.BufferWriteBool(activeDownMigrationPath, *hmp.ActiveDownMigration); err != nil {return err}
			}
			if hmp.AggressiveUpMigration != nil {
				aggressiveUpMigrationPath := pathJoin(hmpPath, hmpPaths.AggressiveUpMigration)
				if debug {
					Debug("> Kernel > HMP > Aggressive Up Migration = %t", hmp.AggressiveUpMigration)
					Debug(aggressiveUpMigrationPath)
				}
				if err := dev.BufferWriteBool(aggressiveUpMigrationPath, *hmp.AggressiveUpMigration); err != nil {return err}
			}
			if hmp.Threshold != nil {
				thld := hmp.Threshold
				down := thld.Down.String()
				if down != "" {
					downPath := pathJoin(hmpPath, hmpPaths.Threshold.Down)
					if debug {
						Debug("> Kernel > HMP > Threshold > Down = %s", down)
						Debug(downPath)
					}
					dev.BufferWrite(downPath, down)
				}
				up := thld.Up.String()
				if up != "" {
					upPath := pathJoin(hmpPath, hmpPaths.Threshold.Up)
					if debug {
						Debug("> Kernel > HMP > Threshold > Up = %s", up)
						Debug(upPath)
					}
					dev.BufferWrite(upPath, up)
				}
			}
			if hmp.SbThreshold != nil {
				thld := hmp.SbThreshold
				down := thld.Down.String()
				if down != "" {
					downPath := pathJoin(hmpPath, hmpPaths.SbThreshold.Down)
					if debug {
						Debug("> Kernel > HMP > Semiboost Threshold > Down = %s", down)
						Debug(downPath)
					}
					dev.BufferWrite(downPath, down)
				}
				up := thld.Up.String()
				if up != "" {
					upPath := pathJoin(hmpPath, hmpPaths.SbThreshold.Up)
					if debug {
						Debug("> Kernel > HMP > Semiboost Threshold > Up = %s", up)
						Debug(upPath)
					}
					dev.BufferWrite(upPath, up)
				}
			}
		}
	}

	if profile.IPA != nil {
		ipa := profile.IPA
		ipaPaths := dev.Paths.IPA
		ipaPath := ipaPaths.Path
		if ipa.Enabled != nil {
			enabledPath := pathJoin(ipaPath, ipaPaths.Enabled)
			if debug {
				Debug("Loading IPA")
				Debug("> IPA > Enabled = %t", ipa.Enabled)
				Debug(enabledPath)
			}
			if err := dev.BufferWriteBool(enabledPath, *ipa.Enabled); err != nil {return err}
			if *ipa.Enabled {
				controlTemp := ipa.ControlTemp.String()
				if controlTemp != "" {
					ctPath := pathJoin(ipaPath, ipaPaths.ControlTemp)
					if debug {
						Debug("> IPA > Control Temp = %s", controlTemp)
						Debug(ctPath)
					}
					dev.BufferWrite(ctPath, controlTemp)
				}
			}
		}
	}

	if profile.InputBooster != nil {
		ib := profile.InputBooster
		ibPaths := dev.Paths.InputBooster
		if debug {
			Debug("Loading input booster")
		}
		if ib.Head != "" {
			headPath := ibPaths.Head
			if debug {
				Debug("> Input Booster > Head = %s", ib.Head)
				Debug(headPath)
			}
			dev.BufferWrite(headPath, ib.Head)
		}
		if ib.Tail != "" {
			tailPath := ibPaths.Tail
			if debug {
				Debug("> Input Booster > Tail = %s", ib.Tail)
				Debug(tailPath)
			}
			dev.BufferWrite(tailPath, ib.Tail)
		}
	}

	if profile.SecSlow != nil {
		slow := profile.SecSlow
		slowPaths := dev.Paths.SecSlow
		if slow.Enabled != nil {
			enabledPath := slowPaths.Enabled
			if debug {
				Debug("Loading sec_slow")
				Debug("> sec_slow > Enabled = %t", slow.Enabled)
				Debug(enabledPath)
			}
			if err := dev.BufferWriteBool(enabledPath, *slow.Enabled); err != nil {return err}
			if *slow.Enabled {
				if slow.Enforced != nil {
					enforcedPath := slowPaths.Enforced
					if debug {
						Debug("> sec_slow > Enforced = %t", slow.Enforced)
						Debug(enforcedPath)
					}
					if err := dev.BufferWriteBool(enforcedPath, *slow.Enforced); err != nil {return err}
				}
				timerRate := slow.TimerRate.String()
				if timerRate != "" {
					timerRatePath := slowPaths.TimerRate
					if debug {
						Debug("> sec_slow > Timer Rate = %s", timerRate)
						Debug(timerRatePath)
					}
					dev.BufferWrite(timerRatePath, timerRate)
				}
			}
		}
	}

	return nil
}
