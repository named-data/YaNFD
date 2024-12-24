package object

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sync"

	enc "github.com/named-data/ndnd/std/encoding"
	bolt "go.etcd.io/bbolt"
)

var BoltBucket = []byte("data")
var ErrBoltNoBucket = errors.New("no bucket in bolt")

// Store implementation using bbolt
// Internally it uses a single bucket to store all data
// Internal storage format is not stable. Do not rely on it.
// Note: insertions to bolt are comically slow unless batched.
//
//	The key is the name of the object as a TLV encoded byte slice
//	The value is the 8-byte version (big endian), followed by data wire
type BoltStore struct {
	db   *bolt.DB
	tx   *bolt.Tx
	wmut sync.Mutex
}

func NewBoltStore(path string) (*BoltStore, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(BoltBucket); err != nil {
			panic(err)
		}
		return nil
	})

	return &BoltStore{db: db}, nil
}

func (s *BoltStore) Close() error {
	return s.db.Close()
}

func (s *BoltStore) Get(name enc.Name, prefix bool) (wire []byte, err error) {
	key := s.encodeName(name)
	err = s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BoltBucket)
		if bucket == nil {
			return ErrBoltNoBucket
		}

		if prefix {
			c := bucket.Cursor()
			iter := 1000
			maxVer := uint64(0)
			for k, v := c.Seek(key); k != nil && bytes.HasPrefix(k, key); k, v = c.Next() {
				if iter--; iter <= 0 {
					// checked too many keys ... give up
					// TODO: find a better way to enforce this never happens upstream
					break
				}

				if len(v) < 8 {
					continue
				}
				ver := binary.BigEndian.Uint64(v[:8])
				if ver > maxVer {
					wire = v[8:]
				}
			}
		} else {
			wire = bucket.Get(key)
			if wire != nil {
				wire = wire[8:]
			}
		}

		if wire != nil {
			wire = append([]byte(nil), wire...) // copy
		}
		return nil
	})

	return
}

func (s *BoltStore) Put(name enc.Name, version uint64, wire []byte) error {
	key := s.encodeName(name)

	buf := make([]byte, 8, 8+len(wire))
	binary.BigEndian.PutUint64(buf, version)
	buf = append(buf, wire...)

	// get lock after encoding data
	s.wmut.Lock()
	defer s.wmut.Unlock()

	// insert data into bolt
	update := func(tx *bolt.Tx) error {
		bucket := tx.Bucket(BoltBucket)
		if bucket == nil {
			return ErrBoltNoBucket
		}
		return bucket.Put(key, buf)
	}

	// use current transaction if available otherwise create a new one
	if s.tx != nil {
		return update(s.tx)
	} else {
		return s.db.Update(update)
	}
}

func (s *BoltStore) Remove(name enc.Name, prefix bool) error {
	key := s.encodeName(name)
	return s.db.Update(func(tx *bolt.Tx) (err error) {
		bucket := tx.Bucket(BoltBucket)
		if bucket == nil {
			return ErrBoltNoBucket
		}

		if prefix {
			c := bucket.Cursor()
			for k, _ := c.Seek(key); k != nil && bytes.HasPrefix(k, key); k, _ = c.Next() {
				if err = bucket.Delete(k); err != nil {
					return err
				}
			}
			return nil
		} else {
			return bucket.Delete(key)
		}
	})
}

func (s *BoltStore) Begin() error {
	// bolt has only one concurrent write transaction
	// so this will block if there is already a write transaction
	tx, err := s.db.Begin(true)
	if err != nil {
		return err
	}

	s.wmut.Lock()
	defer s.wmut.Unlock()

	if s.tx != nil {
		panic("bolt: write transaction already in progress")
	}
	s.tx = tx

	return nil
}

func (s *BoltStore) Commit() error {
	s.wmut.Lock()
	defer s.wmut.Unlock()

	err := s.tx.Commit()
	s.tx = nil
	return err
}

func (s *BoltStore) Rollback() error {
	s.wmut.Lock()
	defer s.wmut.Unlock()

	err := s.tx.Rollback()
	s.tx = nil
	return err
}

func (s *BoltStore) encodeName(name enc.Name) []byte {
	buf := make([]byte, name.EncodingLength())
	size := 0
	for _, comp := range name {
		size += comp.EncodeInto(buf[size:])
	}
	return buf[:size]
}
