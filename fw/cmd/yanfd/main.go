/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package main

import (
	"os"

	"github.com/named-data/ndnd/fw/executor"
)

func main() {
	args := os.Args
	args[0] = "yanfd"
	executor.Main(args)
}
