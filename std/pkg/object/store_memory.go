package object

import enc "github.com/zjkmxy/go-ndn/pkg/encoding"

type MemoryStore struct {
	// root of the store
	root *memoryStoreNode
}

type memoryStoreNode struct {
	// name component
	comp enc.Component
	// children
	children map[uint64]*memoryStoreNode
	// data wire
	wire []byte
	// version of this wire
	version uint64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		root: &memoryStoreNode{},
	}
}

func (s *MemoryStore) Get(name enc.Name, prefix bool) ([]byte, error) {
	if node := s.root.find(name); node != nil {
		if node.wire == nil && prefix {
			node = node.findNewest()
		}
		return node.wire, nil
	}
	return nil, nil
}

func (s *MemoryStore) Put(name enc.Name, version uint64, wire []byte) error {
	s.root.insert(name, version, wire)
	return nil
}

func (s *MemoryStore) Remove(name enc.Name, prefix bool) error {
	s.root.remove(name, prefix)
	return nil
}

func (n *memoryStoreNode) find(name enc.Name) *memoryStoreNode {
	if len(name) == 0 {
		return n
	}

	if n.children == nil {
		return nil
	}

	cHash := name[0].Hash()
	if child := n.children[cHash]; child != nil {
		return child.find(name[1:])
	} else {
		return nil
	}
}

func (n *memoryStoreNode) findNewest() *memoryStoreNode {
	known := n
	for _, child := range n.children {
		cl := child.findNewest()
		if cl.version > known.version {
			known = cl
		}
	}
	return known
}

func (n *memoryStoreNode) insert(name enc.Name, version uint64, wire []byte) {
	if len(name) == 0 {
		n.wire = wire
		n.version = version
		return
	}

	if n.children == nil {
		n.children = make(map[uint64]*memoryStoreNode)
	}

	cHash := name[0].Hash()
	if child := n.children[cHash]; child != nil {
		child.insert(name[1:], version, wire)
	} else {
		child = &memoryStoreNode{comp: name[0]}
		child.insert(name[1:], version, wire)
		n.children[cHash] = child
	}
}

func (n *memoryStoreNode) remove(name enc.Name, prefix bool) bool {
	// return value is if the parent should prune this child
	if len(name) == 0 {
		n.wire = nil
		n.version = 0
		if prefix {
			n.children = nil // prune subtree
		}
		return n.children == nil
	}

	if n.children == nil {
		return false
	}

	cHash := name[0].Hash()
	if child := n.children[cHash]; child != nil {
		prune := child.remove(name[1:], prefix)
		if prune {
			delete(n.children, cHash)
		}
	}

	return n.wire == nil && len(n.children) == 0
}
