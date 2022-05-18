package spec_2022

import (
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
	"github.com/zjkmxy/go-ndn/pkg/ndn"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

const (
	TimeFmt = "20060102T150405" // ISO 8601 time format
)

func _() {
	// Trait for Signature of Data
	var _ ndn.Signature = &Data{}
	// Trait for Signature of Interest
	var _ ndn.Signature = &Interest{}
	// Trait for Data of Data
	var _ ndn.Data = &Data{}
	// Trait for Interest of Interest
	var _ ndn.Interest = &Interest{}
}

type Spec struct{}

func (d *Data) SigType() ndn.SigType {
	if d.SignatureInfo == nil {
		return ndn.SignatureNone
	} else {
		return ndn.SigType(d.SignatureInfo.SignatureType)
	}
}

func (d *Data) SetSigType(sigType ndn.SigType) error {
	if sigType == ndn.SignatureNone {
		d.SignatureInfo = nil
		return nil
	} else if sigType >= 0 {
		if d.SignatureInfo == nil {
			d.SignatureInfo = &SignatureInfo{}
		}
		d.SignatureInfo.SignatureType = uint64(sigType)
		return nil
	} else {
		return ndn.ErrInvalidValue{Item: "Data.SignatureType", Value: sigType}
	}
}

func (d *Data) KeyName() enc.Name {
	if d.SignatureInfo == nil || d.SignatureInfo.KeyLocator == nil {
		return nil
	} else {
		return d.SignatureInfo.KeyLocator.Name
	}
}

func (d *Data) SetKeyName(name enc.Name) error {
	if d.SignatureInfo == nil {
		d.SignatureInfo = &SignatureInfo{}
	}
	if name != nil {
		d.SignatureInfo.KeyLocator = &KeyLocator{
			Name: enc.Name(name),
		}
	} else {
		d.SignatureInfo.KeyLocator = nil
	}
	return nil
}

func (d *Data) SigNonce() []byte {
	return nil
}

func (d *Data) SetSigNonce(nonce []byte) error {
	if d.SignatureInfo == nil {
		d.SignatureInfo = &SignatureInfo{}
	}
	d.SignatureInfo.SignatureNonce = nonce
	return nil
}

func (d *Data) SigTime() *time.Time {
	return nil
}

func (d *Data) SetSigTime(t *time.Time) error {
	if d.SignatureInfo == nil {
		d.SignatureInfo = &SignatureInfo{}
	}
	if t == nil {
		d.SignatureInfo.SignatureTime = nil
	} else {
		d.SignatureInfo.SignatureTime = utils.IdPtr(time.Duration(t.UnixMilli()) * time.Millisecond)
	}
	return nil
}

func (d *Data) SigSeqNum() *uint64 {
	return nil
}

func (d *Data) SetSigSeqNum(seq *uint64) error {
	if d.SignatureInfo == nil {
		d.SignatureInfo = &SignatureInfo{}
	}
	d.SignatureInfo.SignatureSeqNum = seq
	return nil
}

func (d *Data) Validity() (notBefore, notAfter *time.Time) {
	if d.SignatureInfo != nil && d.SignatureInfo.ValidityPeriod != nil {
		notBefore, err := time.Parse(TimeFmt, d.SignatureInfo.ValidityPeriod.NotBefore)
		if err != nil {
			return nil, nil
		}
		notAfter, err := time.Parse(TimeFmt, d.SignatureInfo.ValidityPeriod.NotAfter)
		if err != nil {
			return nil, nil
		}
		return &notBefore, &notAfter
	} else {
		return nil, nil
	}
}

func (d *Data) SetValidity(notBefore, notAfter *time.Time) error {
	if d.SignatureInfo == nil {
		d.SignatureInfo = &SignatureInfo{}
	}
	if notBefore == nil && notAfter == nil {
		d.SignatureInfo.ValidityPeriod = nil
	} else if notBefore != nil && notAfter != nil {
		d.SignatureInfo.ValidityPeriod = &ValidityPeriod{
			NotBefore: notBefore.UTC().Format(TimeFmt),
			NotAfter:  notAfter.UTC().Format(TimeFmt),
		}
	} else {
		return ndn.ErrInvalidValue{Item: "Data.ValidityPeriod", Value: nil}
	}
	return nil
}

func (d *Data) SigValue() []byte {
	if d.SignatureValue == nil {
		return nil
	} else {
		return d.SignatureValue.Join()
	}
}

func (d *Data) Signature() ndn.Signature {
	return d
}

func (d *Data) Name() enc.Name {
	return d.NameV
}

func (d *Data) ContentType() *ndn.ContentType {
	if d.MetaInfo != nil && d.MetaInfo.ContentType != nil {
		ret := ndn.ContentType(*d.MetaInfo.ContentType)
		return &ret
	} else {
		return nil
	}
}

func (d *Data) Freshness() *time.Duration {
	if d.MetaInfo != nil {
		return d.MetaInfo.FreshnessPeriod
	} else {
		return nil
	}
}

func (d *Data) FinalBlockID() *enc.Component {
	if d.MetaInfo != nil && d.MetaInfo.FinalBlockID != nil {
		ret, err := enc.ReadComponent(enc.NewBufferReader(d.MetaInfo.FinalBlockID))
		if err == nil {
			return ret
		} else {
			return nil
		}
	} else {
		return nil
	}
}

func (d *Data) Content() enc.Wire {
	return d.ContentV
}

func (t *Interest) SigType() ndn.SigType {
	if t.SignatureInfo == nil {
		return ndn.SignatureNone
	} else {
		return ndn.SigType(t.SignatureInfo.SignatureType)
	}
}

func (t *Interest) KeyName() enc.Name {
	if t.SignatureInfo == nil || t.SignatureInfo.KeyLocator == nil {
		return nil
	} else {
		return t.SignatureInfo.KeyLocator.Name
	}
}

func (t *Interest) SigNonce() []byte {
	if t.SignatureInfo != nil {
		return t.SignatureInfo.SignatureNonce
	} else {
		return nil
	}
}

func (t *Interest) SigTime() *time.Time {
	if t.SignatureInfo != nil && t.SignatureInfo.SignatureTime != nil {
		return utils.IdPtr(time.UnixMilli(t.SignatureInfo.SignatureTime.Milliseconds()))
	} else {
		return nil
	}
}

func (t *Interest) SigSeqNum() *uint64 {
	if t.SignatureInfo != nil {
		return t.SignatureInfo.SignatureSeqNum
	} else {
		return nil
	}
}

func (t *Interest) Validity() (notBefore, notAfter *time.Time) {
	return nil, nil
}

func (t *Interest) SigValue() []byte {
	return t.SignatureValue.Join()
}

func (t *Interest) Signature() ndn.Signature {
	return t
}

func (t *Interest) Name() enc.Name {
	return t.NameV
}

func (t *Interest) CanBePrefix() bool {
	return t.CanBePrefixV
}

func (t *Interest) MustBeFresh() bool {
	return t.MustBeFreshV
}

func (t *Interest) ForwardingHint() []enc.Name {
	if t.ForwardingHintV == nil {
		return nil
	}
	return t.ForwardingHintV.Names
}

func (t *Interest) Nonce() *uint64 {
	if t.NonceV == nil {
		return nil
	} else {
		return utils.IdPtr(uint64(*t.NonceV))
	}
}

func (t *Interest) Lifetime() *time.Duration {
	return t.InterestLifetimeV
}

func (t *Interest) HopLimit() *uint {
	if t.HopLimitV == nil {
		return nil
	} else {
		return utils.IdPtr(uint(*t.HopLimitV))
	}
}

func (t *Interest) AppParam() enc.Wire {
	return t.ApplicationParameters
}

func (_ Spec) MakeData(
	name enc.Name, config *ndn.DataConfig, content enc.Wire, signer ndn.Signer,
) (enc.Wire, enc.Wire, error) {
	// Create Data packet.
	if name == nil {
		return nil, nil, ndn.ErrInvalidValue{Item: "Data.Name", Value: nil}
	}
	if config == nil {
		return nil, nil, ndn.ErrInvalidValue{Item: "Data.DataConfig", Value: nil}
	}
	contentType := (*uint64)(nil)
	if config.ContentType != nil {
		contentType = utils.IdPtr(uint64(*config.ContentType))
	}
	finalBlock := []byte(nil)
	if config.FinalBlockID != nil {
		finalBlock = config.FinalBlockID.Bytes()
	}
	data := &Data{
		NameV: name,
		MetaInfo: &MetaInfo{
			ContentType:     contentType,
			FreshnessPeriod: config.Freshness,
			FinalBlockID:    finalBlock,
		},
		ContentV:       content,
		SignatureInfo:  nil,
		SignatureValue: nil,
	}
	packet := &Packet{
		Data: data,
	}

	// Fill-in SignatureInfo.
	if signer != nil {
		sigConfig, err := signer.SigInfo(data)
		if err != nil {
			return nil, nil, err
		}
		if sigConfig != nil && sigConfig.Type != ndn.SignatureNone {
			if sigConfig.Nonce != nil {
				return nil, nil, ndn.ErrNotSupported{Item: "Data.SignatureInfo.SignatureNonce"}
			}
			if sigConfig.SeqNum != nil {
				return nil, nil, ndn.ErrNotSupported{Item: "Data.SignatureInfo.SignatureSeqNum"}
			}
			if sigConfig.SigTime != nil {
				return nil, nil, ndn.ErrNotSupported{Item: "Data.SignatureInfo.SignatureTime"}
			}
			if sigConfig.Type != ndn.SignatureDigestSha256 {
				if sigConfig.KeyName == nil {
					return nil, nil, ndn.ErrInvalidValue{Item: "Data.SignatureInfo.KeyLocator", Value: nil}
				}
				data.SignatureInfo = &SignatureInfo{
					SignatureType: uint64(sigConfig.Type),
					KeyLocator: &KeyLocator{
						Name: sigConfig.KeyName,
					},
				}
			} else {
				data.SignatureInfo = &SignatureInfo{SignatureType: uint64(sigConfig.Type)}
			}

			if sigConfig.NotBefore != nil || sigConfig.NotAfter != nil {
				if sigConfig.NotBefore == nil {
					return nil, nil, ndn.ErrInvalidValue{Item: "Data.SignatureInfo.Validity.NotBefore", Value: nil}
				}
				if sigConfig.NotAfter == nil {
					return nil, nil, ndn.ErrInvalidValue{Item: "Data.SignatureInfo.Validity.NotBefore", Value: nil}
				}
				data.SignatureInfo.ValidityPeriod = &ValidityPeriod{
					NotBefore: sigConfig.NotBefore.UTC().Format(TimeFmt),
					NotAfter:  sigConfig.NotAfter.UTC().Format(TimeFmt),
				}
			}

			// Encode packet.
			encoder := PacketEncoder{
				Data_encoder: DataEncoder{
					SignatureValue_estLen: signer.EstimateSize(),
				},
			}
			if encoder.Data_encoder.SignatureValue_estLen >= 253 {
				return nil, nil, ndn.ErrNotSupported{Item: "Too long signature value is not supported"}
			}
			encoder.Init(packet)
			wire := encoder.Encode(packet)
			if wire == nil {
				return nil, nil, ndn.ErrFailedToEncode
			}
			sigCovered := encoder.Data_encoder.sigCovered
			// Compute signature
			// Since PacketEncoder only adds a TL, Data_encoder.SignatureValue_wireIdx is still valid
			if encoder.Data_encoder.SignatureValue_wireIdx >= 0 {
				sigVal, err := signer.ComputeSigValue(sigCovered)
				if err != nil {
					return nil, nil, err
				}
				if uint(len(sigVal)) > encoder.Data_encoder.SignatureValue_estLen {
					return nil, nil, ndn.ErrNotSupported{Item: "Too long signature value is not supported"}
				}
				wire[encoder.Data_encoder.SignatureValue_wireIdx] = sigVal
				// Fix SignatureValue length
				buf := wire[encoder.Data_encoder.SignatureValue_wireIdx-1]
				buf[len(buf)-1] = byte(len(sigVal))
				// Fix packet length
				shrink := int(encoder.Data_encoder.SignatureValue_estLen) - len(sigVal)
				wire[0] = enc.ShrinkLength(wire[0], shrink)
			}

			return wire, sigCovered, nil
		}
	}
	// Encode packet without signature
	encoder := PacketEncoder{
		Data_encoder: DataEncoder{
			SignatureValue_estLen: 0,
		},
	}
	encoder.Init(packet)
	wire := encoder.Encode(packet)
	if wire == nil {
		return nil, nil, ndn.ErrFailedToEncode
	}
	return wire, nil, nil
}

func (_ Spec) ReadData(reader enc.ParseReader) (ndn.Data, enc.Wire, error) {
	context := PacketParsingContext{}
	context.Init()
	ret, err := context.Parse(reader, false)
	if err != nil {
		return nil, nil, err
	}
	if ret.Data == nil {
		return nil, nil, ndn.ErrWrongType
	}
	return ret.Data, context.Data_context.sigCovered, nil
}
