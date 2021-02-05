/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package mgmt

import (
	"errors"

	"github.com/eric135/YaNFD/ndn/tlv"
)

// ControlResponse represents the response from a management command.
type ControlResponse struct {
	StatusCode uint64
	StatusText string
	Body       *tlv.Block
}

// MakeControlResponse creates a ControlResponse.
func MakeControlResponse(statusCode uint64, statusText string, body *tlv.Block) *ControlResponse {
	c := new(ControlResponse)
	c.StatusCode = statusCode
	c.StatusText = statusText
	c.Body = body
	return c
}

// DecodeControlResponse decodes a ControlResponse from the wire.
func DecodeControlResponse(wire *tlv.Block) (*ControlResponse, error) {
	c := new(ControlResponse)

	wire.Parse()
	var err error
	hasStatusCode := false
	hasStatusText := false
	for _, elem := range wire.Subelements() {
		switch elem.Type() {
		case tlv.StatusCode:
			if hasStatusCode {
				return nil, errors.New("Duplicate StatusCode")
			}
			c.StatusCode, err = tlv.DecodeNNIBlock(elem)
			hasStatusCode = true
			if err != nil {
				return nil, errors.New("Unable to decode StatusCode: " + err.Error())
			}
		case tlv.StatusText:
			if hasStatusText {
				return nil, errors.New("Duplicate StatusText")
			}
			c.StatusText = string(elem.Value())
			hasStatusText = true
		default:
			// Make as body
			c.Body = elem
			break
		}
	}

	if !hasStatusCode {
		return nil, errors.New("Missing StatusCode")
	}

	if !hasStatusText {
		return nil, errors.New("Missing StatusText")
	}

	return c, nil
}

// Encode encodes a ControlResponse.
func (c *ControlResponse) Encode() (*tlv.Block, error) {
	wire := tlv.NewEmptyBlock(tlv.ControlResponse)

	wire.Append(tlv.EncodeNNIBlock(tlv.StatusCode, c.StatusCode))
	wire.Append(tlv.NewBlock(tlv.StatusText, []byte(c.StatusText)))
	if c.Body != nil {
		wire.Append(c.Body)
	}

	wire.Encode()
	return wire, nil
}
