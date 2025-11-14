// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package pdu

import (
	"encoding/binary"
)

// OctetString defines the pdu description packet.
type OctetString struct {
	Text string
}

// MarshalBinary returns the pdu packet as a slice of bytes.
func (o *OctetString) MarshalBinary() ([]byte, error) {
	l := len(o.Text)
	pad := (4 - (l % 4)) & 3
	result := make([]byte, 4+l+pad)
	binary.LittleEndian.PutUint32(result[0:], uint32(l))
	copy(result[4:], o.Text)
	// padding bytes are already zeroed by make
	return result, nil
}

// UnmarshalBinary sets the packet structure from the provided slice of bytes.
func (o *OctetString) UnmarshalBinary(data []byte) error {
	length := binary.LittleEndian.Uint32(data[0:])
	o.Text = string(data[4 : 4+length])
	return nil
}
