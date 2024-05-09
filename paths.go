package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

type Paths struct {
	PowerPulse *PathsPowerPulse
	Clusters map[string]PathsCluster
	Cpusets *PathsCpusets
	IPA *PathsIPA
	GPU *PathsGPU
	Kernel *PathsKernel
	InputBooster *PathsInputBooster
	SecSlow *PathsSecSlow
	Inputs map[string]PathsInput
}

type PathsPowerPulse struct {
	Profile string //universal7420: /data/adb/powerpulse.profile
}

type PathsCluster struct { //universal7420: apollo, atlas
	Path string //universal7420: /sys/devices/system/cpu
	CPU string //universal7420: cpu0 for apollo, cpu4 for atlas
	CPUFreq *PathsCPUFreq
}

type PathsCPUFreq struct {
	Path string //universal7420: apollo: cpu0/cpufreq, atlas: cpu4/cpufreq
	Governor string //universal7420: scaling_governor
	Governors string //universal7420: scaling_available_governors
	Max string //universal7420: scaling_max_freq
	Min string //universal7420: scaling_min_freq
	Speed string //universal7420: scaling_setspeed
	Stats *PathsCPUFreqStats
}

type PathsCPUFreqStats struct {
	Path string //universal7420: stats
	TimeInState string //universal7420: time_in_state
	TotalTrans string //universal7420: total_trans
}

type PathsCpusets struct {
	Path string //universal7420: /dev/cpuset
	Sets map[string]PathsCpuset //universal7420: background, foreground, restricted, system-background, top-app
}

type PathsCpuset struct {
	CPUs string //universal7420: cpus
	CPUExclusive string //universal7420: cpu_exclusive
}

type PathsIPA struct {
	Path string //universal7420: /sys/power/ipa
	Enabled string //universal7420: enabled
	ControlTemp string //universal7420: control_temp
}

type PathsGPU struct {
	Path string //universal7420: /sys/devices/14ac0000.mali
	DVFS *PathsGPUDVFS
	Highspeed *PathsGPUHighspeed
}

type PathsGPUDVFS struct {
	Governor string //universal7420: dvfs_governor
	Max string //universal7420: dvfs_max_lock
	Min string //universal7420: dvfs_min_lock
}

type PathsGPUHighspeed struct {
	Clock string //universal7420: highspeed_clock
	Load string //universal7420: highspeed_load
}

type PathsKernel struct {
	DynamicHotplug string //universal7420: /sys/power/enable_dm_hotplug
	PowerEfficient string //universal7420: /sys/modules/workqueue/parameters/power_efficient
	HMP *PathsKernelHMP
}

type PathsKernelHMP struct {
	Path string //universal7420: /sys/kernel/hmp
	Boost string //universal7420: boost
	Semiboost string //universal7420: semiboost
	ActiveDownMigration string //universal7420: active_down_migration
	AggressiveUpMigration string //universal7420: aggressive_up_migration
	Threshold *PathsKernelHMPThreshold
	SbThreshold *PathsKernelHMPThreshold
}

type PathsKernelHMPThreshold struct {
	Down string
	Up string
}

type PathsInputBooster struct {
	Path string //universal7420: /sys/class/input_booster
	Head string //universal7420: head
	Tail string //universal7420: tail
}

type PathsSecSlow struct {
	Path string //universal7420: /sys/devices/virtual/sec/sec_slow
	Enabled string //universal7420: slow_mode
	Enforced string //universal7420: enforced_slow_mode
	TimerRate string //universal7420: timer_rate
}

type PathsInput struct {
	Enabled string //universal7420: enabled
	Path string //universal7420: /sys/class/input/input*
	Type string //universal7420: "touchkey" or "touchscreen" for now, used to toggle input states when told to
}

