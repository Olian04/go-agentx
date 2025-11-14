// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package agentx

import (
	"context"

	"github.com/Olian04/go-agentx/pdu"
	"github.com/Olian04/go-agentx/value"
)

// ListHandler is a helper that takes a list of oids and implements
// a default behaviour for that list.
type ListHandler struct {
	oids  []value.OID
	items map[string]*ListItem
	// itemsByIndex mirrors oids by index for fast lookup without string conversions
	itemsByIndex []*ListItem
}

// Add adds a list item for the provided oid and returns it.
func (l *ListHandler) Add(oid string) *ListItem {
	if l.items == nil {
		l.items = make(map[string]*ListItem)
	}

	parsedOID := value.MustParseOID(oid)
	item := &ListItem{}
	// Insert into sorted oids and parallel itemsByIndex slice
	idx := value.LowerBound(l.oids, parsedOID, true)
	if idx == len(l.oids) {
		l.oids = append(l.oids, parsedOID)
		l.itemsByIndex = append(l.itemsByIndex, item)
	} else {
		l.oids = append(l.oids, nil)
		copy(l.oids[idx+1:], l.oids[idx:])
		l.oids[idx] = parsedOID
		l.itemsByIndex = append(l.itemsByIndex, nil)
		copy(l.itemsByIndex[idx+1:], l.itemsByIndex[idx:])
		l.itemsByIndex[idx] = item
	}
	l.items[oid] = item
	return item
}

// Get tries to find the provided oid and returns the corresponding value.
func (l *ListHandler) Get(ctx context.Context, oid value.OID) (value.OID, pdu.VariableType, any, error) {
	if l.items == nil {
		return nil, pdu.VariableTypeNoSuchObject, nil, nil
	}

	// Binary search to avoid oid.String allocations and map lookup
	idx := value.LowerBound(l.oids, oid, true)
	if idx < len(l.oids) && value.CompareOIDs(l.oids[idx], oid) == 0 {
		item := l.itemsByIndex[idx]
		return oid, item.Type, item.Value, nil
	}
	return nil, pdu.VariableTypeNoSuchObject, nil, nil
}

// GetNext tries to find the value that follows the provided oid and returns it.
func (l *ListHandler) GetNext(ctx context.Context, from value.OID, includeFrom bool, to value.OID) (value.OID, pdu.VariableType, any, error) {
	if l.items == nil {
		return nil, pdu.VariableTypeNoSuchObject, nil, nil
	}

	idx := value.LowerBound(l.oids, from, includeFrom)
	if idx < len(l.oids) && value.CompareOIDs(l.oids[idx], to) == -1 {
		item := l.itemsByIndex[idx]
		return l.oids[idx], item.Type, item.Value, nil
	}

	return nil, pdu.VariableTypeNoSuchObject, nil, nil
}
