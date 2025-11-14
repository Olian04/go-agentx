// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package marshaler

import (
	"encoding"
)

// Multi defines a binary marshaler that marshals all child marshalers
// and concatinate the results.
type Multi []encoding.BinaryMarshaler

// NewMulti returns a new instance of MultiBinaryMarshaler.
func NewMulti(marshalers ...encoding.BinaryMarshaler) Multi {
	return Multi(marshalers)
}

// MarshalBinary marshals all the binary marshalers and concatinates the results.
func (m Multi) MarshalBinary() ([]byte, error) {
	// First pass: marshal each part and compute total length
	type part struct {
		data []byte
	}
	parts := make([]part, 0, len(m))
	total := 0
	for _, marshaler := range m {
		data, err := marshaler.MarshalBinary()
		if err != nil {
			return nil, err
		}
		parts = append(parts, part{data: data})
		total += len(data)
	}
	// Second pass: copy into a single buffer
	result := make([]byte, total)
	offset := 0
	for _, p := range parts {
		copy(result[offset:], p.data)
		offset += len(p.data)
	}
	return result, nil
}
