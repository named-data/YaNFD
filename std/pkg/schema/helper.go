package schema

import (
	"fmt"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
)

// PropertySet is an internal function which sets the property `propName` (stored at `ptr`) to `value`
func PropertySet[T any](ptr *T, propName PropKey, value any) error {
	if v, ok := value.(T); ok {
		*ptr = v
		return nil
	} else {
		return ndn.ErrInvalidValue{Item: string(propName), Value: value}
	}
}

// AddEventListener add `callback` to the event `propKey` of `node`
func AddEventListener[T any](node NTNode, propName PropKey, callback T) error {
	evt, ok := node.Get(propName).(*Event[*T])
	if !ok || evt == nil {
		return fmt.Errorf("invalid event: %s", propName)
	}
	evt.Add(&callback)
	return nil
}

type NeedResult struct {
	Status  ndn.InterestResult
	Content enc.Wire
}

func (r NeedResult) Get() (ndn.InterestResult, enc.Wire) {
	return r.Status, r.Content
}
