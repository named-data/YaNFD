package face

import (
	"errors"
	"io"

	defn "github.com/named-data/YaNFD/defn"
	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

func readTlvStream(
	reader io.Reader,
	onFrame func([]byte),
	ignoreError func(error) bool,
) error {
	recvBuf := make([]byte, defn.MaxNDNPacketSize*32)
	recvOff := 0
	tlvOff := 0

	for {
		readSize, err := reader.Read(recvBuf[recvOff:])
		recvOff += readSize
		if err != nil {
			if ignoreError != nil && ignoreError(err) {
				continue
			}
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		// Determine whether valid packet received
		for {
			rdr := enc.NewBufferReader(recvBuf[tlvOff:recvOff])

			typ, err := enc.ReadTLNum(rdr)
			if err != nil {
				// Probably incomplete packet
				break
			}

			len, err := enc.ReadTLNum(rdr)
			if err != nil {
				// Probably incomplete packet
				break
			}

			tlvSize := typ.EncodingLength() + len.EncodingLength() + int(len)

			if recvOff-tlvOff >= tlvSize {
				// Packet was successfully received, send up to link service
				onFrame(recvBuf[tlvOff : tlvOff+tlvSize])
				tlvOff += tlvSize
			} else if recvOff-tlvOff > defn.MaxNDNPacketSize {
				// Invalid packet, something went wrong
				return errors.New("received too much data without valid TLV block")
			} else {
				// Incomplete packet (for sure)
				break
			}
		}

		// If less than one packet space remains in buffer, shift to beginning
		if recvOff-tlvOff < defn.MaxNDNPacketSize {
			copy(recvBuf, recvBuf[tlvOff:recvOff])
			recvOff -= tlvOff
			tlvOff = 0
		}
	}
}
