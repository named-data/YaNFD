package schema

import (
	"crypto/sha256"
	"fmt"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

// SegmentedNode handles the segmentation and reassembly
type SegmentedNode struct {
	BaseNodeImpl

	ContentType         ndn.ContentType
	Freshness           time.Duration
	ValidDur            time.Duration
	Lifetime            time.Duration
	MustBeFresh         bool
	SegmentSize         uint64
	MaxRetriesOnFailure uint64
	Pipeline            string
}

func (n *SegmentedNode) NodeImplTrait() NodeImpl {
	return n
}

func CreateSegmentedNode(node *Node) NodeImpl {
	ret := &SegmentedNode{
		BaseNodeImpl: BaseNodeImpl{
			Node:        node,
			OnAttachEvt: &EventTarget{},
			OnDetachEvt: &EventTarget{},
		},
		ContentType:         ndn.ContentTypeBlob,
		MustBeFresh:         true,
		Lifetime:            4 * time.Second,
		ValidDur:            876000 * time.Hour,
		Freshness:           10 * time.Second,
		SegmentSize:         8000,
		MaxRetriesOnFailure: 15,
		Pipeline:            "SinglePacket",
	}
	path, _ := enc.NamePatternFromStr("<seg=segmentNumber>")
	node.PutNode(path, LeafNodeDesc)
	return ret
}

func (n *SegmentedNode) Provide(mNode MatchedNode, content enc.Wire, needManifest bool) []enc.Buffer {
	if mNode.Node != n.Node {
		panic("NTSchema tree compromised.")
	}

	var wireIdx, bufferIdx int = 0, 0
	var ret []enc.Buffer = nil
	// Segmentation
	segCnt := (content.Length() + n.SegmentSize - 1) / n.SegmentSize
	if needManifest {
		ret = make([]enc.Buffer, segCnt)
	}
	newName := make(enc.Name, len(mNode.Name)+1)
	copy(newName, mNode.Name)

	dataCfg := &ndn.DataConfig{
		ContentType:  utils.IdPtr(n.ContentType),
		Freshness:    utils.IdPtr(n.Freshness),
		FinalBlockID: utils.IdPtr(enc.NewSegmentComponent(segCnt - 1)),
	}

	for i := uint64(0); i < segCnt; i++ {
		newName[len(mNode.Name)] = enc.NewSegmentComponent(i)
		pktContent := enc.Wire{}
		remSize := n.SegmentSize
		for remSize > 0 && wireIdx < len(content) && bufferIdx < len(content[wireIdx]) {
			curSize := int(utils.Min(uint64(len(content[wireIdx])-bufferIdx), remSize))
			pktContent = append(pktContent, content[wireIdx][bufferIdx:bufferIdx+curSize])
			bufferIdx += curSize
			remSize -= uint64(curSize)
			if bufferIdx >= len(content[wireIdx]) {
				wireIdx += 1
				bufferIdx = 0
			}
		}
		// generate the data packet
		newMNode := mNode.Refine(newName)
		dataWire := newMNode.Call("Provide", pktContent, dataCfg).(enc.Wire)

		// compute implicit sha256 for manifest if needed
		if needManifest {
			h := sha256.New()
			for _, buf := range dataWire {
				h.Write(buf)
			}
			ret[i] = h.Sum(nil)
		}
	}
	mNode.Logger("SegmentedNode").Debugf("Segmented into %d segments \n", segCnt)
	return ret
}

func (n *SegmentedNode) NeedCallback(mNode MatchedNode, callback Callback, manifest []enc.Name) error {
	if mNode.Node != n.Node {
		panic("NTSchema tree compromised.")
	}
	switch n.Pipeline {
	case "SinglePacket":
		go n.SinglePacketPipeline(mNode, callback, manifest)
		return nil
	}
	mNode.Logger("SegmentedNode").Errorf("unrecognized pipeline: %s", n.Pipeline)
	return fmt.Errorf("unrecognized pipeline: %s", n.Pipeline)
}

func (n *SegmentedNode) NeedChan(mNode MatchedNode, manifest []enc.Name) chan NeedResult {
	ret := make(chan NeedResult, 1)
	callback := func(event *Event) any {
		result := NeedResult{
			Status:      *event.NeedStatus,
			Content:     event.Content,
			Data:        event.Data,
			ValidResult: event.ValidResult,
			NackReason:  event.NackReason,
		}
		ret <- result
		close(ret)
		return nil
	}
	n.NeedCallback(mNode, callback, manifest)
	return ret
}

func (n *SegmentedNode) SinglePacketPipeline(mNode MatchedNode, callback Callback, manifest []enc.Name) {
	if len(manifest) > 0 {
		panic("TODO: manifest not supported")
	}
	fragments := enc.Wire{}
	var lastData ndn.Data
	var lastNackReason *uint64
	var lastValidationRes *ValidRes
	var lastNeedStatus ndn.InterestResult
	logger := mNode.Logger("SegmentedNode")
	newName := make(enc.Name, len(mNode.Name)+1)
	copy(newName, mNode.Name)
	succeeded := true
	for i := uint64(0); succeeded; i++ {
		newName[len(mNode.Name)] = enc.NewSegmentComponent(i)
		newMNode := mNode.Refine(newName)
		succeeded = false
		for j := 0; !succeeded && j < int(n.MaxRetriesOnFailure); j++ {
			logger.Debugf("Fetching the %d fragment [the %d trial]", i, j)
			result := <-newMNode.Call("NeedChan").(chan NeedResult)
			lastData = result.Data
			lastNackReason = result.NackReason
			lastValidationRes = result.ValidResult
			lastNeedStatus = result.Status
			switch result.Status {
			case ndn.InterestResultData:
				fragments = append(fragments, result.Content...)
				succeeded = true
			}
		}
		if succeeded && lastData.FinalBlockID().Compare(newName[len(mNode.Name)]) == 0 {
			// In the last segment, finalBlockId equals the last name component
			break
		}
	}

	event := &Event{
		TargetNode:  n.Node,
		Target:      &mNode,
		Content:     fragments,
		Data:        lastData,
		NackReason:  lastNackReason,
		ValidResult: lastValidationRes,
	}
	if succeeded {
		event.NeedStatus = utils.IdPtr(ndn.InterestResultData)
	} else {
		event.NeedStatus = utils.IdPtr(lastNeedStatus)
	}
	callback(event)
}

func (n *SegmentedNode) CastTo(ptr any) any {
	switch ptr.(type) {
	case (*SegmentedNode):
		return n
	case (*BaseNodeImpl):
		return &(n.BaseNodeImpl)
	default:
		return nil
	}
}

// RdrNode handles the version discovery
type RdrNode struct {
	BaseNodeImpl

	MetaFreshness     time.Duration
	MaxRetriesForMeta uint64
}

func (n *RdrNode) NodeImplTrait() NodeImpl {
	return n
}

func CreateRdrNode(node *Node) NodeImpl {
	ret := &RdrNode{
		BaseNodeImpl: BaseNodeImpl{
			Node:        node,
			OnAttachEvt: &EventTarget{},
			OnDetachEvt: &EventTarget{},
		},
		MetaFreshness:     10 * time.Millisecond,
		MaxRetriesForMeta: 15,
	}
	path, _ := enc.NamePatternFromStr("<v=versionNumber>")
	node.PutNode(path, SegmentedNodeDesc)
	path, _ = enc.NamePatternFromStr("32=metadata")
	node.PutNode(path, ExpressPointDesc)
	path, _ = enc.NamePatternFromStr("32=metadata/<v=versionNumber>/seg=0")
	node.PutNode(path, LeafNodeDesc)
	return ret
}

func (n *RdrNode) Provide(mNode MatchedNode, content enc.Wire) uint64 {
	if mNode.Node != n.Node {
		panic("NTSchema tree compromised.")
	}

	// NOTE: This version of RDR node puts the metadata into storage (same as python-ndn's cmd_serve_rdrcontent).
	// It is possible to serve metadata packet in real time, but needs special handling for matching
	// There are two ways:
	// 1. Ask the storage to provide a function (via Node's event) to search with version
	// 2. Have a mapping between matching and version
	timer := mNode.Node.Engine().Timer()
	ver := utils.MakeTimestamp(timer.Now())
	nameLen := len(mNode.Name)
	metaName := make(enc.Name, nameLen+3)
	copy(metaName, mNode.Name) // Note this does not actually copies the component values
	metaName[nameLen] = enc.NewStringComponent(32, "metadata")
	metaName[nameLen+1] = enc.NewVersionComponent(ver)
	metaName[nameLen+2] = enc.NewSegmentComponent(0)
	metaMNode := mNode.Refine(metaName)

	dataName := make(enc.Name, nameLen+1)
	copy(dataName, mNode.Name)
	dataName[nameLen] = enc.NewVersionComponent(ver)
	dataMNode := mNode.Refine(dataName)

	// generate metadata
	metaDataCfg := &ndn.DataConfig{
		ContentType:  utils.IdPtr(ndn.ContentTypeBlob),
		Freshness:    utils.IdPtr(n.MetaFreshness),
		FinalBlockID: utils.IdPtr(enc.NewSegmentComponent(0)),
	}
	metaMNode.Call("Provide", enc.Wire{dataName.Bytes()}, metaDataCfg)

	// generate segmented data
	dataMNode.Call("Provide", content)

	return ver
}

func (n *RdrNode) NeedCallback(mNode MatchedNode, callback Callback, version *uint64) {
	if mNode.Node != n.Node {
		panic("NTSchema tree compromised.")
	}

	go func() {
		nameLen := len(mNode.Name)
		logger := mNode.Logger("RdrNode")
		var err error = nil
		var fullName enc.Name
		var lastResult NeedResult

		if version == nil {
			// Fetch the version
			metaIntName := make(enc.Name, nameLen+1)
			copy(metaIntName, mNode.Name)
			metaIntName[nameLen] = enc.NewStringComponent(32, "metadata")
			epMNode := mNode.Refine(metaIntName)

			succeeded := false
			for j := 0; !succeeded && j < int(n.MaxRetriesForMeta); j++ {
				logger.Debugf("Fetching the metadata packet [the %d trial]", j)
				lastResult = <-epMNode.Call("NeedChan").(chan NeedResult)
				switch lastResult.Status {
				case ndn.InterestResultData:
					succeeded = true
					fullName, err = enc.NameFromBytes(lastResult.Content.Join())
					if err != nil {
						logger.Errorf("Unable to extract name from the metadata packet: %v\n", err)
						lastResult.Status = ndn.InterestResultError
					}
				}
			}

			if !succeeded || lastResult.Status == ndn.InterestResultError || !mNode.Name.IsPrefix(fullName) {
				event := &Event{
					TargetNode:  n.Node,
					Target:      &mNode,
					Data:        lastResult.Data,
					NackReason:  lastResult.NackReason,
					ValidResult: lastResult.ValidResult,
					NeedStatus:  utils.IdPtr(lastResult.Status),
					Content:     nil,
				}
				if succeeded {
					event.Error = fmt.Errorf("the metadata packet is malformed: %v", err)
				} else {
					event.Error = fmt.Errorf("unable to fetch the metadata packet")
				}
				callback(event)
				return
			}
		} else {
			fullName = make(enc.Name, nameLen+1)
			fullName[nameLen] = enc.NewVersionComponent(*version)
		}

		segMNode := mNode.Refine(fullName)
		segMNode.Call("Need", callback)
	}()
}

func (n *RdrNode) NeedChan(mNode MatchedNode, version *uint64) chan NeedResult {
	ret := make(chan NeedResult, 1)
	callback := func(event *Event) any {
		result := NeedResult{
			Status:      *event.NeedStatus,
			Content:     event.Content,
			Data:        event.Data,
			ValidResult: event.ValidResult,
			NackReason:  event.NackReason,
		}
		ret <- result
		close(ret)
		return nil
	}
	n.NeedCallback(mNode, callback, version)
	return ret
}

func (n *RdrNode) CastTo(ptr any) any {
	switch ptr.(type) {
	case (*RdrNode):
		return n
	case (*BaseNodeImpl):
		return &(n.BaseNodeImpl)
	default:
		return nil
	}
}

type GeneralObjNode struct {
}

var (
	RdrNodeDesc       *NodeImplDesc
	SegmentedNodeDesc *NodeImplDesc
)

func initRdrNodes() {
	SegmentedNodeDesc = &NodeImplDesc{
		ClassName: "SegmentedNode",
		Properties: map[PropKey]PropertyDesc{
			"ContentType":         DefaultPropertyDesc("ContentType"),
			"Lifetime":            TimePropertyDesc("Lifetime"),
			"Freshness":           TimePropertyDesc("Freshness"),
			"ValidDuration":       TimePropertyDesc("ValidDur"),
			"MustBeFresh":         DefaultPropertyDesc("MustBeFresh"),
			"SegmentSize":         DefaultPropertyDesc("SegmentSize"),
			"MaxRetriesOnFailure": DefaultPropertyDesc("MaxRetriesOnFailure"),
			"Pipeline":            DefaultPropertyDesc("Pipeline"),
		},
		Events: map[PropKey]EventGetter{
			PropOnAttach: DefaultEventTarget(PropOnAttach), // Inherited from base
			PropOnDetach: DefaultEventTarget(PropOnDetach), // Inherited from base
		},
		Functions: map[string]NodeFunc{
			"Provide": func(mNode MatchedNode, args ...any) any {
				if len(args) < 1 || len(args) > 2 {
					err := fmt.Errorf("SegmentedNode.Provide requires 1~2 arguments but got %d", len(args))
					mNode.Logger("SegmentedNode").Error(err.Error())
					return err
				}
				content, ok := args[0].(enc.Wire)
				if !ok && args[0] != nil {
					err := ndn.ErrInvalidValue{Item: "content", Value: args[0]}
					mNode.Logger("SegmentedNode").Error(err.Error())
					return err
				}
				var needManifest bool = false
				if len(args) >= 2 {
					needManifest, ok = args[1].(bool)
					if !ok && args[1] != nil {
						err := ndn.ErrInvalidValue{Item: "needManifest", Value: args[0]}
						mNode.Logger("SegmentedNode").Error(err.Error())
						return err
					}
				}
				return QueryInterface[*SegmentedNode](mNode.Node).Provide(mNode, content, needManifest)
			},
			"Need": func(mNode MatchedNode, args ...any) any {
				if len(args) < 1 || len(args) > 2 {
					err := fmt.Errorf("SegmentedNode.Need requires 1~2 arguments but got %d", len(args))
					mNode.Logger("SegmentedNode").Error(err.Error())
					return err
				}
				callback, ok := args[0].(Callback)
				if !ok {
					err := ndn.ErrInvalidValue{Item: "callback", Value: args[0]}
					mNode.Logger("SegmentedNode").Error(err.Error())
					return err
				}
				var manifest []enc.Name = nil
				if len(args) >= 2 {
					manifest, ok = args[1].([]enc.Name)
					if !ok && args[1] != nil {
						err := ndn.ErrInvalidValue{Item: "manifest", Value: args[0]}
						mNode.Logger("SegmentedNode").Error(err.Error())
						return err
					}
				}
				return QueryInterface[*SegmentedNode](mNode.Node).NeedCallback(mNode, callback, manifest)
			},
			"NeedChan": func(mNode MatchedNode, args ...any) any {
				if len(args) > 1 {
					err := fmt.Errorf("SegmentedNode.NeedChan requires 0~1 arguments but got %d", len(args))
					mNode.Logger("SegmentedNode").Error(err.Error())
					return err
				}
				var manifest []enc.Name = nil
				var ok bool = true
				if len(args) >= 1 {
					manifest, ok = args[0].([]enc.Name)
					if !ok && args[0] != nil {
						err := ndn.ErrInvalidValue{Item: "manifest", Value: args[0]}
						mNode.Logger("SegmentedNode").Error(err.Error())
						return err
					}
				}
				return QueryInterface[*SegmentedNode](mNode.Node).NeedChan(mNode, manifest)
			},
		},
		Create: CreateSegmentedNode,
	}
	RegisterNodeImpl(SegmentedNodeDesc)

	RdrNodeDesc = &NodeImplDesc{
		ClassName: "RdrNode",
		Properties: map[PropKey]PropertyDesc{
			"MetaFreshness":       TimePropertyDesc("MetaFreshness"),
			"MaxRetriesForMeta":   DefaultPropertyDesc("MaxRetriesForMeta"),
			"MetaLifetime":        SubNodePropertyDesc("32=metadata", PropLifetime),
			"ContentType":         SubNodePropertyDesc("<v=versionNumber>", "ContentType"),
			"Lifetime":            SubNodePropertyDesc("<v=versionNumber>", "Lifetime"),
			"Freshness":           SubNodePropertyDesc("<v=versionNumber>", "Freshness"),
			"ValidDuration":       SubNodePropertyDesc("<v=versionNumber>", "ValidDuration"),
			"MustBeFresh":         SubNodePropertyDesc("<v=versionNumber>", "MustBeFresh"),
			"SegmentSize":         SubNodePropertyDesc("<v=versionNumber>", "SegmentSize"),
			"MaxRetriesOnFailure": SubNodePropertyDesc("<v=versionNumber>", "MaxRetriesOnFailure"),
			"Pipeline":            SubNodePropertyDesc("<v=versionNumber>", "Pipeline"),
		},
		Events: map[PropKey]EventGetter{
			PropOnAttach: DefaultEventTarget(PropOnAttach), // Inherited from base
			PropOnDetach: DefaultEventTarget(PropOnDetach), // Inherited from base
		},
		Functions: map[string]NodeFunc{
			"Provide": func(mNode MatchedNode, args ...any) any {
				if len(args) != 1 {
					err := fmt.Errorf("RdrNode.Provide requires 1 arguments but got %d", len(args))
					mNode.Logger("RdrNode").Error(err.Error())
					return err
				}
				content, ok := args[0].(enc.Wire)
				if !ok && args[0] != nil {
					err := ndn.ErrInvalidValue{Item: "content", Value: args[0]}
					mNode.Logger("RdrNode").Error(err.Error())
					return err
				}
				return QueryInterface[*RdrNode](mNode.Node).Provide(mNode, content)
			},
			"Need": func(mNode MatchedNode, args ...any) any {
				if len(args) < 1 || len(args) > 2 {
					err := fmt.Errorf("RdrNode.Need requires 1~2 arguments but got %d", len(args))
					mNode.Logger("RdrNode").Error(err.Error())
					return err
				}
				callback, ok := args[0].(Callback)
				if !ok {
					err := ndn.ErrInvalidValue{Item: "callback", Value: args[0]}
					mNode.Logger("RdrNode").Error(err.Error())
					return err
				}
				var version *uint64 = nil
				if len(args) >= 2 {
					version, ok = args[1].(*uint64)
					if !ok && args[1] != nil {
						err := ndn.ErrInvalidValue{Item: "version", Value: args[0]}
						mNode.Logger("RdrNode").Error(err.Error())
						return err
					}
				}
				QueryInterface[*RdrNode](mNode.Node).NeedCallback(mNode, callback, version)
				return nil
			},
			"NeedChan": func(mNode MatchedNode, args ...any) any {
				if len(args) > 1 {
					err := fmt.Errorf("RdrNode.NeedChan requires 0~1 arguments but got %d", len(args))
					mNode.Logger("RdrNode").Error(err.Error())
					return err
				}
				var version *uint64 = nil
				var ok bool = true
				if len(args) >= 1 {
					version, ok = args[0].(*uint64)
					if !ok && args[0] != nil {
						err := ndn.ErrInvalidValue{Item: "version", Value: args[0]}
						mNode.Logger("RdrNode").Error(err.Error())
						return err
					}
				}
				return QueryInterface[*RdrNode](mNode.Node).NeedChan(mNode, version)
			},
		},
		Create: CreateRdrNode,
	}
	RegisterNodeImpl(RdrNodeDesc)
}
