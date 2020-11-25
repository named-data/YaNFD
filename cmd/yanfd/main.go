package main

import (
	"flag"
	"fmt"

	"github.com/eric135/YaNFD/core"
)

func main() {
	var shouldPrintVersion bool
	flag.BoolVar(&shouldPrintVersion, "version", false, "Print version and exit")
	flag.BoolVar(&shouldPrintVersion, "V", false, "Print version and exit (short)")
	flag.Parse()

	if shouldPrintVersion {
		fmt.Println("YaNFD: Yet another NDN Forwarding Daemon")
		fmt.Println("Version", core.Version)
		fmt.Println("Copyright (C) 2020 Eric Newberry and Xinyu Ma")
		fmt.Println("Released under the terms of the MIT License")
		return
	}
}
