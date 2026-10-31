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

package net

import (
	"context"
	stdnet "net"
	"testing"

	libnet "github.com/fatedier/golib/net"
	"github.com/stretchr/testify/require"
)

func TestDialHookTCPSendQueueIgnoresOtherConns(t *testing.T) {
	require := require.New(t)

	client, server := stdnet.Pipe()
	defer client.Close()
	defer server.Close()

	_, conn, err := DialHookTCPSendQueue(16*1024, 64*1024)(context.Background(), client, "")
	require.NoError(err)
	require.Same(client, conn)
}

func TestDialHookTCPSendQueueKeepsTCPConn(t *testing.T) {
	require := require.New(t)

	ln, err := stdnet.Listen("tcp", "127.0.0.1:0")
	require.NoError(err)
	defer ln.Close()

	conn, err := libnet.DialContext(context.Background(), ln.Addr().String(),
		libnet.WithAfterHook(libnet.AfterHook{Priority: 1, Hook: DialHookTCPSendQueue(0, 0)}))
	require.NoError(err)
	defer conn.Close()
	require.IsType(&stdnet.TCPConn{}, conn)
}
