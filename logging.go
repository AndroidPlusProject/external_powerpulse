package main

import (
	"fmt"
)

const LOG_TAG = "PowerPulse"

type LogPriority int32
const (
	LogUnknown = iota
	LogDefault
	LogVerbose
	LogDebug
	LogInfo
	LogWarn
	LogError
	LogFatal
	LogSilent
)

func parseMsg(prio LogPriority, format string, replacements ...any) {
	if replacements == nil || len(replacements) < 1 {
		replacements = []any{format}
		format = "%v"
	}
	msg := fmt.Sprintf(format, replacements...)
	for msg[len(msg)-1] == '\n' {
		msg = string(msg[len(msg)-1])
		if msg == "" {
			break
		}
	}
	if msg != "" {
		logMsg(prio, msg)
	}
}

func Info(format string, replacements ...any) {
	parseMsg(LogInfo, format, replacements...)
}
func Warn(format string, replacements ...any) {
	parseMsg(LogWarn, format, replacements...)
}
func Error(format string, replacements ...any) {
	parseMsg(LogError, format, replacements...)
}
func Fatal(format string, replacements ...any) {
	parseMsg(LogFatal, format, replacements...)
}
func Verbose(format string, replacements ...any) {
	parseMsg(LogVerbose, format, replacements...)
}
func Debug(format string, replacements ...any) {
	if debug {
		parseMsg(LogDebug, format, replacements...)
	}
}
