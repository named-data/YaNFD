/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"github.com/eric135/YaNFD/core"
)

// enableLocalhopManagement determines whether management will listen for command and dataset Interests on non-local faces.
var enableLocalhopManagement bool

// Configure configures the face system.
func Configure() {
	enableLocalhopManagement = core.GetConfigBoolDefault("mgmt.allow_localhop", false)
}
