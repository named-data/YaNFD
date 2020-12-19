package fw

import (
	"crypto/sha512"
	"encoding/binary"

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
	threadID int
	HasQuit  chan bool
}

// NewThread creates a new forwarding thread
func NewThread(id int) Thread {
	return Thread{id, make(chan bool)}
}

// Run forwarding thread
func (t *Thread) Run() {
	// TODO
}

// GetID returns the ID of the forwarding thread
func (t *Thread) GetID() int {
	return t.GetID()
}