func (p *Paths) Init() error {
	if p.Clusters != nil && len(p.Clusters) > 0 {
		for clusterName, _ := range p.Clusters {
			cluster := p.Clusters[clusterName]
			clusterPath, err := pathOrStockMustExist(&cluster.Path, GetPaths_Cluster)
			if err != nil {
				Error("Error finding path for cluster %s: %v", clusterName, err)
				//Cluster defined in manifest paths, require a valid path to be available
				return pathErrorDefinition("clusters/%s/path", clusterName)
			}
			Debug("Chose cluster path: %s", cluster.Path)
			if cluster.CPU == "" {
				//Cluster defined in manifest paths, requirqe a valid CPU node to use as the parent node
				return pathErrorDefinition("clusters/%s/cpu", clusterName)
			}
			cpuPath := pathJoin(clusterPath, cluster.CPU)
			if !pathValid(cpuPath) {
				return pathErrorInvalid("clusters/%s/cpu", clusterName)
			}
			CachePaths_Cluster(cluster.Path)
			Debug("Chose cluster node: %s", cluster.CPU)

			freq := cluster.CPUFreq
			freqPath := ""
			if freq == nil {
				freq = &PathsCPUFreq{}
				freqPath, _ = GetPaths_CPUFreq(cpuPath)
				if freqPath != "" {
					Debug("Chose cpufreq path: %s", freqPath)
					freq.Path = freqPath
					freqPath = pathJoin(cpuPath, freqPath)
					freq.Governor, _ = GetPaths_CPUFreq_Governor(freqPath)
					freq.Governors, _ = GetPaths_CPUFreq_Governors(freqPath)
					freq.Max, _ = GetPaths_CPUFreq_Max(freqPath)
					freq.Min, _ = GetPaths_CPUFreq_Min(freqPath)
					freq.Speed, _ = GetPaths_CPUFreq_Speed(freqPath)
				}
			} else {
				freqPath, err := pathOrStockMustExist(&freq.Path, GetPaths_CPUFreq, cluster.CPU)
				if err != nil {
					//CPUFreq defined in manifest paths, require a valid path to be available
					return pathErrorDefinition("clusters/%s/cpufreq relative to path %s", clusterName, cluster.CPU)
				}
				Info("Found cpufreq path: %s", freqPath)
				if err := pathMustOrStockCanExist(&freq.Governor, GetPaths_CPUFreq_Governor, freqPath); err != nil {
					return pathErrorInvalid(freq.Governor, "clusters/%s/cpufreq/governor", clusterName)
				}
				if err := pathMustOrStockCanExist(&freq.Governors, GetPaths_CPUFreq_Governors, freqPath); err != nil {
					return pathErrorInvalid(freq.Governors, "clusters/%s/cpufreq/governors", clusterName)
				}
				if err := pathMustOrStockCanExist(&freq.Max, GetPaths_CPUFreq_Max, freqPath); err != nil {
					return pathErrorInvalid(freq.Max, "clusters/%s/cpufreq/max", clusterName)
				}
				if err := pathMustOrStockCanExist(&freq.Min, GetPaths_CPUFreq_Min, freqPath); err != nil {
					return pathErrorInvalid(freq.Min, "clusters/%s/cpufreq/min", clusterName)
				}
				if err := pathMustOrStockCanExist(&freq.Speed, GetPaths_CPUFreq_Speed, freqPath); err != nil {
					return pathErrorInvalid(freq.Speed, "clusters/%s/cpufreq/speed", clusterName)
				}
			}
			if freq.Stats == nil {
				stats := &PathsCPUFreqStats{}
				statsPath, prefix := GetPaths_CPUFreq_Stats(freqPath)
				if statsPath != "" {
					stats.Path = statsPath
					statsPath = pathJoin(prefix, statsPath)
					stats.TimeInState, _ = GetPaths_CPUFreq_Stats_TimeInState(statsPath)
					stats.TotalTrans, _ = GetPaths_CPUFreq_Stats_TotalTrans(statsPath)
					freq.Stats = stats
				}
			} else {
				stats := freq.Stats
				statsPath, err := pathOrStockMustExist(&stats.Path, GetPaths_CPUFreq_Stats, freqPath)
				if err != nil {
					//Stats defined in manifest paths, require a valid path to be available
					return pathErrorDefinition("clusters/%s/cpufreq/stats relative to path %s", clusterName, freqPath)
				}
				if err := pathMustOrStockCanExist(&stats.TimeInState, GetPaths_CPUFreq_Stats_TimeInState, statsPath); err != nil {
					return pathErrorInvalid(stats.TimeInState, "clusters/%s/stats/time_in_state", clusterName)
				}
				if err := pathMustOrStockCanExist(&stats.TotalTrans, GetPaths_CPUFreq_Stats_TotalTrans, statsPath); err != nil {
					return pathErrorInvalid(stats.TotalTrans, "clusters/%s/stats/total_trans", clusterName)
				}
				freq.Stats = stats
			}
			cluster.CPUFreq = freq
			delete(p.Clusters, clusterName)
			p.Clusters[clusterName] = cluster
		}
	}

	if p.Cpusets == nil {
		cpusets := &PathsCpusets{Sets: make(map[string]PathsCpuset)}
		cpusetsPath, _ := GetPaths_Cpusets()
		if cpusetsPath != "" {
			cpusets.Path = cpusetsPath
			sets, err := ioutil.ReadDir(cpusetsPath)
			if err != nil {
				return pathErrorDefinition("cpusets/path")
			}
			for _, set := range sets {
				if set.IsDir() {
					cpusetPath := PathsCpuset{}
					cpusetPath.CPUs, _ = GetPaths_Cpusets_CPUs(cpusetsPath)
					cpusetPath.CPUExclusive, _ = GetPaths_Cpusets_CPUExclusive(cpusetsPath)
					cpusets.Sets[set.Name()] = cpusetPath
				}
			}
			p.Cpusets = cpusets
		}
	} else {
		cpusets := p.Cpusets
		cpusetsPath, err := pathOrStockMustExist(&cpusets.Path, GetPaths_Cpusets)
		if err != nil {
			//Cpusets defined in manifest paths, require a valid path to be available
			return pathErrorDefinition("cpusets")
		}
		for cpusetName, cpusetPath := range cpusets.Sets {
			setPath := pathJoin(cpusetsPath, cpusetName)
			if err := pathMustOrStockCanExist(&cpusetPath.CPUs, GetPaths_Cpusets_CPUs, setPath); err != nil {
				return pathErrorInvalid(cpusetPath.CPUs, "cpusets/" + cpusetName + "/cpus")
			}
			if err := pathMustOrStockCanExist(&cpusetPath.CPUExclusive, GetPaths_Cpusets_CPUExclusive, setPath); err != nil {
				return pathErrorInvalid(cpusetPath.CPUExclusive, "cpusets/" + cpusetName + "/cpu_exclusive")
			}
			cpusets.Sets[cpusetName] = cpusetPath
		}
		p.Cpusets = cpusets
	}

	if p.IPA == nil {
		ipa := &PathsIPA{}
		ipaPath, _ := GetPaths_IPA()
		if ipaPath != "" {
			ipa.Path = ipaPath
			ipa.Enabled, _ = GetPaths_IPA_Enabled(ipaPath)
			ipa.ControlTemp, _ = GetPaths_IPA_ControlTemp(ipaPath)
			p.IPA = ipa
		}
	} else {
		ipa := p.IPA
		ipaPath, err := pathOrStockMustExist(&ipa.Path, GetPaths_IPA)
		if err != nil {
			//IPA defined in manifest paths, require a valid path to be available
			return pathErrorDefinition("ipa")
		}
		if err := pathMustOrStockCanExist(&ipa.Enabled, GetPaths_IPA_Enabled, ipaPath); err != nil {
			return pathErrorInvalid(ipa.Enabled, "ipa/enabled")
		}
		if err := pathMustOrStockCanExist(&ipa.ControlTemp, GetPaths_IPA_ControlTemp, ipaPath); err != nil {
			return pathErrorInvalid(ipa.ControlTemp, "ipa/control_temp")
		}
	}

	if p.GPU == nil {
		gpu := &PathsGPU{}
		gpuPath, _ := GetPaths_GPU()
		if gpuPath != "" {
			gpu.Path = gpuPath
			dvfs := &PathsGPUDVFS{}
			if err := pathMustOrStockCanExist(&dvfs.Max, GetPaths_GPU_DVFS_Max, gpuPath); err == nil {
				if err := pathMustOrStockCanExist(&dvfs.Min, GetPaths_GPU_DVFS_Min, gpuPath); err == nil {
					gpu.DVFS = dvfs
				}
			} else if err := pathMustOrStockCanExist(&dvfs.Governor, GetPaths_GPU_DVFS_Governor, gpuPath); err == nil {
				gpu.DVFS = dvfs
			}
			highspeed := &PathsGPUHighspeed{}
			if err := pathMustOrStockCanExist(&highspeed.Clock, GetPaths_GPU_Highspeed_Clock, gpuPath); err == nil {
				if err := pathMustOrStockCanExist(&highspeed.Load, GetPaths_GPU_Highspeed_Load, gpuPath); err == nil {
					gpu.Highspeed = highspeed
				}
			}
			p.GPU = gpu
		}
	} else {
		gpu := p.GPU

		gpuPath, err := pathOrStockMustExist(&gpu.Path, GetPaths_GPU)
		if err != nil {
			//GPU defined in manifest paths, require a valid path to be available
			return pathErrorDefinition("gpu")
		}

		if gpu.DVFS != nil {
			dvfs := gpu.DVFS

			if err := pathMustOrStockCanExist(&dvfs.Governor, GetPaths_GPU_DVFS_Governor, gpuPath); err != nil {
				return pathErrorInvalid(dvfs.Governor, "gpu/dvfs/governor")
			}
			if err := pathMustOrStockCanExist(&dvfs.Max, GetPaths_GPU_DVFS_Max, gpuPath); err != nil {
				return pathErrorInvalid(dvfs.Max, "gpu/dvfs/max")
			}
			if err := pathMustOrStockCanExist(&dvfs.Min, GetPaths_GPU_DVFS_Min, gpuPath); err != nil {
				return pathErrorInvalid(dvfs.Min, "gpu/dvfs/min")
			}
		}

		if gpu.Highspeed != nil {
			hs := gpu.Highspeed

			if err := pathMustOrStockCanExist(&hs.Clock, GetPaths_GPU_Highspeed_Clock, gpuPath); err != nil {
				return pathErrorInvalid(hs.Clock, "gpu/highspeed/clock")
			}
			if err := pathMustOrStockCanExist(&hs.Load, GetPaths_GPU_Highspeed_Load, gpuPath); err != nil {
				return pathErrorInvalid(hs.Load, "gpu/highspeed/load")
			}
		}
	}

	if p.Kernel == nil {
		krnl := &PathsKernel{}
		pathStockCanExist(&krnl.DynamicHotplug, GetPaths_Kernel_DynamicHotplug)
		pathStockCanExist(&krnl.PowerEfficient, GetPaths_Kernel_Power_Efficient)
		hmp := &PathsKernelHMP{}
		hmpPath, _ := GetPaths_Kernel_HMP()
		if hmpPath != "" {
			hmp.Path = hmpPath
			hmp.Boost, _ = GetPaths_Kernel_HMP_Boost(hmpPath)
			hmp.Semiboost, _ = GetPaths_Kernel_HMP_Semiboost(hmpPath)
			hmp.ActiveDownMigration, _ = GetPaths_Kernel_HMP_ActiveDownMigration(hmpPath)
			hmp.AggressiveUpMigration, _ = GetPaths_Kernel_HMP_AggressiveUpMigration(hmpPath)
			threshold := &PathsKernelHMPThreshold{}
			if err := pathStockMustExist(&threshold.Down, GetPaths_Kernel_HMP_Threshold_Down, hmpPath); err == nil {
				if err := pathStockMustExist(&threshold.Up, GetPaths_Kernel_HMP_Threshold_Up, hmpPath); err == nil {
					hmp.Threshold = threshold
				}
			}
			sbThreshold := &PathsKernelHMPThreshold{}
			if err := pathStockMustExist(&sbThreshold.Down, GetPaths_Kernel_HMP_SbThreshold_Down, hmpPath); err == nil {
				if err := pathStockMustExist(&sbThreshold.Up, GetPaths_Kernel_HMP_SbThreshold_Up, hmpPath); err == nil {
					hmp.SbThreshold = sbThreshold
				}
			}
			krnl.HMP = hmp
		}
		p.Kernel = krnl
	} else {
		krnl := p.Kernel

		if err := pathMustOrStockCanExist(&krnl.DynamicHotplug, GetPaths_Kernel_DynamicHotplug); err != nil {
			return pathErrorInvalid(krnl.DynamicHotplug, "kernel/dynamic_hotplug")
		}
		if err := pathMustOrStockCanExist(&krnl.PowerEfficient, GetPaths_Kernel_Power_Efficient); err != nil {
			return pathErrorInvalid(krnl.PowerEfficient, "kernel/power_efficient")
		}

		if krnl.HMP != nil {
			hmp := krnl.HMP

			hmpPath, err := pathOrStockMustExist(&hmp.Path, GetPaths_Kernel_HMP)
			if err != nil {
				//HMP defined in manifest paths, require a valid path to be available
				return pathErrorDefinition("kernel/hmp")
			}
			if err := pathMustOrStockCanExist(&hmp.Boost, GetPaths_Kernel_HMP_Boost, hmpPath); err != nil {
				return pathErrorInvalid(hmp.Boost, "kernel/hmp/boost")
			}
			if err := pathMustOrStockCanExist(&hmp.Semiboost, GetPaths_Kernel_HMP_Semiboost, hmpPath); err != nil {
				return pathErrorInvalid(hmp.Semiboost, "kernel/hmp/semiboost")
			}
			if err := pathMustOrStockCanExist(&hmp.ActiveDownMigration, GetPaths_Kernel_HMP_ActiveDownMigration, hmpPath); err != nil {
				return pathErrorInvalid(hmp.ActiveDownMigration, "kernel/hmp/active_down_migration")
			}
			if err := pathMustOrStockCanExist(&hmp.AggressiveUpMigration, GetPaths_Kernel_HMP_AggressiveUpMigration, hmpPath); err != nil {
				return pathErrorInvalid(hmp.AggressiveUpMigration, "kernel/hmp/aggressive_up_migration")
			}

			if hmp.Threshold != nil {
				thld := hmp.Threshold

				if err := pathMustOrStockCanExist(&thld.Down, GetPaths_Kernel_HMP_Threshold_Down, hmpPath); err != nil {
					return pathErrorInvalid(thld.Down, "kernel/hmp/threshold/down")
				}
				if err := pathMustOrStockCanExist(&thld.Up, GetPaths_Kernel_HMP_Threshold_Up, hmpPath); err != nil {
					return pathErrorInvalid(thld.Up, "kernel/hmp/threshold/up")
				}
			}

			if hmp.SbThreshold != nil {
				thld := hmp.SbThreshold

				if err := pathMustOrStockCanExist(&thld.Down, GetPaths_Kernel_HMP_SbThreshold_Down, hmpPath); err != nil {
					return pathErrorInvalid(thld.Down, "kernel/hmp/sb_threshold/down")
				}
				if err := pathMustOrStockCanExist(&thld.Up, GetPaths_Kernel_HMP_SbThreshold_Up, hmpPath); err != nil {
					return pathErrorInvalid(thld.Up, "kernel/hmp/sb_threshold/up")
				}
			}
		}
	}

	if p.InputBooster == nil {
		ib := &PathsInputBooster{}
		ibPath, _ := GetPaths_InputBooster()
		if ibPath != "" {
			ib.Path = ibPath
			ib.Head, _ = GetPaths_InputBooster_Head(ibPath)
			ib.Tail, _ = GetPaths_InputBooster_Tail(ibPath)
			p.InputBooster = ib
		}
	} else {
		ib := p.InputBooster

		ibPath, err := pathOrStockMustExist(&ib.Path, GetPaths_InputBooster)
		if err != nil {
			//InputBooster defined in manifest paths, require a valid path to be available
			return pathErrorDefinition("input_booster")
		}
		if err := pathMustOrStockCanExist(&ib.Head, GetPaths_InputBooster_Head, ibPath); err != nil {
			return pathErrorInvalid(ib.Head, "input_booster/head")
		}
		if err := pathMustOrStockCanExist(&ib.Tail, GetPaths_InputBooster_Tail, ibPath); err != nil {
			return pathErrorInvalid(ib.Tail, "input_booster/tail")
		}
	}

	if p.SecSlow == nil {
		slow := &PathsSecSlow{}
		slowPath, _ := GetPaths_SecSlow()
		if slowPath != "" {
			slow.Path = slowPath
			slow.Enabled, _ = GetPaths_SecSlow_Enabled(slowPath)
			slow.Enforced, _ = GetPaths_SecSlow_Enforced(slowPath)
			slow.TimerRate, _ = GetPaths_SecSlow_TimerRate(slowPath)
			p.SecSlow = slow
		}
	} else {
		slow := p.SecSlow

		slowPath, err := pathOrStockMustExist(&slow.Path, GetPaths_SecSlow)
		if err != nil {
			//SecSlow defined in manifest paths, require a valid path to be available
			return pathErrorDefinition("sec_slow")
		}
		if err := pathMustOrStockCanExist(&slow.Enabled, GetPaths_SecSlow_Enabled, slowPath); err != nil {
			return pathErrorInvalid(slow.Enabled, "sec_slow/enabled")
		}
		if err := pathMustOrStockCanExist(&slow.Enforced, GetPaths_SecSlow_Enforced, slowPath); err != nil {
			return pathErrorInvalid(slow.Enforced, "sec_slow/enforced")
		}
		if err := pathMustOrStockCanExist(&slow.TimerRate, GetPaths_SecSlow_TimerRate, slowPath); err != nil {
			return pathErrorInvalid(slow.TimerRate, "sec_slow/timer_rate")
		}
	}

	if p.Inputs != nil && len(p.Inputs) > 0 {
		for inputName, _ := range p.Inputs {
			//We only want to recognize plugged in inputs, as they won't always be present
			//TODO: Listen to all input paths for any new directories so we can register configured paths
			input := p.Inputs[inputName]
			delete(p.Inputs, inputName)
			if _, err := pathOrStockMustExist(&input.Path, GetPaths_Input); err != nil {
				continue
			}
			if err := pathMustOrStockCanExist(&input.Enabled, GetPaths_Input_Enabled, input.Path); err != nil {
				continue
			}
			p.Inputs[inputName] = input
		}
	}

	return nil
}

