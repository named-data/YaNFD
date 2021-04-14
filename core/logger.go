/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package core

import (
	"fmt"
	"os"

	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
)

var shouldPrintTraceLogs = false
var logLevel log.Level

// InitializeLogger initializes the logger.
func InitializeLogger() {
	log.SetHandler(text.New(os.Stdout))

	logLevelString := GetConfigStringDefault("core.log_level", "INFO")

	var err error
	logLevel, err = log.ParseLevel(logLevelString)
	if err == nil {
		log.SetLevel(logLevel)
	} else if logLevelString == "TRACE" {
		// Apex doesn't support the TRACE level, so we have to work around that by calling them DEBUG, but not printing them if not TRACE
		log.SetLevel(log.DebugLevel)
		shouldPrintTraceLogs = true
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

// LogFatal logs a message at the FATAL level.
func LogFatal(module interface{}, message string) {
	if logLevel <= log.FatalLevel {
		log.Fatal(fmt.Sprintf("[%v] ", module) + ": " + message)
	}
}

// LogError logs a message at the ERROR level.
func LogError(module interface{}, message string) {
	if logLevel <= log.ErrorLevel {
		log.Error(fmt.Sprintf("[%v] ", module) + ": " + message)
	}
}

// LogWarn logs a message at the WARN level.
func LogWarn(module interface{}, message string) {
	if logLevel <= log.WarnLevel {
		log.Warn(fmt.Sprintf("[%v] ", module) + ": " + message)
	}
}

// LogInfo logs a message at the INFO level.
func LogInfo(module interface{}, message string) {
	if logLevel <= log.InfoLevel {
		log.Info(fmt.Sprintf("[%v] ", module) + ": " + message)
	}
}

// LogDebug logs a message at the DEBUG level.
func LogDebug(module interface{}, message string) {
	if logLevel <= log.DebugLevel {
		log.Debug(fmt.Sprintf("[%v] ", module) + ": " + message)
	}
}

// LogTrace logs a message at the TRACE level (really just additional DEBUG messages).
func LogTrace(module interface{}, message string) {
	if shouldPrintTraceLogs {
		log.Debug(fmt.Sprintf("[%v] ", module) + ": " + message)
	}
}
