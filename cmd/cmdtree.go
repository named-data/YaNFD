package cmd

import (
	"fmt"
	"os"
	"strings"
)

const banner = `
  _   _ ____  _   _     _
 | \ | |  _ \| \ | | __| |
 |  \| | | | |  \| |/ _  |
 | |\  | |_| | |\  | (_| |
 |_| \_|____/|_| \_|\____|
`

type CmdTree struct {
	Name string
	Help string
	Sub  []*CmdTree
	Fun  func([]string)
}

func (c *CmdTree) Usage(args []string) {
	fmt.Fprintln(os.Stderr, banner[1:])
	fmt.Fprintf(os.Stderr, "%s (%s)\n\n", c.Help, c.Name)
	fmt.Fprintf(os.Stderr, "Usage: %s [command]\n", args[0])
	for _, sub := range c.Sub {
		spaces := strings.Repeat(" ", 16-len(sub.Name))
		fmt.Fprintf(os.Stderr, "  %s%s%s\n", sub.Name, spaces, sub.Help)
	}
	fmt.Fprintln(os.Stderr)
	os.Exit(2)
}

func (c *CmdTree) Execute(args []string) {
	// eagerly execute command if found
	if c.Fun != nil {
		c.Fun(args)
		return
	}

	if len(args) <= 1 {
		c.Usage(args)
		return
	}

	// recursively search for subcommand
	for _, sub := range c.Sub {
		if len(sub.Name) > 0 && args[1] == sub.Name {
			name := args[0] + " " + args[1]
			sargs := append([]string{name}, args[2:]...)
			sub.Execute(sargs)
			return
		}
	}

	// command not found
	c.Usage(args)
}
