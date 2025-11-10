#!/usr/bin/env bash
set -euo pipefail

SNMP_CONF="${SNMP_CONF:-/etc/snmp/snmpd.conf}"

snmpd -f -Lo -C -c "${SNMP_CONF}" &
SNMPD_PID=$!

cleanup() {
	kill "${SNMPD_PID}" 2>/dev/null || true
	wait "${SNMPD_PID}" 2>/dev/null || true
}
trap cleanup EXIT

export AGENTX_USE_EXTERNAL_SNMPD=1

# Give snmpd a moment to become ready before running tests.
sleep 1

if [ "$#" -eq 0 ]; then
	set -- ./...
fi

go test "$@"
