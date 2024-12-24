package ndn

import enc "github.com/named-data/ndnd/std/encoding"

type Store interface {
	// returns a Data wire matching the given name
	// prefix = return the newest Data wire with the given prefix
	Get(name enc.Name, prefix bool) ([]byte, error)

	// inserts a Data wire into the store
	Put(name enc.Name, version uint64, wire []byte) error

	// removes a Data wire from the store
	// if prefix is set, all names with the given prefix are removed
	Remove(name enc.Name, prefix bool) error

	// begin a write transaction (for put only)
	// we support these primarily for performance rather than correctness
	// do not rely on atomicity of transactions as far as possible
	Begin() error
	// commit a write transaction
	Commit() error
	// rollback a write transaction
	Rollback() error
}
