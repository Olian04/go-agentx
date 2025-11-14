// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package pdu

import (
	"fmt"
	"encoding/binary"
)

// HeaderPacket defines a container structure for a header and a packet.
type HeaderPacket struct {
	Header *Header
	Packet Packet
}

// MarshalBinary returns the pdu packet as a slice of bytes.
func (hp *HeaderPacket) MarshalBinary() ([]byte, error) {
	payloadBytes, err := hp.Packet.MarshalBinary()
	if err != nil {
		return nil, err
	}

	hp.Header.Version = 1
	hp.Header.Type = hp.Packet.Type()
	hp.Header.PayloadLength = uint32(len(payloadBytes))

	// Single allocation: header (20 bytes) + payload
	result := make([]byte, HeaderSize+len(payloadBytes))
	// Write header inline (mirror Header.MarshalBinary but avoid extra alloc)
	result[0] = hp.Header.Version
	result[1] = byte(hp.Header.Type)
	result[2] = byte(hp.Header.Flags)
	// result[3] reserved
	binary.LittleEndian.PutUint32(result[4:], hp.Header.SessionID)
	binary.LittleEndian.PutUint32(result[8:], hp.Header.TransactionID)
	binary.LittleEndian.PutUint32(result[12:], hp.Header.PacketID)
	binary.LittleEndian.PutUint32(result[16:], hp.Header.PayloadLength)
	// Copy payload
	copy(result[HeaderSize:], payloadBytes)
	return result, nil
}

func (hp *HeaderPacket) String() string {
	return fmt.Sprintf("[head %v, body %v]", hp.Header, hp.Packet)
}
