// Copyright 2026 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux

package net

import (
	"context"
	stdnet "net"
	"testing"

	libnet "github.com/fatedier/golib/net"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func getSockoptInt(t *testing.T, c stdnet.Conn, level, opt int) int {
	t.Helper()
	raw, err := c.(*stdnet.TCPConn).SyscallConn()
	require.NoError(t, err)
	var (
		value   int
		sockErr error
	)
	require.NoError(t, raw.Control(func(fd uintptr) {
		value, sockErr = unix.GetsockoptInt(int(fd), level, opt)
	}))
	require.NoError(t, sockErr)
	return value
}

func TestDialHookTCPSendQueueSetsSocketOptions(t *testing.T) {
	ln, err := stdnet.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	const (
		notSentLowat = 16 * 1024
		// Below the default net.core.wmem_max, so the request is not clamped.
		sendBuffer = 64 * 1024
	)
	conn, err := libnet.DialContext(context.Background(), ln.Addr().String(),
		libnet.WithAfterHook(libnet.AfterHook{Priority: 1, Hook: DialHookTCPSendQueue(notSentLowat, sendBuffer)}))
	require.NoError(t, err)
	defer conn.Close()

	require.Equal(t, notSentLowat, getSockoptInt(t, conn, unix.IPPROTO_TCP, unix.TCP_NOTSENT_LOWAT))
	// The kernel doubles the requested value for its bookkeeping overhead.
	require.Equal(t, 2*sendBuffer, getSockoptInt(t, conn, unix.SOL_SOCKET, unix.SO_SNDBUF))
}

func TestDialHookTCPSendQueueZeroLeavesDefaults(t *testing.T) {
	ln, err := stdnet.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	plain, err := stdnet.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	defer plain.Close()
	hooked, err := libnet.DialContext(context.Background(), ln.Addr().String(),
		libnet.WithAfterHook(libnet.AfterHook{Priority: 1, Hook: DialHookTCPSendQueue(0, 0)}))
	require.NoError(t, err)
	defer hooked.Close()

	require.Equal(t,
		getSockoptInt(t, plain, unix.IPPROTO_TCP, unix.TCP_NOTSENT_LOWAT),
		getSockoptInt(t, hooked, unix.IPPROTO_TCP, unix.TCP_NOTSENT_LOWAT))
}
