/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package fw

import (
	"reflect"

	"github.com/named-data/YaNFD/core"
)

//const strategyDir = "strategy"

// var strategyPlugins []*plugin.Plugin
var strategyTypes []reflect.Type

// StrategyVersions contains a list of strategies mapping to a list of their versions
var StrategyVersions = make(map[string][]uint64)

// InstantiateStrategies instantiates all strategies for a forwarding thread.
func InstantiateStrategies(fwThread *Thread) map[uint64]Strategy {
	strategies := make(map[uint64]Strategy, len(strategyTypes))

	for _, strategyType := range strategyTypes {
		strategy := reflect.New(strategyType.Elem()).Interface().(Strategy)
		strategy.Instantiate(fwThread)
		strategies[strategy.GetName().Hash()] = strategy
		core.LogDebug("StrategyLoader", "Instantiated Strategy=", strategy.GetName(), " for Thread=", fwThread.GetID())
	}

	return strategies
}
