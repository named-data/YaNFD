package main

import (
	"fmt"
	"os"

	dv "github.com/pulsejet/ndnd/dv/executor"
	fw "github.com/pulsejet/ndnd/fw/executor"
)

const banner = `
  _   _ ____  _   _     _
 | \ | |  _ \| \ | | __| |
 |  \| | | | |  \| |/ _  |
 | |\  | |_| | |\  | (_| |
 |_| \_|____/|_| \_|\____|
`

type CmdTree struct {
	cmd  string
	help string
	sub  []*CmdTree
	fun  func([]string)
}

func (c *CmdTree) Usage(args []string) {
	fmt.Fprintln(os.Stderr, banner[1:])
	fmt.Fprintf(os.Stderr, "%s (%s)\n\n", c.help, c.cmd)
	fmt.Fprintf(os.Stderr, "Usage: %s [command]\n", args[0])
	for _, sub := range c.sub {
		fmt.Fprintf(os.Stderr, "  %s\t\t%s\n", sub.cmd, sub.help)
	}
	fmt.Fprintln(os.Stderr)
	os.Exit(2)
}

func (c *CmdTree) Execute(args []string) {
	// eagerly execute command if found
	if c.fun != nil {
		c.fun(args)
		return
	}

	if len(args) <= 1 {
		c.Usage(args)
		return
	}

	// recursively search for subcommand
	for _, sub := range c.sub {
		if args[1] == sub.cmd {
			name := args[0] + " " + args[1]
			sargs := append([]string{name}, args[2:]...)
			sub.Execute(sargs)
			return
		}
	}

	// command not found
	c.Usage(args)
}

func main() {
	// create a command tree
	tree := CmdTree{
		cmd:  "ndnd",
		help: "Named Data Networking Daemon",
		sub: []*CmdTree{{
			cmd:  "fw",
			help: "NDN Forwarding Daemon",
			sub: []*CmdTree{{
				cmd:  "start",
				help: "Start the NDN Forwarding Daemon",
				fun:  fw.Main,
			}},
		}, {
			cmd:  "dv",
			help: "NDN Distance Vector Routing Daemon",
			sub: []*CmdTree{{
				cmd:  "start",
				help: "Start the NDN Distance Vector Routing Daemon",
				fun:  dv.Main,
			}},
		}},
	}

	// Parse the command line arguments
	args := os.Args
	args[0] = "ndnd"
	tree.Execute(args)
}
