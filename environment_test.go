// Copyright 2018 The agentx authors
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package agentx_test

import (
	"log/slog"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Olian04/go-agentx"
)

type environment struct {
	client *agentx.Client
}

func setUpTestEnvironment(tb testing.TB) *environment {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	})).With("test", tb.Name()))

	cmd := exec.Command("snmpd", "-Ln", "-f", "-C", "-c", "snmpd.conf")

	_, err := cmd.StdoutPipe()
	require.NoError(tb, err)

	slog.Info("running command", slog.String("command", cmd.String()))
	require.NoError(tb, cmd.Start())
	time.Sleep(500 * time.Millisecond)

	client, err := agentx.Dial("tcp", "127.0.0.1:30705",
		agentx.WithLogger(slog.Default()),
		agentx.WithTimeout(60*time.Second),
		agentx.WithReconnectInterval(1*time.Second),
	)
	require.NoError(tb, err)

	tb.Cleanup(func() {
		require.NoError(tb, client.Close())
		require.NoError(tb, cmd.Process.Kill())
	})

	return &environment{client: client}
}
