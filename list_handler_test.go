// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package agentx_test

import (
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
		assert.Equal(t,
			".1.3.6.1.4.1.45995.3.1 = STRING: \"test\"",
			SNMPGetNext(t, "1.3.6.1.4.1.45995.3.0"))

		assert.Equal(t,
			".1.3.6.1.4.1.45995.3.1 = STRING: \"test\"",
			SNMPGetNext(t, "1.3.6.1.4.1.45995.3"))

	})

	t.Run("GetBulk", func(t *testing.T) {
		assert.Equal(t,
			".1.3.6.1.4.1.45995.3.1 = STRING: \"test\"",
			SNMPGetBulk(t, "1.3.6.1.4.1.45995.3.0", 0, 1))
	})
}
