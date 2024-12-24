package executor

import (
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/named-data/ndnd/fw/core"
)

type Profiler struct {
	config  *YaNFDConfig
	cpuFile *os.File
	block   *pprof.Profile
}

func NewProfiler(config *YaNFDConfig) *Profiler {
	return &Profiler{config: config}
}

func (p *Profiler) Start() (err error) {
	if p.config.CpuProfile != "" {
		p.cpuFile, err = os.Create(p.config.CpuProfile)
		if err != nil {
			core.LogFatal("Main", "Unable to open output file for CPU profile: ", err)
		}

		core.LogInfo("Main", "Profiling CPU - outputting to ", p.config.CpuProfile)
		pprof.StartCPUProfile(p.cpuFile)
	}

	if p.config.BlockProfile != "" {
		core.LogInfo("Main", "Profiling blocking operations - outputting to ", p.config.BlockProfile)
		runtime.SetBlockProfileRate(1)
		p.block = pprof.Lookup("block")
	}

	return
}

func (p *Profiler) Stop() {
	if p.block != nil {
		blockProfileFile, err := os.Create(p.config.BlockProfile)
		if err != nil {
			core.LogFatal("Main", "Unable to open output file for block profile: ", err)
		}
		if err := p.block.WriteTo(blockProfileFile, 0); err != nil {
			core.LogFatal("Main", "Unable to write block profile: ", err)
		}
		blockProfileFile.Close()
	}

	if p.config.MemProfile != "" {
		memProfileFile, err := os.Create(p.config.MemProfile)
		if err != nil {
			core.LogFatal("Main", "Unable to open output file for memory profile: ", err)
		}
		defer memProfileFile.Close()

		core.LogInfo("Main", "Profiling memory - outputting to ", p.config.MemProfile)
		runtime.GC()
		if err := pprof.WriteHeapProfile(memProfileFile); err != nil {
			core.LogFatal("Main", "Unable to write memory profile: ", err)
		}
	}

	if p.cpuFile != nil {
		pprof.StopCPUProfile()
		p.cpuFile.Close()
	}
}
