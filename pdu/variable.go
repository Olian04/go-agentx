// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package pdu

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/Olian04/go-agentx/value"
)

// Variable defines the pdu varbind packet.
type Variable struct {
	Type  VariableType
	Name  ObjectIdentifier
	Value interface{}
}

// Set sets the variable.
func (v *Variable) Set(oid value.OID, t VariableType, value interface{}) {
	v.Name.SetIdentifier(oid)
	v.Type = t
	v.Value = value
}

// ByteSize returns the number of bytes the variable would occupy when marshaled.
func (v *Variable) ByteSize() int {
	size := 4 // varbind header: type + 3 reserved bytes
	size += v.Name.ByteSize()

	switch v.Type {
	case VariableTypeInteger:
		size += 4
	case VariableTypeOctetString:
		text := v.Value.(string)
		l := len(text)
		pad := (4 - (l % 4)) & 3
		size += 4 + l + pad
	case VariableTypeNull, VariableTypeNoSuchObject, VariableTypeNoSuchInstance, VariableTypeEndOfMIBView:
		// no payload
	case VariableTypeObjectIdentifier:
		// v.Value is string during marshal path, value.OID during unmarshal path
		var oidVal value.OID
		switch vv := v.Value.(type) {
		case string:
			parsed, _ := value.ParseOID(vv)
			oidVal = parsed
		case value.OID:
			oidVal = vv
		default:
			oidVal = nil
		}
		oi := &ObjectIdentifier{}
		oi.SetIdentifier(oidVal)
		size += oi.ByteSize()
	case VariableTypeIPAddress:
		ip := v.Value.(net.IP)
		l := len(ip)
		pad := (4 - (l % 4)) & 3
		size += 4 + l + pad
	case VariableTypeCounter32, VariableTypeGauge32:
		size += 4
	case VariableTypeTimeTicks:
		size += 4
	case VariableTypeOpaque:
		data := v.Value.([]byte)
		l := len(data)
		pad := (4 - (l % 4)) & 3
		size += 4 + l + pad
	case VariableTypeCounter64:
		size += 8
	default:
		// unknown - treat as no payload
	}

	return size
}

// MarshalBinary returns the pdu packet as a slice of bytes.
func (v *Variable) MarshalBinary() ([]byte, error) {
	total := v.ByteSize()
	result := make([]byte, total)
	if _, err := v.MarshalTo(result); err != nil {
		return nil, err
	}
	return result, nil
}

