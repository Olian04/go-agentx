// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package value

import (
	"sort"
	"strconv"
	"strings"
)

// OID defines an OID.
type OID []uint32

// ParseOID parses the provided string and returns a valid oid. If one of the
// subidentifiers cannot be parsed to an uint32, the function will return an error.
func ParseOID(text string) (OID, error) {
	result := make(OID, 0, 8) // default small capacity
	var current uint32
	haveDigit := false
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if ch >= '0' && ch <= '9' {
			haveDigit = true
			current = current*10 + uint32(ch-'0')
			continue
		}
		if ch == '.' {
			if !haveDigit {
				// invalid empty component, fall back to strconv for error consistency
				parts := strings.Split(text, ".")
				result = result[:0]
				for _, part := range parts {
					val, err := strconv.ParseUint(part, 10, 32)
					if err != nil {
						return nil, err
					}
					result = append(result, uint32(val))
				}
				return result, nil
			}
			result = append(result, current)
			current = 0
			haveDigit = false
			continue
		}
		// invalid char, delegate to slower parser for error
		parts := strings.Split(text, ".")
		result = result[:0]
		for _, part := range parts {
			val, err := strconv.ParseUint(part, 10, 32)
			if err != nil {
				return nil, err
			}
			result = append(result, uint32(val))
		}
		return result, nil
	}
	if haveDigit {
		result = append(result, current)
	}
	return result, nil
}

// MustParseOID works like ParseOID expect it panics on a parsing error.
func MustParseOID(text string) OID {
	result, err := ParseOID(text)
	if err != nil {
		panic(err)
	}
	return result
}

// First returns the first n subidentifiers as a new oid.
func (o OID) First(count int) OID {
	return o[:count]
}

// CommonPrefix compares the oid with the provided one and
// returns a new oid containing all matching prefix subidentifiers.
func (o OID) CommonPrefix(other OID) OID {
	matchCount := 0

	for index, subidentifier := range o {
		if index >= len(other) || subidentifier != other[index] {
			break
		}
		matchCount++
	}

	return o[:matchCount]
}

// CompareOIDs returns an integer comparing two OIDs lexicographically.
// The result will be 0 if oid1 == oid2, -1 if oid1 < oid2, +1 if oid1 > oid2.
func CompareOIDs(oid1, oid2 OID) int {
	if oid2 != nil {
		oid1Length := len(oid1)
		oid2Length := len(oid2)
		for i := 0; i < oid1Length && i < oid2Length; i++ {
			if oid1[i] < oid2[i] {
				return -1
			}
			if oid1[i] > oid2[i] {
				return 1
			}
		}
		if oid1Length == oid2Length {
			return 0
		} else if oid1Length < oid2Length {
			return -1
		} else {
			return 1
		}
	}
	return 1
}

// SortOIDs performs sorting of the OID list.
func SortOIDs(oids []OID) {
	sort.Slice(oids, func(i, j int) bool {
		return CompareOIDs(oids[i], oids[j]) == -1
	})
}

func (o OID) String() string {
	if len(o) == 0 {
		return ""
	}
	var b strings.Builder
	// Estimate: up to 10 digits per subid + dots
	b.Grow(len(o)*11 - 1)
	// first element without leading dot
	b.WriteString(strconv.FormatUint(uint64(o[0]), 10))
	for i := 1; i < len(o); i++ {
		b.WriteByte('.')
		b.WriteString(strconv.FormatUint(uint64(o[i]), 10))
	}
	return b.String()
}

// LowerBound returns the first index i in oids such that:
// - oids[i] >= target if include == true
// - oids[i] >  target if include == false
// If no such index exists, returns len(oids).
func LowerBound(oids []OID, target OID, include bool) int {
	lo, hi := 0, len(oids)
	for lo < hi {
		mid := (lo + hi) >> 1
		cmp := CompareOIDs(oids[mid], target)
		if cmp < 0 || (!include && cmp == 0) {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// InsertSorted inserts oid into a sorted slice and returns the new slice.
func InsertSorted(oids []OID, oid OID) []OID {
	i := LowerBound(oids, oid, true)
	if i == len(oids) {
		return append(oids, oid)
	}
	oids = append(oids, nil)   // grow by one
	copy(oids[i+1:], oids[i:]) // shift right
	oids[i] = oid              // place
	return oids
}
