package dv

import (
	"time"

	basic_engine "github.com/zjkmxy/go-ndn/pkg/engine/basic"
	"github.com/zjkmxy/go-ndn/pkg/log"
	mgmt "github.com/zjkmxy/go-ndn/pkg/ndn/mgmt_2022"
)

type mgmt_cmd struct {
	module  string
	cmd     string
	args    *mgmt.ControlArgs
	retries uint
}

type mgmt_thread struct {
	// engine
	engine *basic_engine.Engine
	// channel for management commands
	channel chan mgmt_cmd
	// stop the management thread
	stop chan bool
}

func newMgmtThread(engine *basic_engine.Engine) *mgmt_thread {
	return &mgmt_thread{
		engine:  engine,
		channel: make(chan mgmt_cmd, 4096), // TODO: deadlocks if full
		stop:    make(chan bool),
	}
}

func (m *mgmt_thread) Start() {
	for {
		select {
		case cmd := <-m.channel:
			for i := uint(1); i < cmd.retries+2; i++ {
				err := m.engine.ExecMgmtCmd(cmd.module, cmd.cmd, cmd.args)
				if err != nil {
					log.Errorf("NFD Management command failed: %+v", err)
					time.Sleep(100 * time.Millisecond)
				} else {
					time.Sleep(10 * time.Millisecond)
				}
			}
		case <-m.stop:
			return
		}
	}
}

func (m *mgmt_thread) Stop() {
	m.stop <- true
	close(m.channel)
	close(m.stop)
}

func (m *mgmt_thread) Exec(mgmt_cmd mgmt_cmd) {
	m.channel <- mgmt_cmd
}
