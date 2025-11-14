// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package pdu


import (
	"encoding/binary"
	"time"
)

// Response defines the pdu response packet.
type Response struct {
	UpTime    time.Duration
	Error     Error
	Index     uint16
	Variables Variables
}

// Type returns the pdu packet type.
func (r *Response) Type() Type {
	return TypeResponse
}

// MarshalBinary returns the pdu packet as a slice of bytes.
func (r *Response) MarshalBinary() ([]byte, error) {
	// AgentX encodes sysUpTime in hundredths of a second (centiseconds)
	upTime := uint32(r.UpTime.Seconds() * 100)
	vBytes, err := r.Variables.MarshalBinary()
	if err != nil {
		return nil, err
	}
	result := make([]byte, 8+len(vBytes))
	binary.LittleEndian.PutUint32(result[0:], upTime)
	binary.LittleEndian.PutUint16(result[4:], uint16(r.Error))
	binary.LittleEndian.PutUint16(result[6:], r.Index)
	copy(result[8:], vBytes)
	return result, nil
}

// UnmarshalBinary sets the packet structure from the provided slice of bytes.
func (r *Response) UnmarshalBinary(data []byte) error {
	upTime := binary.LittleEndian.Uint32(data[0:])
	// Convert centiseconds to duration
	r.UpTime = time.Duration(upTime) * time.Second / 100
	r.Error = Error(binary.LittleEndian.Uint16(data[4:]))
	r.Index = binary.LittleEndian.Uint16(data[6:])
	if err := r.Variables.UnmarshalBinary(data[8:]); err != nil {
		return err
	}

	return nil
}

func (r *Response) String() string {
	return "(response " + r.Variables.String() + ")"
}
