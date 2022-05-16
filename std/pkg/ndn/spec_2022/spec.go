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

func (d *Data) Nonce() []byte {
	if d.SignatureInfo != nil {
		return d.SignatureInfo.SignatureNonce
	} else {
		return nil
	}
}

func (d *Data) SetNonce(nonce []byte) error {
	if d.SignatureInfo == nil {
		d.SignatureInfo = &SignatureInfo{}
	}
	d.SignatureInfo.SignatureNonce = nonce
	return nil
}

func (d *Data) SigTime() *time.Time {
	if d.SignatureInfo != nil && d.SignatureInfo.SignatureTime != nil {
		return utils.ConstPtr(time.UnixMilli(d.SignatureInfo.SignatureTime.Milliseconds()))
	} else {
		return nil
	}
}

func (d *Data) SetSigTime(t *time.Time) error {
	if d.SignatureInfo == nil {
		d.SignatureInfo = &SignatureInfo{}
	}
	if t == nil {
		d.SignatureInfo.SignatureTime = nil
	} else {
		d.SignatureInfo.SignatureTime = utils.ConstPtr(time.Duration(t.UnixMilli()) * time.Millisecond)
	}
	return nil
}

func (d *Data) SeqNum() *uint64 {
	if d.SignatureInfo != nil {
		return d.SignatureInfo.SignatureSeqNum
	} else {
		return nil
	}
}

func (d *Data) SetSeqNum(seq *uint64) error {
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

func (d *Data) Value() []byte {
	return d.SignatureValue.Join()
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
		ret, _ := enc.ReadComponent(enc.NewBufferReader(d.MetaInfo.FinalBlockID))
		return ret
	} else {
		return nil
	}
}

func (d *Data) Content() enc.Wire {
	return d.ContentV
}

func (_ Spec) MakeData(name enc.Name, config *ndn.DataConfig,
	content enc.Wire, signer ndn.Signer) (enc.Wire, enc.Wire, error) {
	ret := &Data{}
	panic(ret) // TODO
}
