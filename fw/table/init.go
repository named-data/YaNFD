/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import "github.com/cornelk/hashmap"

func init() {
	measurements = &hashmap.HashMap{}
}
