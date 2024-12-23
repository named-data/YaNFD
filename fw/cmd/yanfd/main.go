/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package main

import (
	"os"

	"github.com/pulsejet/ndnd/fw/executor"
)

func main() {
	executor.Main(os.Args)
}
