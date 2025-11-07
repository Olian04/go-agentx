// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package pdu

import (
	"encoding/binary"
	"fmt"
)

// GetBulk defines the pdu get-bulk packet.
type GetBulk struct {
	NonRepeaters   uint16
	MaxRepetitions uint16
	SearchRanges   Ranges
}

// Type returns the pdu packet type.
func (g *GetBulk) Type() Type {
	return TypeGetBulk
}

// MarshalBinary returns the pdu packet as a slice of bytes.
func (g *GetBulk) MarshalBinary() ([]byte, error) {
	result := make([]byte, 4)
	binary.LittleEndian.PutUint16(result[0:], g.NonRepeaters)
	binary.LittleEndian.PutUint16(result[2:], g.MaxRepetitions)

	rangesBytes, err := g.SearchRanges.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return append(result, rangesBytes...), nil
}

// UnmarshalBinary sets the packet structure from the provided slice of bytes.
func (g *GetBulk) UnmarshalBinary(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("not enough bytes (%d) to unmarshal GetBulk packet header", len(data))
	}

	g.NonRepeaters = binary.LittleEndian.Uint16(data[0:])
	g.MaxRepetitions = binary.LittleEndian.Uint16(data[2:])

	if err := g.SearchRanges.UnmarshalBinary(data[4:]); err != nil {
		return err
	}

	return nil
}
