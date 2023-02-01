package certdemo

import "github.com/zjkmxy/go-ndn/pkg/schema"

type MemTpm struct {
}

type KeyNode struct {
	certNode *schema.LeafNode
}

type SignedBy struct {
	keyNode *KeyNode
}
