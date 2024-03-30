package main

var Paths_Cluster = []string{"/sys/devices/system/cpu"}
func GetPaths_Cluster(prefix ...string) (string, string) {
	return pathLoop(Paths_Cluster, prefix...)
}
func CachePaths_Cluster(cache string) {
	for _, path := range Paths_Cluster {
		if cache == path {
			return
		}
	}
	Paths_Cluster = append(Paths_Cluster, cache)
}

var Paths_CPUFreq = []string{"cpu0/cpufreq"}
func GetPaths_CPUFreq(prefix ...string) (string, string) {
	return pathLoop(Paths_CPUFreq, prefix...)
}

var Paths_CPUFreq_Governor = []string{"scaling_governor"}
func GetPaths_CPUFreq_Governor(prefix ...string) (string, string) {
	return pathLoop(Paths_CPUFreq_Governor, prefix...)
}

var Paths_CPUFreq_Governors = []string{"scaling_available_governors"}
func GetPaths_CPUFreq_Governors(prefix ...string) (string, string) {
	return pathLoop(Paths_CPUFreq_Governors, prefix...)
}

var Paths_CPUFreq_Max = []string{"scaling_max_freq"}
func GetPaths_CPUFreq_Max(prefix ...string) (string, string) {
	return pathLoop(Paths_CPUFreq_Max, prefix...)
}

var Paths_CPUFreq_Min = []string{"scaling_min_freq"}
func GetPaths_CPUFreq_Min(prefix ...string) (string, string) {
	return pathLoop(Paths_CPUFreq_Min, prefix...)
}

var Paths_CPUFreq_Speed = []string{"scaling_setspeed"}
func GetPaths_CPUFreq_Speed(prefix ...string) (string, string) {
	return pathLoop(Paths_CPUFreq_Speed, prefix...)
}

var Paths_CPUFreq_Stats = []string{"stats"}
func GetPaths_CPUFreq_Stats(prefix ...string) (string, string) {
	return pathLoop(Paths_CPUFreq_Stats, prefix...)
}

var Paths_CPUFreq_Stats_TimeInState = []string{"time_in_state"}
func GetPaths_CPUFreq_Stats_TimeInState(prefix ...string) (string, string) {
	return pathLoop(Paths_CPUFreq_Stats_TimeInState, prefix...)
}

var Paths_CPUFreq_Stats_TotalTrans = []string{"total_trans"}
func GetPaths_CPUFreq_Stats_TotalTrans(prefix ...string) (string, string) {
	return pathLoop(Paths_CPUFreq_Stats_TotalTrans, prefix...)
}

var Paths_Cpusets = []string{"/dev/cpuset"}
func GetPaths_Cpusets(prefix ...string) (string, string) {
	return pathLoop(Paths_Cpusets, prefix...)
}

var Paths_Cpusets_CPUs = []string{"cpus"}
func GetPaths_Cpusets_CPUs(prefix ...string) (string, string) {
	return pathLoop(Paths_Cpusets_CPUs, prefix...)
}

var Paths_Cpusets_CPUExclusive = []string{"cpu_exclusive"}
func GetPaths_Cpusets_CPUExclusive(prefix ...string) (string, string) {
	return pathLoop(Paths_Cpusets_CPUExclusive, prefix...)
}

var Paths_IPA = []string{"/sys/power/ipa"}
func GetPaths_IPA(prefix ...string) (string, string) {
	return pathLoop(Paths_IPA, prefix...)
}

var Paths_IPA_Enabled = []string{"enabled"}
func GetPaths_IPA_Enabled(prefix ...string) (string, string) {
	return pathLoop(Paths_IPA_Enabled, prefix...)
}

var Paths_IPA_ControlTemp = []string{"control_temp"}
func GetPaths_IPA_ControlTemp(prefix ...string) (string, string) {
	return pathLoop(Paths_IPA_ControlTemp, prefix...)
}

var Paths_GPU = []string{"/sys/devices/14ac0000.mali"}
func GetPaths_GPU(prefix ...string) (string, string) {
	return pathLoop(Paths_GPU, prefix...)
}

var Paths_GPU_DVFS_Governor = []string{"dvfs_governor"}
func GetPaths_GPU_DVFS_Governor(prefix ...string) (string, string) {
	return pathLoop(Paths_GPU_DVFS_Governor, prefix...)
}

var Paths_GPU_DVFS_Max = []string{"dvfs_max_lock"}
func GetPaths_GPU_DVFS_Max(prefix ...string) (string, string) {
	return pathLoop(Paths_GPU_DVFS_Max, prefix...)
}

var Paths_GPU_DVFS_Min = []string{"dvfs_min_lock"}
func GetPaths_GPU_DVFS_Min(prefix ...string) (string, string) {
	return pathLoop(Paths_GPU_DVFS_Min, prefix...)
}

var Paths_GPU_Highspeed_Clock = []string{"highspeed_clock"}
func GetPaths_GPU_Highspeed_Clock(prefix ...string) (string, string) {
	return pathLoop(Paths_GPU_Highspeed_Clock, prefix...)
}

