// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package pdu

import (
	"encoding/binary"
	"strings"

	"github.com/Olian04/go-agentx/value"
)

// Variables defines a list of variable bindings.
type Variables []Variable

// Add adds the provided variable.
func (v *Variables) Add(oid value.OID, t VariableType, value interface{}) {
	variable := Variable{}
	variable.Set(oid, t, value)
	*v = append(*v, variable)
}

// MarshalBinary returns the pdu packet as a slice of bytes.
func (v *Variables) MarshalBinary() ([]byte, error) {
	// Precompute total size to allocate once
	total := 0
	for i := range *v {
		total += (*v)[i].ByteSize()
	}
	result := make([]byte, total)
	offset := 0
	for i := range *v {
		n, err := (*v)[i].MarshalTo(result[offset:])
		if err != nil {
			return nil, err
		}
		offset += n
	}
	return result, nil
}

// UnmarshalBinary sets the packet structure from the provided slice of bytes.
func (v *Variables) UnmarshalBinary(data []byte) error {
	// Pre-size allocation by scanning encoded sizes once
	count := 0
	for off := 0; off < len(data); {
		size, err := encodedVarSize(data[off:])
		if err != nil {
			return err
		}
		off += size
		count++
	}

	*v = make([]Variable, 0, count)
	for offset := 0; offset < len(data); {
		variable := Variable{}
		if err := variable.UnmarshalBinary(data[offset:]); err != nil {
			return err
		}
		*v = append(*v, variable)
		offset += variable.ByteSize()
	}
	return nil
}

func (v Variables) String() string {
	parts := make([]string, len(v))
	for index, va := range v {
		parts[index] = va.String()
	}
	return "[variables " + strings.Join(parts, ", ") + "]"
}

// encodedVarSize returns the number of bytes occupied by a single encoded Variable at data.
func encodedVarSize(data []byte) (int, error) {
	if len(data) < 8 {
		return 0, nil
	}
	// header
	offset := 4
	// name ObjectIdentifier
	if len(data) < offset+1 {
		return 0, nil
	}
	nameCount := int(data[offset])
	nameSize := 4 + nameCount*4
	offset += nameSize
	if len(data) < offset {
		return 0, nil
	}
	t := VariableType(data[0])
	switch t {
	case VariableTypeInteger, VariableTypeCounter32, VariableTypeGauge32, VariableTypeTimeTicks:
		return offset + 4, nil
	case VariableTypeCounter64:
		return offset + 8, nil
	case VariableTypeOctetString, VariableTypeIPAddress, VariableTypeOpaque:
		if len(data) < offset+4 {
			return 0, nil
		}
		l := int(binary.LittleEndian.Uint32(data[offset:]))
		pad := (4 - (l % 4)) & 3
		return offset + 4 + l + pad, nil
	case VariableTypeObjectIdentifier:
		if len(data) < offset+1 {
			return 0, nil
		}
		valCount := int(data[offset])
		return offset + 4 + valCount*4, nil
	case VariableTypeNull, VariableTypeNoSuchObject, VariableTypeNoSuchInstance, VariableTypeEndOfMIBView:
		return offset, nil
	default:
		return offset, nil
	}
}
