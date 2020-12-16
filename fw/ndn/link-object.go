/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn

// LinkObject is a specialized Data packet containing delegation information.
type LinkObject struct {
	Data
}

// DeepCopy returns a deep copy of the LinkObject.
func (l *LinkObject) DeepCopy() *LinkObject {
	// TODO
	return nil
}