func pathErrorDefinition(nameFormat string, formats ...any) error {
	name := fmt.Sprintf(nameFormat, formats...)
	return fmt.Errorf("please define path for %s, or remove it from manifest", name)
}

func pathErrorInvalid(path, nameFormat string, formats ...any) error {
	name := fmt.Sprintf(nameFormat, formats...)
	return fmt.Errorf("invalid %s path %s", name, path)
}

func pathJoin(parts ...string) string {
	path := ""
	for i := 0; i < len(parts); i++ {
		if i > 0 {
			path += "/"
		}
		path += parts[i]
	}
	return path
}

func pathOrStockMustExist(path *string, handler func(...string) (string, string), prefixes ...string) (string, error) {
	if *path == "" {
		tmp, prefix := handler(prefixes...)
		if tmp == "" {
			return "", fmt.Errorf("stock path not found")
		}
		*path = tmp
		if prefix != "" {
			return pathJoin(prefix, *path), nil
		}
	} else {
		if len(prefixes) > 0 {
			for i := 0; i < len(prefixes); i++ {
				tmp := prefixes[i] + "/" + *path
				if pathValid(tmp) {
					return tmp, nil
				}
			}
			return "", fmt.Errorf("defined path is invalid for prefixes")
		} else if !pathValid(*path) {
			return "", fmt.Errorf("defined path is invalid")
		}
	}
	return *path, nil
}

