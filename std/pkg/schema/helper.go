package schema

import (
	"fmt"

	"github.com/zjkmxy/go-ndn/pkg/ndn"
)

func PropertySet[T any](ptr *T, propName PropKey, value any) error {
	if v, ok := value.(T); ok {
		*ptr = v
		return nil
	} else {
		return ndn.ErrInvalidValue{Item: string(propName), Value: value}
	}
}

func AddEventListener[T any](node NTNode, propName PropKey, callback T) error {
	evt, ok := node.Get(propName).(*Event[*T])
	if !ok || evt == nil {
		return fmt.Errorf("invalid event: %s", propName)
	}
	evt.Add(&callback)
	return nil
}
