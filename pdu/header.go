// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package pdu

import (
	"encoding/binary"
	"fmt"
)

const (
	// HeaderSize defines the total size of a header packet.
	HeaderSize = 20
)

// Header defines a pdu packet header
type Header struct {
	Version       byte
	Type          Type
	Flags         Flags
	SessionID     uint32
	TransactionID uint32
	PacketID      uint32
	PayloadLength uint32
}

// MarshalBinary returns the pdu header as a slice of bytes.
func (h *Header) MarshalBinary() ([]byte, error) {
	result := make([]byte, HeaderSize)
	result[0] = h.Version
	result[1] = byte(h.Type)
	result[2] = byte(h.Flags)
	// result[3] is reserved padding byte (0x00)
	binary.LittleEndian.PutUint32(result[4:], h.SessionID)
	binary.LittleEndian.PutUint32(result[8:], h.TransactionID)
	binary.LittleEndian.PutUint32(result[12:], h.PacketID)
	binary.LittleEndian.PutUint32(result[16:], h.PayloadLength)
	return result, nil
}

// UnmarshalBinary sets the header structure from the provided slice of bytes.
func (h *Header) UnmarshalBinary(data []byte) error {
	if len(data) < HeaderSize {
		return fmt.Errorf("not enough bytes (%d) to unmarshal the header (%d)", len(data), HeaderSize)
	}

	h.Version, h.Type, h.Flags = data[0], Type(data[1]), Flags(data[2])

	h.SessionID = binary.LittleEndian.Uint32(data[4:])
	h.TransactionID = binary.LittleEndian.Uint32(data[8:])
	h.PacketID = binary.LittleEndian.Uint32(data[12:])
	h.PayloadLength = binary.LittleEndian.Uint32(data[16:])

	return nil
}

func (h *Header) String() string {
	return "(header " + h.Type.String() + ")"
}
