/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package core

import (
	"fmt"
	"log"
	"os"
)

// Logger is the central logger for YaNFD
var logger *log.Logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds)

// LogFatal logs a message at the FATAL level
func LogFatal(module interface{}, v ...interface{}) {
	logger.Fatalln(append([]interface{}{fmt.Sprintf("FATAL: [%v]", module)}, v...)...)
}

// LogError logs a message at the ERROR level
func LogError(module interface{}, v ...interface{}) {
	logger.Println(append([]interface{}{fmt.Sprintf("ERROR: [%v]", module)}, v...)...)
}

// LogWarn logs a message at the WARN level
func LogWarn(module interface{}, v ...interface{}) {
	logger.Println(append([]interface{}{fmt.Sprintf("WARN: [%v]", module)}, v...)...)
}

// LogInfo logs a message at the INFO level
func LogInfo(module interface{}, v ...interface{}) {
	logger.Println(append([]interface{}{fmt.Sprintf("INFO: [%v]", module)}, v...)...)
}

// LogDebug logs a message at the DEBUG level
func LogDebug(module interface{}, v ...interface{}) {
	logger.Println(append([]interface{}{fmt.Sprintf("DEBUG: [%v]", module)}, v...)...)
}

// LogTrace logs a message at the TRACE level
func LogTrace(module interface{}, v ...interface{}) {
	logger.Println(append([]interface{}{fmt.Sprintf("TRACE: [%v]", module)}, v...)...)
}
