// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package pdu

// Get defines the pdu get packet.
type Get struct {
	SearchRanges Ranges
}

// Type returns the pdu packet type.
func (g *Get) Type() Type {
	return TypeGet
}

// MarshalBinary returns the pdu packet as a slice of bytes.
func (g *Get) MarshalBinary() ([]byte, error) {
	// Subagents do not send Get PDUs; master sends them. Keep empty.
	return []byte{}, nil
}

// UnmarshalBinary sets the packet structure from the provided slice of bytes.
func (g *Get) UnmarshalBinary(data []byte) error {
	// A Get request may contain a list of search ranges (one per varbind)
	return g.SearchRanges.UnmarshalBinary(data)
}
