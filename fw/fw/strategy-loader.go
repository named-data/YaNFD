/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package fw

import (
	"reflect"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/table"
)

const strategyDir = "strategy"

//var strategyPlugins []*plugin.Plugin
var strategyTypes []reflect.Type

// LoadStrategies loads the strategy modules.
func LoadStrategies() {
	/*strategyPlugins = make([]*plugin.Plugin, 0)

	// TODO: Make path configurable
	filepath.Walk("strategies", func(path string, info os.FileInfo, err error) error {
		if len(path) < 3 || path[len(path)-3:] != ".so" {
			// Skip non-plugin files
			return nil
		}

		if err != nil {
			core.LogError("StrategyLoader", "Unable to load strategy "+path+": "+err.Error())
			return nil
		}
		strategyPlugin, err := plugin.Open(path)
		if err != nil {
			core.LogError("StrategyLoader", "Unable to load strategy "+path+": "+err.Error())
			return nil
		}

		// Get strategy name
		strategyName, err := strategyPlugin.Lookup("StrategyName")
		if err != nil {
			core.LogError("StrategyLoader", "Unable to load strategy "+path+": StrategyName missing")
			return nil
		}

		// Make sure strategy class exists
		strategy, err := strategyPlugin.Lookup(strategyName.(string))
		if err != nil {
			core.LogError("StrategyLoader", "Unable to load strategy "+path+": Type "+strategyName.(string)+" missing")
			return nil
		}

		// Make sure strategy class can be cast to Strategy
		_, ok := strategy.(Strategy)
		if !ok {
			core.LogError("StrategyLoader", "Unable to load strategy "+path+": "+strategyName.(string)+" does not satisfy the requirements of Strategy")
			return nil
		}

		core.LogDebug("StrategyLoader", "Loaded "+strategyName.(string))
		strategyPlugins = append(strategyPlugins, strategyPlugin)
		return nil
	})*/
}

// InstantiateStrategies instantiates all strategies for a forwarding thread.
func InstantiateStrategies() map[string]Strategy {
	strategies := make(map[string]Strategy, len(strategyTypes))

	for _, strategyType := range strategyTypes {
		strategy := reflect.New(strategyType.Elem()).Interface().(Strategy)
		strategy.Instantiate(table.FibStrategyTable)
		strategies[strategy.GetName().String()] = strategy
		core.LogDebug("StrategyLoader", "Instantiated strategy "+strategy.GetName().String())
	}

	/*for _, plugin := range strategyPlugins {
		// We've already guaranteed that these won't error out in LoadStrategies
		name, _ := plugin.Lookup("Name")
		rawStrategy, _ := plugin.Lookup(name.(string))
		strategy := rawStrategy.(Strategy)
		strategies[strategy.GetName().String()] = strategy
	}*/

	return strategies
}
