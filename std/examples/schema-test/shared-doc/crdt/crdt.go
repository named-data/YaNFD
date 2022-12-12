package crdt

import "github.com/zjkmxy/go-ndn/pkg/utils"

type Doc[V any] struct {
	Producer uint64
	Clock    uint64
	Start    *Item[V]
}

type Item[V any] struct {
	ID          IDType
	Parent      *Doc[V]
	Left        *Item[V]
	Right       *Item[V]
	Origin      *IDType
	RightOrigin *IDType
	Content     V
	Deleted     bool
}

// Equal returns if two ids are equal
func (l *IDType) Equal(r *IDType) bool {
	if l == nil {
		return r == nil
	} else if r != nil {
		return *l == *r
	} else {
		return false
	}
}

// InsertInto inserts new item i (with Origin==Left and RightOrigin==Right) into d
func (i *Item[V]) InsertInto(d *Doc[V]) {
	i.Parent = d
	if d.Start == nil {
		d.Start = i
		return
	}
	// Check if there is a record in d that the producer of i did not know
	// If i is the leftmost item, check conflict if i.Right has an left element
	// If i is not the leftmost item, check if there is an item between left and right
	if (i.Left == nil && (i.Right == nil || i.Right.Left != nil)) || (i.Left != nil && i.Left.Right != i.Right) {
		// o iterates over all conflicting items
		o := d.Start
		if i.Left != nil {
			o = i.Left.Right // != i.Right
		}
		// (from Yjs) Let c in conflictingItems, b in itemsBeforeO
		// ***{origin}bbbb{this}{c,b}{c,b}{o}***
		// Note that conflictingItems is a subset of itemsBeforeO
		conflictingItems := make(map[IDType]struct{}, 0)
		itemsBeforeO := make(map[IDType]struct{}, 0)
		left := i.Left
		for ; o != nil && o != i.Right; o = o.Right {
			conflictingItems[o.ID] = struct{}{}
			itemsBeforeO[o.ID] = struct{}{}
			if i.Origin.Equal(o.Origin) {
				// Case 1: Two elements are attached to the same left
				if o.ID.Producer < i.ID.Producer {
					// The one with a smaller producer ID goes first.
					left = o
					conflictingItems = make(map[IDType]struct{}, 0)
				} else if i.RightOrigin.Equal(o.RightOrigin) {
					// i and o has the same left and right, and i's producer id is less, so i is before o
					// By non-crossing property, there shouldn't be any item before i.
					// This is only an optimization. Can be removed together with RightOrigin.
					break
				}
			} else if _, ok := itemsBeforeO[*o.Origin]; o.Origin != nil && ok {
				// Case 2: i.origin << o.origin
				if _, ok := conflictingItems[*o.Origin]; !ok {
					// If o.origin << i, by no-crossing policy, we must put i after o.
					left = o
					conflictingItems = make(map[IDType]struct{}, 0)
				}
			} else {
				// Case 3: o.origin << i.origin. By no-crossing policy, we must put i before o.
				break
			}
		}
		i.Left = left
	}
	// Join i into the linked list
	if i.Left != nil {
		right := i.Left.Right
		i.Right = right
		i.Left.Right = i
	} else {
		i.Right = d.Start
		d.Start = i
	}
	if i.Right != nil {
		i.Right.Left = i
	}
}

// Next returns next non-deleted item after i
func (i *Item[V]) Next() *Item[V] {
	n := i.Right
	for n != nil && n.Deleted {
		n = n.Right
	}
	return n
}

// Prev returns previous non-deleted item before i
func (i *Item[V]) Prev() *Item[V] {
	n := i.Left
	for n != nil && n.Deleted {
		n = n.Left
	}
	return n
}

func (i *Item[V]) Delete() {
	i.Deleted = true
}

func (d *Doc[V]) Insert(iid *IDType, origin *IDType, rightOrigin *IDType, content V) bool {
	if iid == nil {
		return false
	}
	item := &Item[V]{
		ID:          *iid,
		Parent:      d,
		Origin:      origin,
		RightOrigin: rightOrigin,
		Content:     content,
		Deleted:     false,
	}
	for r := d.Start; r != nil; r = r.Right {
		if r.ID.Equal(origin) {
			item.Left = r
		} else if r.ID.Equal(rightOrigin) {
			item.Right = r
		}
	}
	if (origin != nil && item.Left == nil) || (rightOrigin != nil && item.Right == nil) {
		return false
	}
	item.InsertInto(d)
	d.Clock = utils.Max(d.Clock, iid.Clock) + 1
	return true
}

func (d *Doc[V]) Delete(iid *IDType) bool {
	if iid == nil {
		return false
	}
	for r := d.Start; r != nil; r = r.Right {
		if r.ID.Equal(iid) {
			r.Deleted = true
			d.Clock = utils.Max(d.Clock, iid.Clock) + 1
			return true
		}
	}
	return false
}

func (d *Doc[V]) ItemAt(offset int) *Item[V] {
	if offset <= 0 {
		return nil
	}
	ret := d.Start
	offset--
	for offset > 0 && ret != nil {
		ret = ret.Next()
		offset--
	}
	return ret
}

func (d *Doc[V]) LocalInsert(offset int, content V) *Item[V] {
	left := d.ItemAt(offset)
	if left == nil && offset > 0 {
		return nil
	}
	right := d.Start
	if left != nil {
		right = left.Right
	}
	item := &Item[V]{
		ID:      IDType{Producer: d.Producer, Clock: d.Clock},
		Parent:  d,
		Left:    left,
		Right:   right,
		Content: content,
		Deleted: false,
	}
	if left != nil {
		item.Origin = &left.ID
		left.Right = item
	} else {
		d.Start = item
	}
	if right != nil {
		item.RightOrigin = &right.ID
		right.Left = item
	}
	d.Clock++
	return item
}

func (d *Doc[V]) LocalDelete(offset int) *Item[V] {
	item := d.ItemAt(offset)
	if item != nil {
		item.Deleted = true
	}
	return item
}
