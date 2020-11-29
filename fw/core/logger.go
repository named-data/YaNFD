/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package core

import (
	"log"
	"os"
)

// Logger is the central logger for YaNFD
var logger *log.Logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds)

// LogFatal logs a message at the FATAL level
func LogFatal(v ...interface{}) {
	logger.Fatalln([]interface{}{"FATAL:", v})
}

// LogError logs a message at the ERROR level
func LogError(v ...interface{}) {
	logger.Println([]interface{}{"ERROR:", v})
}

// LogWarn logs a message at the WARN level
func LogWarn(v ...interface{}) {
	logger.Println([]interface{}{"WARN:", v})
}

// LogInfo logs a message at the INFO level
func LogInfo(v ...interface{}) {
	logger.Println([]interface{}{"INFO:", v})
}

// LogDebug logs a message at the DEBUG level
func LogDebug(v ...interface{}) {
	logger.Println([]interface{}{"DEBUG:", v})
}

// LogTrace logs a message at the TRACE level
func LogTrace(v ...interface{}) {
	logger.Println([]interface{}{"TRACE:", v})
}