var Paths_GPU_Highspeed_Load = []string{"highspeed_load"}
func GetPaths_GPU_Highspeed_Load(prefix ...string) (string, string) {
	return pathLoop(Paths_GPU_Highspeed_Load, prefix...)
}

var Paths_Kernel_DynamicHotplug = []string{"/sys/power/enable_dm_hotplug"}
func GetPaths_Kernel_DynamicHotplug(prefix ...string) (string, string) {
	return pathLoop(Paths_Kernel_DynamicHotplug, prefix...)
}

var Paths_Kernel_Power_Efficient = []string{"/sys/module/workqueue/parameters/power_efficient"}
func GetPaths_Kernel_Power_Efficient(prefix ...string) (string, string) {
	return pathLoop(Paths_Kernel_Power_Efficient, prefix...)
}

var Paths_Kernel_HMP = []string{"/sys/kernel/hmp"}
func GetPaths_Kernel_HMP(prefix ...string) (string, string) {
	return pathLoop(Paths_Kernel_HMP, prefix...)
}

var Paths_Kernel_HMP_Boost = []string{"boost"}
func GetPaths_Kernel_HMP_Boost(prefix ...string) (string, string) {
	return pathLoop(Paths_Kernel_HMP_Boost, prefix...)
}

var Paths_Kernel_HMP_Semiboost = []string{"semiboost"}
func GetPaths_Kernel_HMP_Semiboost(prefix ...string) (string, string) {
	return pathLoop(Paths_Kernel_HMP_Semiboost, prefix...)
}

var Paths_Kernel_HMP_ActiveDownMigration = []string{"active_down_migration"}
func GetPaths_Kernel_HMP_ActiveDownMigration(prefix ...string) (string, string) {
	return pathLoop(Paths_Kernel_HMP_ActiveDownMigration, prefix...)
}

var Paths_Kernel_HMP_AggressiveUpMigration = []string{"aggressive_up_migration"}
func GetPaths_Kernel_HMP_AggressiveUpMigration(prefix ...string) (string, string) {
	return pathLoop(Paths_Kernel_HMP_AggressiveUpMigration, prefix...)
}

var Paths_Kernel_HMP_Threshold_Down = []string{"down_threshold"}
func GetPaths_Kernel_HMP_Threshold_Down(prefix ...string) (string, string) {
	return pathLoop(Paths_Kernel_HMP_Threshold_Down, prefix...)
}

var Paths_Kernel_HMP_Threshold_Up = []string{"up_threshold"}
func GetPaths_Kernel_HMP_Threshold_Up(prefix ...string) (string, string) {
	return pathLoop(Paths_Kernel_HMP_Threshold_Up, prefix...)
}

var Paths_Kernel_HMP_SbThreshold_Down = []string{"sb_down_threshold"}
func GetPaths_Kernel_HMP_SbThreshold_Down(prefix ...string) (string, string) {
	return pathLoop(Paths_Kernel_HMP_SbThreshold_Down, prefix...)
}

var Paths_Kernel_HMP_SbThreshold_Up = []string{"sb_up_threshold"}
func GetPaths_Kernel_HMP_SbThreshold_Up(prefix ...string) (string, string) {
	return pathLoop(Paths_Kernel_HMP_SbThreshold_Up, prefix...)
}

var Paths_InputBooster = []string{"/sys/class/input_booster"}
func GetPaths_InputBooster(prefix ...string) (string, string) {
	return pathLoop(Paths_InputBooster, prefix...)
}

var Paths_InputBooster_Head = []string{"head"}
func GetPaths_InputBooster_Head(prefix ...string) (string, string) {
	return pathLoop(Paths_InputBooster_Head, prefix...)
}

var Paths_InputBooster_Tail = []string{"tail"}
func GetPaths_InputBooster_Tail(prefix ...string) (string, string) {
	return pathLoop(Paths_InputBooster_Tail, prefix...)
}

var Paths_SecSlow = []string{"/sys/devices/virtual/sec/sec_slow"}
func GetPaths_SecSlow(prefix ...string) (string, string) {
	return pathLoop(Paths_SecSlow, prefix...)
}

var Paths_SecSlow_Enabled = []string{"slow_mode"}
func GetPaths_SecSlow_Enabled(prefix ...string) (string, string) {
	return pathLoop(Paths_SecSlow_Enabled, prefix...)
}

var Paths_SecSlow_Enforced = []string{"enforced_slow_mode"}
func GetPaths_SecSlow_Enforced(prefix ...string) (string, string) {
	return pathLoop(Paths_SecSlow_Enforced, prefix...)
}

var Paths_SecSlow_TimerRate = []string{"timer_rate"}
func GetPaths_SecSlow_TimerRate(prefix ...string) (string, string) {
	return pathLoop(Paths_SecSlow_TimerRate, prefix...)
}

var Paths_Input = []string{"/sys/class/input"}
func GetPaths_Input(prefix ...string) (string, string) {
	return pathLoop(Paths_Input, prefix...)
}

var Paths_Input_Enabled = []string{"enabled"}
func GetPaths_Input_Enabled(prefix ...string) (string, string) {
	return pathLoop(Paths_Input_Enabled, prefix...)
}
