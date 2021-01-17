/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package fw

import (
	"crypto/sha512"
	"encoding/binary"
	"strconv"

	"github.com/eric135/YaNFD/core"
	"github.com/eric135/YaNFD/ndn"
)

// MaxFwThreads Maximum number of forwarding threads
const MaxFwThreads = 32

// Threads contains all forwarding threads
var Threads map[int]*Thread

// HashNameToFwThread hashes an NDN name to a forwarding thread.
func HashNameToFwThread(name *ndn.Name) int {
	sum := sha512.Sum512([]byte(name.String()))
	return int(binary.BigEndian.Uint64(sum[56:]) % uint64(len(Threads)))
}

// HashNameToAllPrefixFwThreads hahes an NDN name to all forwarding threads for all prefixes of the name.
func HashNameToAllPrefixFwThreads(name *ndn.Name) []int {
	threadMap := make(map[int]interface{})

	for i := name.Size(); i > 0; i++ {
		threadMap[HashNameToFwThread(name.Prefix(i))] = true
	}

	threadList := make([]int, 0, len(threadMap))
	for i := range threadMap {
		threadList = append(threadList, i)
	}
	return threadList
}

// Thread Represents a forwarding thread
type Thread struct {
	threadID         int
	pendingInterests chan *ndn.Interest
	pendingDatas     chan *ndn.Data
	HasQuit          chan bool
}

// NewThread creates a new forwarding thread
func NewThread(id int) Thread {
	return Thread{
		threadID:         id,
		pendingInterests: make(chan *ndn.Interest),
		pendingDatas:     make(chan *ndn.Data),
		HasQuit:          make(chan bool)}
}

func (t *Thread) String() string {
	return "FwThread-" + strconv.Itoa(t.threadID)
}

// Run forwarding thread
func (t *Thread) Run() {
	for !core.ShouldQuit {
		select {
		case data := <-t.pendingDatas:
			core.LogTrace(t, "Processing Data "+data.Name().String())

			// TODO
		case interest := <-t.pendingInterests:
			core.LogTrace(t, "Processing Interest "+interest.Name().String())

			// TODO
		}
	}

	core.LogInfo(t, "Stopping thread")
	t.HasQuit <- true
}

// QueueInterest queues an Interest for processing by this forwarding thread.
func (t *Thread) QueueInterest(interest *ndn.Interest) {
	t.pendingInterests <- interest
}

// QueueData queues a DAta packet for processing by this forwarding thread.
func (t *Thread) QueueData(data *ndn.Data) {
	t.pendingDatas <- data
}

// GetID returns the ID of the forwarding thread
func (t *Thread) GetID() int {
	return t.GetID()
}
