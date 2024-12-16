package main

import (
	"os"

	"github.com/goccy/go-yaml"
	"github.com/pulsejet/go-ndn-dv/cmd"
	"github.com/zjkmxy/go-ndn/pkg/log"
)

func main() {
	var cfgFile string = "/etc/ndn/dv.yml"
	if len(os.Args) >= 2 {
		cfgFile = os.Args[1]
	}

	cfgBytes, err := os.ReadFile(cfgFile)
	if err != nil {
		panic(err)
	}

	yc := cmd.YamlConfig{}
	if err = yaml.Unmarshal(cfgBytes, &yc); err != nil {
		panic(err)
	}

	log.SetLevel(log.InfoLevel)

	if err = cmd.Run(yc); err != nil {
		panic(err)
	}
}
