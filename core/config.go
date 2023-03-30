/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package core

import (
	"math"
	"path/filepath"

	"github.com/pelletier/go-toml"
)

var (
	configFileDir string
	config        *toml.Tree
)

// LoadConfig loads the YaNFD configuration from the specified configuration file.
func LoadConfig(file string) {
	var err error
	config, err = toml.LoadFile(file)
	if err != nil {
		LogFatal("Config", "Unable to load configuration file: ", err)
	}
	configFileDir = filepath.Dir(file)
}

// GetConfigBoolDefault returns the boolean configuration value at the specified key or the specified default value
// if it does not exist.
func GetConfigBoolDefault(key string, def bool) bool {
	valRaw := config.Get(key)
	if valRaw == nil {
		return def
	}
	val, ok := valRaw.(bool)
	if ok {
		return val
	}
	return def
}

// GetConfigIntDefault returns the integer configuration value at the specified key or the specified default value
// if it does not exist.
func GetConfigIntDefault(key string, def int) int {
	valRaw := config.Get(key)
	if valRaw == nil {
		return def
	}
	val, ok := valRaw.(int64)
	if ok && val >= math.MinInt32 && val <= math.MaxInt32 {
		return int(val)
	}
	return def
}

// GetConfigStringDefault returns the string configuration value at the specified key or the specified default value
// if it does not exist.
func GetConfigStringDefault(key string, def string) string {
	valRaw := config.Get(key)
	if valRaw == nil {
		return def
	}
	val, ok := valRaw.(string)
	if ok {
		return val
	}
	return def
}

// GetConfigUint16Default returns the integer configuration value at the specified key or the specified default value
// if it does not exist.
func GetConfigUint16Default(key string, def uint16) uint16 {
	valRaw := config.Get(key)
	if valRaw == nil {
		return def
	}
	val, ok := valRaw.(int64)
	if ok && val > 0 && val <= math.MaxUint16 {
		return uint16(val)
	}
	return def
}

// GetConfigArrayString returns the configuration array value at the specified key or nil if it does not exist.
func GetConfigArrayString(key string) []string {
	array := config.GetArray(key)
	if array == nil {
		return nil
	}
	if val, ok := array.([]string); ok {
		return val
	}
	return nil
}

// ResolveConfigFileRelPath resolves a possibly relative path based on config file path.
func ResolveConfigFileRelPath(target string) (abs string) {
	if filepath.IsAbs(target) {
		return target
	}
	return filepath.Join(configFileDir, target)
}
