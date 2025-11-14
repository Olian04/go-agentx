// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package pdu

// Ranges defines the pdu search range list packet.
type Ranges []Range

// MarshalBinary returns the pdu packet as a slice of bytes.
func (r *Ranges) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}

// UnmarshalBinary sets the packet structure from the provided slice of bytes.
func (r *Ranges) UnmarshalBinary(data []byte) error {
	// Pre-size allocation by scanning encoded sizes once
	count := 0
	for offset := 0; offset < len(data); {
		size, ok := encodedRangeSize(data[offset:])
		if !ok {
			break
		}
		offset += size
		count++
	}
	*r = make([]Range, 0, count)
	for offset := 0; offset < len(data); {
		rng := Range{}
		if err := rng.UnmarshalBinary(data[offset:]); err != nil {
			return err
		}
		*r = append(*r, rng)
		offset += rng.ByteSize()
	}
	return nil
}

// encodedOIDSize returns the encoded size of an ObjectIdentifier starting at data.
// Format: 1 byte count, 1 byte prefix, 1 byte include, 1 byte reserved, then count*4 bytes subids.
func encodedOIDSize(data []byte) (int, bool) {
	if len(data) < 4 {
		return 0, false
	}
	count := int(data[0])
	size := 4 + count*4
	if len(data) < size {
		return 0, false
	}
	return size, true
}

// encodedRangeSize returns the encoded size of a Range (From OID + To OID).
func encodedRangeSize(data []byte) (int, bool) {
	fromSize, ok := encodedOIDSize(data)
	if !ok {
		return 0, false
	}
	toSize, ok := encodedOIDSize(data[fromSize:])
	if !ok {
		return 0, false
	}
	return fromSize + toSize, true
}
