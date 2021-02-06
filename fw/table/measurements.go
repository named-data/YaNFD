/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"github.com/cornelk/hashmap"
)

type measurements struct {
	table hashmap.HashMap
}

// Measurements contains the global measurements table,
var Measurements *measurements

func init() {
	Measurements = new(measurements)
}

// Get returns the measurement table value at the specified key or nil if it does not exist.
func (m *measurements) Get(key string) interface{} {
	value, isOk := m.table.GetStringKey(key)
	if !isOk {
		return nil
	}
	return value
}

// Set atomically sets the value of the specified measurement table key only if it is equal to the expected value, returning whether the operation was successful.
func (m *measurements) Set(key string, expected interface{}, value interface{}) bool {
	return m.table.Cas(key, expected, value)
}

// AddToInt adds the specified value to the given measurement key, setting as value if unitialized.
func (m *measurements) AddToInt(key string, value int) {
	wasSet := false
	for !wasSet {
		expected := m.Get(key)
		if expected != nil {
			wasSet = m.Set(key, expected, expected.(int)+value)
		} else {
			_, wasSet = m.table.GetOrInsert(key, value)
			// We need to flip this because it returns false if set
			wasSet = !wasSet
		}
	}
}

// AddSampleToEWMA adds a sample to an exponentially weighted moving average
func (m *measurements) AddSampleToEWMA(key string, measurement float64, alpha float64) {
	wasSet := false
	for !wasSet {
		expected := m.Get(key)
		if expected != nil {
			newValue := measurement + alpha*(measurement-expected.(float64))
			wasSet = m.Set(key, expected, newValue)
		} else {
			_, wasSet = m.table.GetOrInsert(key, measurement)
			// We need to flip this because it returns false if set
			wasSet = !wasSet
		}
	}
}
