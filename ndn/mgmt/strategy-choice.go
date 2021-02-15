/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"github.com/eric135/YaNFD/ndn"
	"github.com/eric135/YaNFD/ndn/tlv"
)

// StrategyChoice contains status information about a Strategy Choice table entry.
type StrategyChoice struct {
	Name     *ndn.Name
	Strategy *ndn.Name
}

// MakeStrategyChoice creates a StrategyChoice entry.
func MakeStrategyChoice(name *ndn.Name, strategy *ndn.Name) *StrategyChoice {
	s := new(StrategyChoice)
	s.Name = name
	s.Strategy = strategy
	return s
}

// StrategyChoiceList is a list of strategy choices.
type StrategyChoiceList []*StrategyChoice

// MakeStrategyChoiceList creates an empty StrategyChoiceList.
func MakeStrategyChoiceList() StrategyChoiceList {
	s := make(StrategyChoiceList, 0)
	return s
}

// Encode encodes a StrategyChoiceList as a series of blocks.
func (s *StrategyChoiceList) Encode() ([]*tlv.Block, error) {
	encoded := make([]*tlv.Block, 0)
	for _, strategyChoice := range *s {
		wire := tlv.NewEmptyBlock(tlv.StrategyChoice)
		wire.Append(strategyChoice.Name.Encode())
		strategyName := strategyChoice.Strategy.Encode()
		strategy := tlv.NewEmptyBlock(tlv.Strategy)
		strategy.Append(strategyName)
		wire.Append(strategy)
		wire.Encode()
		encoded = append(encoded, wire)
	}
	return encoded, nil
}