func pathMustOrStockCanExist(path *string, handler func(...string) (string, string), prefixes ...string) error {
	if *path == "" {
		*path, _ = handler(prefixes...)
	} else {
		if len(prefixes) > 0 {
			for i := 0; i < len(prefixes); i++ {
				if pathValid(prefixes[i] + "/" + *path) {
					return nil
				}
			}
			return fmt.Errorf("defined path is invalid for prefixes")
		} else if !pathValid(*path) {
			return fmt.Errorf("defined path is invalid")
		}
	}
	return nil
}

func pathStockCanExist(path *string, handler func(...string) (string, string), prefixes ...string) {
	_ = pathStockMustExist(path, handler, prefixes...)
}

func pathStockMustExist(path *string, handler func(...string) (string, string), prefixes ...string) error {
	*path, _ = handler(prefixes...)
	if *path == "" {
		return fmt.Errorf("stock path not found")
	}
	return nil
}

func pathValid(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

/* NOTES:
- When testing against prefixes, paths MUST NOT be prefixed as root paths! For example:
    paths[0]: /bar
    prefix[0]: /foo
    test: /foo//bar - likely invalid on your platform!
*/
func pathLoop(paths []string, prefix ...string) (string, string) {
	if paths == nil || len(paths) < 1 {
		return "", ""
	}
	if len(prefix) > 0 {
		for i := 0; i < len(prefix); i++ {
			if !pathValid(prefix[i]) {
				continue
			}
			for j := 0; j < len(paths); j++ {
				path := prefix[i] + "/" + paths[j]
				if pathValid(path) {
					return paths[j], prefix[i]
				}
			}
		}
		return "", ""
	}
	for i := 0; i < len(paths); i++ {
		if pathValid(paths[i]) {
			return paths[i], ""
		}
	}
	return "", ""
}
