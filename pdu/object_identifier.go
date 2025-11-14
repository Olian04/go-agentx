// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package pdu

import (
	"encoding/binary"

	"github.com/Olian04/go-agentx/value"
)

const (
	INCLUDE_TRUE  = 0x01
	INCLUDE_FALSE = 0x00
)

// ObjectIdentifier defines the pdu object identifier packet.
type ObjectIdentifier struct {
	Prefix         uint8
	Include        byte
	Subidentifiers []uint32
}

// SetInclude sets the include field.
func (o *ObjectIdentifier) SetInclude(value bool) {
	if value {
		o.Include = INCLUDE_TRUE
	} else {
		o.Include = INCLUDE_FALSE
	}
}

// GetInclude returns true if the include field ist set, false otherwise.
func (o *ObjectIdentifier) GetInclude() bool {
	return o.Include == INCLUDE_TRUE
}

// SetIdentifier set the subidentifiers by the provided oid string.
func (o *ObjectIdentifier) SetIdentifier(oid value.OID) {
	// AgentX OID wire format uses a 'Prefix' to compress 1.3.6.1.<prefix>
	// The Subidentifiers must NOT include the 1.3.6.1.<prefix> part when Prefix is set.
	o.Prefix = 0
	if len(oid) > 4 && oid[0] == 1 && oid[1] == 3 && oid[2] == 6 && oid[3] == 1 {
		// Set Prefix to the 5th arc and strip the first 5 arcs from subids
		o.Prefix = uint8(oid[4])
		oid = oid[5:]
	}
	o.Subidentifiers = make([]uint32, len(oid))
	copy(o.Subidentifiers, oid)
}

// GetIdentifier returns the identifier as an oid string.
func (o *ObjectIdentifier) GetIdentifier() value.OID {
	var oid value.OID
	if o.Prefix != 0 {
		oid = append(oid, 1, 3, 6, 1, uint32(o.Prefix))
	}
	return append(oid, o.Subidentifiers...)
}

// ByteSize returns the number of bytes, the binding would need in the encoded version.
func (o *ObjectIdentifier) ByteSize() int {
	return 4 + len(o.Subidentifiers)*4
}

// MarshalBinary returns the pdu packet as a slice of bytes.
func (o *ObjectIdentifier) MarshalBinary() ([]byte, error) {
	count := len(o.Subidentifiers)
	result := make([]byte, 4+count*4)
	result[0] = byte(count)
	result[1] = o.Prefix
	result[2] = o.Include
	// result[3] reserved (0x00)
	for i, sub := range o.Subidentifiers {
		binary.LittleEndian.PutUint32(result[4+i*4:], sub)
	}
	return result, nil
}

// UnmarshalBinary sets the packet structure from the provided slice of bytes.
func (o *ObjectIdentifier) UnmarshalBinary(data []byte) error {
	count := int(data[0])
	o.Prefix = data[1]
	o.Include = data[2]

	o.Subidentifiers = make([]uint32, count)
	base := 4
	for i := 0; i < count; i++ {
		o.Subidentifiers[i] = binary.LittleEndian.Uint32(data[base+i*4:])
	}

	return nil
}

func (o ObjectIdentifier) String() string {
	return o.GetIdentifier().String()
}
