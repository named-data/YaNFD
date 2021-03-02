/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"bytes"
	"container/list"
	"time"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/ndn"
)

// DeadNonceList represents the Dead Nonce List for a forwarding thread.
type DeadNonceList struct {
	root            *DeadNonceListNode
	ExpirationTimer chan interface{}
	expiringEntries list.List
}

// DeadNonceListNode represents a node in a Dead Nonce List tree.
type DeadNonceListNode struct {
	component ndn.NameComponent
	depth     int

	parent   *DeadNonceListNode
	children []*DeadNonceListNode

	nonces [][]byte
}

type deadNonceListExpirationEntry struct {
	node           *DeadNonceListNode
	nonce          []byte
	expirationTime time.Time
}

// NewDeadNonceList creates a new Dead Nonce List for a forwarding thread.
func NewDeadNonceList() *DeadNonceList {
	d := new(DeadNonceList)
	d.root = new(DeadNonceListNode)
	d.root.component = nil // Root component will be nil since it represents zero components
	d.root.children = make([]*DeadNonceListNode, 0)
	d.root.nonces = make([][]byte, 0)
	d.ExpirationTimer = make(chan interface{}, core.FwQueueSize)
	return d
}

func (d *DeadNonceListNode) findExactMatchEntry(name *ndn.Name) *DeadNonceListNode {
	if name.Size() > d.depth {
		for _, child := range d.children {
			if name.At(child.depth - 1).Equals(child.component) {
				return child.findExactMatchEntry(name)
			}
		}
	} else if name.Size() == d.depth {
		return d
	}
	return nil
}

func (d *DeadNonceListNode) findLongestPrefixEntry(name *ndn.Name) *DeadNonceListNode {
	if name.Size() > d.depth {
		for _, child := range d.children {
			if name.At(child.depth - 1).Equals(child.component) {
				return child.findLongestPrefixEntry(name)
			}
		}
	}
	return d
}

func (d *DeadNonceListNode) fillTreeToPrefix(name *ndn.Name) *DeadNonceListNode {
	curNode := d.findLongestPrefixEntry(name)
	for depth := curNode.depth + 1; depth <= name.Size(); depth++ {
		newNode := new(DeadNonceListNode)
		newNode.component = name.At(depth - 1).DeepCopy()
		newNode.depth = depth
		newNode.parent = curNode
		curNode.children = append(curNode.children, newNode)
		curNode = newNode
	}
	return curNode
}

func (d *DeadNonceListNode) pruneIfEmpty() {
	for curNode := d; curNode.parent != nil && len(curNode.children) == 0 && len(curNode.nonces) == 0; curNode = curNode.parent {
		// Remove from parent's children
		for i, child := range curNode.parent.children {
			if child == d {
				if i < len(curNode.parent.children)-1 {
					copy(curNode.parent.children[i:], curNode.parent.children[i+1:])
				}
				curNode.parent.children = curNode.parent.children[:len(curNode.parent.children)-1]
				break
			}
		}
	}
}

// Find returns whether the specified name and nonce combination are present in the Dead Nonce List.
func (d *DeadNonceList) Find(name *ndn.Name, nonce []byte) bool {
	node := d.root.findExactMatchEntry(name)
	if node == nil {
		return false
	}

	for _, curNonce := range node.nonces {
		if bytes.Equal(curNonce, nonce) {
			return true
		}
	}
	return false
}

// Insert inserts an entry in the Dead Nonce List with the specified name and nonce. Returns node and whether nonce already present.
func (d *DeadNonceList) Insert(name *ndn.Name, nonce []byte) (*DeadNonceListNode, bool) {
	node := d.root.fillTreeToPrefix(name)

	for _, curEntry := range node.nonces {
		if bytes.Equal(curEntry, nonce) {
			return node, true
		}
	}

	node.nonces = append(node.nonces, nonce)
	d.expiringEntries.PushBack(&deadNonceListExpirationEntry{node: node, nonce: nonce, expirationTime: time.Now().Add(core.DeadNonceListLifetime)})
	go func() {
		time.Sleep(core.DeadNonceListLifetime)
		d.ExpirationTimer <- []interface{}{}
	}()
	return node, false
}

// RemoveExpiredEntry removes the front entry from Dead Nonce List.
func (d *DeadNonceList) RemoveExpiredEntry() {
	if d.expiringEntries.Len() > 0 {
		entry := d.expiringEntries.Front().Value.(*deadNonceListExpirationEntry)
		for i, nonce := range entry.node.nonces {
			if bytes.Equal(nonce, entry.nonce) {
				if i < len(entry.node.nonces)-1 {
					copy(entry.node.nonces[i:], entry.node.nonces[i+1:])
				}
				entry.node.nonces = entry.node.nonces[:len(entry.node.nonces)-1]
				if len(entry.node.nonces) == 0 {
					entry.node.pruneIfEmpty()
				}
				break
			}
		}
		d.expiringEntries.Remove(d.expiringEntries.Front())
	}
}
