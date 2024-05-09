package main

import (
	"fmt"
	"time"
)

//boostpulse_duration takes microseconds as input, so do we
func (dev *Device) Boost(durMs int32) {
	//Snapshot the time so we don't prolong or delay boosts
	startTime := time.Now()

	profile := dev.GetProfileNow()
	if profile == nil { return }

	for clusterName := range profile.Clusters {
		clusterDurMs := durMs
		if durMs <= 0 {
			if governor := dev.GetCPUGovernor(clusterName); governor != "" {
				if data := dev.GetCPUGovernorData(clusterName, governor); data != nil {
					if boostpulse, exists := data["boostpulse_duration"]; exists {
						clusterDurMs = int32(boostpulse.(float64))
						if clusterDurMs <= 0 { continue }
					} else { continue }
				} else { continue }
			} else { continue }
		}
		clusterDurMs -= int32(time.Now().Sub(startTime).Milliseconds()) * 1000
		if clusterDurMs <= 0 { continue }
		if governorName := dev.GetCPUGovernor(clusterName); governorName != "" {
			if governorName == "interactive" {
				dev.BoostMutex.Lock()
				defer dev.BoostMutex.Unlock()
				//go Debug("Boosting %s for %dμs", clusterName, clusterDurMs)
				if err := dev.SetCPUGovernorData(clusterName, governorName, "boostpulse_duration", fmt.Sprintf("%d", clusterDurMs)); err != nil {
					Error("Failed to time boost on %s for %dμs: %v", clusterName, clusterDurMs, err)
					continue
				}
				if err := dev.SetCPUGovernorData(clusterName, governorName, "boostpulse", "1"); err != nil {
					Error("Failed to boost %s: %v", clusterName, err)
					continue
				}
			}
		} else { Error("Failed to boost %s: Could not identify governor", clusterName) }
	}
}

func (dev *Device) GovernCPU(clusterName string) {
	for {
		start := time.Now()
		if dev.GetCPUGovernor(clusterName) != "powerpulse" {
			break
		}
		data := dev.GetCPUGovernorData(clusterName, "powerpulse")
		if data == nil {
			continue //What do?
		}

		//Do things like stats tracking and scaling from the frequencies table based on CPU usage %
		/* asdf */

		delta := time.Now().Sub(start).Milliseconds()
		timing := data["min_sample_time"].(int64)
		if delta < timing {
			time.Sleep(time.Millisecond * time.Duration(timing - delta))
		}
	}
}

func (dev *Device) GetCPUGovernor(clusterName string) string {
	profile := dev.GetProfileNow()
	if profile == nil {
		return ""
	}

	cluster, exists := profile.Clusters[clusterName]
	if !exists {
		return ""
	}

	if cluster.CPUFreq == nil {
		return ""
	}
	return cluster.CPUFreq.Governor
}

func (dev *Device) SetCPUGovernor(clusterName, governorName string) error {
	if pathCluster, exists := dev.Paths.Clusters[clusterName]; exists {
		if pathFreq := pathCluster.CPUFreq; pathFreq != nil {
			governorPath := pathJoin(pathCluster.Path, pathCluster.CPU, pathFreq.Path, pathFreq.Governor)
			if !pathValid(governorPath) {
				return fmt.Errorf("failed to find valid path at %s when setting governor: %s/%s", governorPath, clusterName, governorName)
			}
			return dev.write(governorPath, governorName)
		}
		return fmt.Errorf("failed to find cpufreq for cluster %s when setting governor", clusterName)
	}
	return fmt.Errorf("failed to find cluster %s when setting governor", clusterName)
}

func (dev *Device) GetCPUGovernorData(clusterName, governorName string) map[string]interface{} {
	profile := dev.GetProfileNow()
	if profile == nil {
		return nil
	}

	cluster, exists := profile.Clusters[clusterName]
	if !exists {
		return nil
	}

	if cluster.CPUFreq == nil || cluster.CPUFreq.Governors == nil {
		return nil
	}

	data, exists := cluster.CPUFreq.Governors[governorName]
	if !exists {
		return nil
	}
	return data
}

func (dev *Device) SetCPUGovernorData(clusterName, governorName, controlName, data string) error {
	if pathCluster, exists := dev.Paths.Clusters[clusterName]; exists {
		if pathFreq := pathCluster.CPUFreq; pathFreq != nil {
			controlPath := pathJoin(pathCluster.Path, pathCluster.CPU, pathFreq.Path, governorName, controlName)
			if !pathValid(controlPath) {
				return fmt.Errorf("failed to find valid path at %s when setting data: %s/%s/%s", controlPath, clusterName, governorName, controlName)
			}
			return dev.write(controlPath, data)
		}
		return fmt.Errorf("failed to find cpufreq for cluster %s when setting governor data", clusterName)
	}
	return fmt.Errorf("failed to find cluster %s when setting governor data", clusterName)
}