// MarshalTo writes the variable into dst and returns bytes written.
// dst must have capacity >= v.ByteSize().
func (v *Variable) MarshalTo(dst []byte) (int, error) {
	offset := 0

	// VarBind header
	dst[offset] = byte(v.Type)
	dst[offset+1] = 0x00
	dst[offset+2] = 0x00
	dst[offset+3] = 0x00
	offset += 4

	// Name (inline ObjectIdentifier marshal)
	nameCount := len(v.Name.Subidentifiers)
	dst[offset] = byte(nameCount)
	dst[offset+1] = v.Name.Prefix
	dst[offset+2] = v.Name.Include
	// dst[offset+3] reserved
	offset += 4
	for i, sub := range v.Name.Subidentifiers {
		binary.LittleEndian.PutUint32(dst[offset+i*4:], sub)
	}
	offset += nameCount * 4

	// Value
	switch v.Type {
	case VariableTypeInteger:
		value := uint32(v.Value.(int32))
		binary.LittleEndian.PutUint32(dst[offset:], value)
		offset += 4
	case VariableTypeOctetString:
		text := v.Value.(string)
		l := len(text)
		pad := (4 - (l % 4)) & 3
		binary.LittleEndian.PutUint32(dst[offset:], uint32(l))
		offset += 4
		copy(dst[offset:], text)
		offset += l + pad
	case VariableTypeNull, VariableTypeNoSuchObject, VariableTypeNoSuchInstance, VariableTypeEndOfMIBView:
		// no payload
	case VariableTypeObjectIdentifier:
		// Accept string or value.OID
		var oidVal value.OID
		switch vv := v.Value.(type) {
		case string:
			parsed, err := value.ParseOID(vv)
			if err != nil {
				return 0, err
			}
			oidVal = parsed
		case value.OID:
			oidVal = vv
		default:
			return 0, fmt.Errorf("unexpected OID value type %T", v.Value)
		}
		// Apply AgentX prefix compression like ObjectIdentifier.SetIdentifier
		var subids []uint32
		prefix := byte(0)
		if len(oidVal) > 4 && oidVal[0] == 1 && oidVal[1] == 3 && oidVal[2] == 6 && oidVal[3] == 1 {
			prefix = byte(oidVal[4])
			subids = oidVal[5:]
		} else {
			subids = oidVal
		}
		count := len(subids)
		dst[offset] = byte(count)
		dst[offset+1] = prefix
		dst[offset+2] = 0 // Include defaults to false for value OID
		offset += 4
		for i, sub := range subids {
			binary.LittleEndian.PutUint32(dst[offset+i*4:], sub)
		}
		offset += count * 4
	case VariableTypeIPAddress:
		ip := []byte(v.Value.(net.IP))
		l := len(ip)
		pad := (4 - (l % 4)) & 3
		binary.LittleEndian.PutUint32(dst[offset:], uint32(l))
		offset += 4
		copy(dst[offset:], ip)
		offset += l + pad
	case VariableTypeCounter32, VariableTypeGauge32:
		value := v.Value.(uint32)
		binary.LittleEndian.PutUint32(dst[offset:], value)
		offset += 4
	case VariableTypeTimeTicks:
		value := uint32(v.Value.(time.Duration).Seconds() * 100)
		binary.LittleEndian.PutUint32(dst[offset:], value)
		offset += 4
	case VariableTypeOpaque:
		data := v.Value.([]byte)
		l := len(data)
		pad := (4 - (l % 4)) & 3
		binary.LittleEndian.PutUint32(dst[offset:], uint32(l))
		offset += 4
		copy(dst[offset:], data)
		offset += l + pad
	case VariableTypeCounter64:
		value := v.Value.(uint64)
		binary.LittleEndian.PutUint64(dst[offset:], value)
		offset += 8
	default:
		return 0, fmt.Errorf("unhandled variable type %s", v.Type)
	}

	return offset, nil
}

// UnmarshalBinary sets the packet structure from the provided slice of bytes.
func (v *Variable) UnmarshalBinary(data []byte) error {
	// Type + 3 reserved bytes
	v.Type = VariableType(data[0])
	offset := 4

	if err := v.Name.UnmarshalBinary(data[offset:]); err != nil {
		return err
	}
	offset += v.Name.ByteSize()

	switch v.Type {
	case VariableTypeInteger:
		v.Value = int32(binary.LittleEndian.Uint32(data[offset:]))
		offset += 4
	case VariableTypeOctetString:
		length := int(binary.LittleEndian.Uint32(data[offset:]))
		start := offset + 4
		end := start + length
		v.Value = string(data[start:end])
	case VariableTypeNull, VariableTypeNoSuchObject, VariableTypeNoSuchInstance, VariableTypeEndOfMIBView:
		v.Value = nil
	case VariableTypeObjectIdentifier:
		oid := &ObjectIdentifier{}
		if err := oid.UnmarshalBinary(data[offset:]); err != nil {
			return err
		}
		v.Value = oid.GetIdentifier()
	case VariableTypeIPAddress:
		length := int(binary.LittleEndian.Uint32(data[offset:]))
		start := offset + 4
		end := start + length
		b := make([]byte, length)
		copy(b, data[start:end])
		v.Value = net.IP(b)
	case VariableTypeCounter32, VariableTypeGauge32:
		v.Value = binary.LittleEndian.Uint32(data[offset:])
		offset += 4
	case VariableTypeTimeTicks:
		value := binary.LittleEndian.Uint32(data[offset:])
		offset += 4
		v.Value = time.Duration(value) * time.Second / 100
	case VariableTypeOpaque:
		length := int(binary.LittleEndian.Uint32(data[offset:]))
		start := offset + 4
		end := start + length
		b := make([]byte, length)
		copy(b, data[start:end])
		v.Value = b
	case VariableTypeCounter64:
		v.Value = binary.LittleEndian.Uint64(data[offset:])
		offset += 8
	default:
		return fmt.Errorf("unhandled variable type %s", v.Type)
	}

	return nil
}

func (v *Variable) String() string {
	return fmt.Sprintf("(variable %s = %v)", v.Type, v.Value)
}
