/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"github.com/cornelk/hashmap"
)

// Measurements contains the global measurements table,
var measurements *hashmap.HashMap

// GetMeasurement returns the measurement table value at the specified key or nil if it does not exist.
func GetMeasurement(key string) interface{} {
	value, isOk := measurements.GetStringKey(key)
	if !isOk {
		return nil
	}
	return value
}

// SetMeasurement atomically sets the value of the specified measurement table key only if it is equal to the expected value, returning whether the operation was successful.
func SetMeasurement(key string, expected interface{}, value interface{}) bool {
	return measurements.Cas(key, expected, value)
}

// AddToMeasurementInt adds the specified value to the given measurement key, setting as value if unitialized.
func AddToMeasurementInt(key string, value int) {
	wasSet := false
	for !wasSet {
		expected := GetMeasurement(key)
		if expected != nil {
			wasSet = SetMeasurement(key, expected, expected.(int)+value)
		} else {
			_, wasSet = measurements.GetOrInsert(key, value)
			// We need to flip this because it returns false if set
			wasSet = !wasSet
		}
	}
}

// AddSampleToEWMA adds a sample to an exponentially weighted moving average
func AddSampleToEWMA(key string, measurement float64, alpha float64) {
	wasSet := false
	for !wasSet {
		expected := GetMeasurement(key)
		if expected != nil {
			newValue := measurement + alpha*(measurement-expected.(float64))
			wasSet = SetMeasurement(key, expected, newValue)
		} else {
			_, wasSet = measurements.GetOrInsert(key, measurement)
			// We need to flip this because it returns false if set
			wasSet = !wasSet
		}
	}
}
