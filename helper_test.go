package agentx_test

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func SNMPGet(tb testing.TB, oid string) string {
	args := []string{"-v2c", "-cpublic", "-On", "127.0.0.1:30161"}
	// Allow multiple OIDs separated by spaces
	args = append(args, strings.Fields(oid)...)
	cmd := exec.Command("snmpget", args...)
	output, err := cmd.CombinedOutput()
	require.NoError(tb, err)
	return strings.TrimSpace(string(output))
}

func SNMPGetNext(tb testing.TB, oid string) string {
	cmd := exec.Command("snmpgetnext", "-v2c", "-cpublic", "-On", "127.0.0.1:30161", oid)
	output, err := cmd.CombinedOutput()
	require.NoError(tb, err)
	return strings.TrimSpace(string(output))
}

func SNMPGetBulk(tb testing.TB, oid string, nonRepeaters, maxRepetitions int) string {
	// Run once per OID to ensure each provided OID yields output
	var outputs []string
	for _, single := range strings.Fields(oid) {
		args := []string{"-v2c", "-cpublic", "-On", fmt.Sprintf("-Cn%d", nonRepeaters), fmt.Sprintf("-Cr%d", maxRepetitions), "127.0.0.1:30161", single}
		cmd := exec.Command("snmpbulkget", args...)
		output, err := cmd.CombinedOutput()
		require.NoError(tb, err)
		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			outputs = append(outputs, trimmed)
		}
	}
	return strings.Join(outputs, "\n")
}

func SNMPBulkWalk(tb testing.TB, oid string, maxRepetitions int) string {
	// Run once per OID to ensure each provided OID yields output
	var outputs []string
	for _, single := range strings.Fields(oid) {
		args := []string{"-v2c", "-cpublic", "-On", fmt.Sprintf("-Cr%d", maxRepetitions), "127.0.0.1:30161", single}
		cmd := exec.Command("snmpbulkwalk", args...)
		output, err := cmd.CombinedOutput()
		require.NoError(tb, err)
		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			outputs = append(outputs, trimmed)
		}
	}
	return strings.Join(outputs, "\n")
}

func SNMPWalk(tb testing.TB, oid string) string {
	// Run once per OID to ensure each provided OID yields output
	var outputs []string
	for _, single := range strings.Fields(oid) {
		args := []string{"-v2c", "-cpublic", "-On", "127.0.0.1:30161", single}
		cmd := exec.Command("snmpwalk", args...)
		output, err := cmd.CombinedOutput()
		require.NoError(tb, err)
		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			outputs = append(outputs, trimmed)
		}
	}
	return strings.Join(outputs, "\n")
}
