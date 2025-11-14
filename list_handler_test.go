// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package agentx_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Olian04/go-agentx"
	"github.com/Olian04/go-agentx/pdu"
	"github.com/Olian04/go-agentx/value"
)

func TestListHandler(t *testing.T) {
	e := setUpTestEnvironment(t)

	lh := &agentx.ListHandler{}
	i1 := lh.Add("1.3.6.1.4.1.45995.3.1")
	i1.Type = pdu.VariableTypeOctetString
	i1.Value = "test"

	i2 := lh.Add("1.3.6.1.4.1.45995.3.3")
	i2.Type = pdu.VariableTypeOctetString
	i2.Value = "test2"

	// Additional OIDs for broader coverage
	i3 := lh.Add("1.3.6.1.4.1.45995.3.5")
	i3.Type = pdu.VariableTypeOctetString
	i3.Value = "test5"

	i4 := lh.Add("1.3.6.1.4.1.45995.3.7")
	i4.Type = pdu.VariableTypeOctetString
	i4.Value = "test7"

	session, err := e.client.Session(value.MustParseOID("1.3.6.1.4.1.45995"), "test client", lh)
	require.NoError(t, err)
	defer session.Close()

	baseOID := value.MustParseOID("1.3.6.1.4.1.45995")

	require.NoError(t, session.Register(127, baseOID))
	defer session.Unregister(127, baseOID)

	t.Run("Get (single OID)", func(t *testing.T) {
		assert.Equal(t,
			".1.3.6.1.4.1.45995.3.1 = STRING: \"test\"",
			SNMPGet(t, "1.3.6.1.4.1.45995.3.1"))

		assert.Equal(t,
			".1.3.6.1.4.1.45995.3.2 = No Such Object available on this agent at this OID",
			SNMPGet(t, "1.3.6.1.4.1.45995.3.2"))
	})

	t.Run("Get (multiple OIDs)", func(t *testing.T) {
		assert.Equal(t,
			".1.3.6.1.4.1.45995.3.1 = STRING: \"test\"\n.1.3.6.1.4.1.45995.3.3 = STRING: \"test2\"",
			SNMPGet(t, "1.3.6.1.4.1.45995.3.1 1.3.6.1.4.1.45995.3.3"))
	})

	t.Run("GetNext", func(t *testing.T) {
		t.Run("OK", func(t *testing.T) {
			out := SNMPGetNext(t, "1.3.6.1.4.1.45995.3.0")
			rows := strings.Split(out, "\n")
			assert.Len(t, rows, 1)
			assert.Equal(t, rows[0], ".1.3.6.1.4.1.45995.3.1 = STRING: \"test\"")
		})

		t.Run("No such object", func(t *testing.T) {
			out := SNMPGet(t, "1.3.6.1.4.1.45995.3.10")
			rows := strings.Split(out, "\n")
			assert.Len(t, rows, 1)
			assert.Equal(t, rows[0], ".1.3.6.1.4.1.45995.3.10 = No Such Object available on this agent at this OID")
		})
	})

	t.Run("GetBulk (snmpbulkget, single OID)", func(t *testing.T) {
		out := SNMPGetBulk(t, "1.3.6.1.4.1.45995.3.0", 0, 1)
		rows := strings.Split(out, "\n")
		assert.Len(t, rows, 1)
		assert.Equal(t, rows[0], ".1.3.6.1.4.1.45995.3.1 = STRING: \"test\"")
	})

	t.Run("GetBulk (snmpbulkget, multiple OIDs)", func(t *testing.T) {
		out := SNMPGetBulk(t, "1.3.6.1.4.1.45995.3.0 1.3.6.1.4.1.45995.3.1", 1, 1)
		rows := strings.Split(out, "\n")
		assert.Len(t, rows, 2)
		assert.Equal(t, rows[0], ".1.3.6.1.4.1.45995.3.1 = STRING: \"test\"")
		assert.Equal(t, rows[1], ".1.3.6.1.4.1.45995.3.3 = STRING: \"test2\"")
	})

	t.Run("GetBulk (snmpbulkwalk, single OID)", func(t *testing.T) {
		out := SNMPBulkWalk(t, "1.3.6.1.4.1.45995.3", 10)
		rows := strings.Split(out, "\n")
		assert.Len(t, rows, 4)
		assert.Equal(t, rows[0], ".1.3.6.1.4.1.45995.3.1 = STRING: \"test\"")
		assert.Equal(t, rows[1], ".1.3.6.1.4.1.45995.3.3 = STRING: \"test2\"")
		assert.Equal(t, rows[2], ".1.3.6.1.4.1.45995.3.5 = STRING: \"test5\"")
		assert.Equal(t, rows[3], ".1.3.6.1.4.1.45995.3.7 = STRING: \"test7\"")
	})

	t.Run("GetBulk (snmpbulkwalk, multiple OIDs)", func(t *testing.T) {
		out := SNMPBulkWalk(t, "1.3.6.1.4.1.45995.3.0 1.3.6.1.4.1.45995.3", 10)
		rows := strings.Split(out, "\n")
		assert.Len(t, rows, 5)
		assert.Equal(t, rows[0], ".1.3.6.1.4.1.45995.3.0 = No Such Object available on this agent at this OID")
		assert.Equal(t, rows[1], ".1.3.6.1.4.1.45995.3.1 = STRING: \"test\"")
		assert.Equal(t, rows[2], ".1.3.6.1.4.1.45995.3.3 = STRING: \"test2\"")
		assert.Equal(t, rows[3], ".1.3.6.1.4.1.45995.3.5 = STRING: \"test5\"")
		assert.Equal(t, rows[4], ".1.3.6.1.4.1.45995.3.7 = STRING: \"test7\"")
	})

	t.Run("Walk (snmpwalk, single OID)", func(t *testing.T) {
		out := SNMPWalk(t, "1.3.6.1.4.1.45995.3")
		rows := strings.Split(out, "\n")
		assert.Len(t, rows, 4)
		assert.Equal(t, rows[0], ".1.3.6.1.4.1.45995.3.1 = STRING: \"test\"")
		assert.Equal(t, rows[1], ".1.3.6.1.4.1.45995.3.3 = STRING: \"test2\"")
		assert.Equal(t, rows[2], ".1.3.6.1.4.1.45995.3.5 = STRING: \"test5\"")
		assert.Equal(t, rows[3], ".1.3.6.1.4.1.45995.3.7 = STRING: \"test7\"")
	})

	t.Run("Walk (snmpwalk, multiple OIDs)", func(t *testing.T) {
		out := SNMPWalk(t, "1.3.6.1.4.1.45995.3.0 1.3.6.1.4.1.45995.3")
		rows := strings.Split(out, "\n")
		assert.Len(t, rows, 5)
		assert.Equal(t, rows[0], ".1.3.6.1.4.1.45995.3.0 = No Such Object available on this agent at this OID")
		assert.Equal(t, rows[1], ".1.3.6.1.4.1.45995.3.1 = STRING: \"test\"")
		assert.Equal(t, rows[2], ".1.3.6.1.4.1.45995.3.3 = STRING: \"test2\"")
		assert.Equal(t, rows[3], ".1.3.6.1.4.1.45995.3.5 = STRING: \"test5\"")
		assert.Equal(t, rows[4], ".1.3.6.1.4.1.45995.3.7 = STRING: \"test7\"")
	})
}
