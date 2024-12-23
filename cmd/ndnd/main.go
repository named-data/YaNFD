package main

import (
	"os"

	"github.com/pulsejet/ndnd/cmd"
	dv "github.com/pulsejet/ndnd/dv/executor"
	fw "github.com/pulsejet/ndnd/fw/executor"
)

func main() {
	// create a command tree
	tree := cmd.CmdTree{
		Name: "ndnd",
		Help: "Named Data Networking Daemon",
		Sub: []*cmd.CmdTree{{
			Name: "fw",
			Help: "NDN Forwarding Daemon",
			Sub: []*cmd.CmdTree{{
				Name: "run",
				Help: "Start the NDN Forwarding Daemon",
				Fun:  fw.Main,
			}},
		}, {
			Name: "dv",
			Help: "NDN Distance Vector Routing Daemon",
			Sub: []*cmd.CmdTree{{
				Name: "run",
				Help: "Start the NDN Distance Vector Routing Daemon",
				Fun:  dv.Main,
			}},
		}},
	}

	// Parse the command line arguments
	args := os.Args
	args[0] = tree.Name
	tree.Execute(args)
}
