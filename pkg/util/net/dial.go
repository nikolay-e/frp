package net

import (
	"context"
	"net"
	"net/url"

	libnet "github.com/fatedier/golib/net"
	"golang.org/x/net/websocket"

	"github.com/fatedier/frp/pkg/util/xlog"
)

// DialHookTCPSendQueue bounds what a TCP connection to the server may queue on
// the sending side. notSentLowat (TCP_NOTSENT_LOWAT, Linux only) caps the bytes
// written but not yet sent, so the writer blocks while the kernel already holds
// that much and whatever sits above the socket, not the kernel, decides what
// goes next. sendBuffer (SO_SNDBUF) is the room for bytes in flight; setting it
// stops the kernel from autotuning it, so a small net.ipv4.tcp_wmem maximum no
// longer caps the throughput. A value of 0 leaves that option alone. It must run
// before any hook that wraps the connection (TLS, websocket).
func DialHookTCPSendQueue(notSentLowat, sendBuffer int) libnet.AfterHookFunc {
	return func(ctx context.Context, c net.Conn, addr string) (context.Context, net.Conn, error) {
		tcpConn, ok := c.(*net.TCPConn)
		if !ok {
			return ctx, c, nil
		}
		xl := xlog.FromContextSafe(ctx)
		if sendBuffer > 0 {
			if err := tcpConn.SetWriteBuffer(sendBuffer); err != nil {
				xl.Warnf("set send buffer of the connection to server failed: %v", err)
			}
		}
		if notSentLowat > 0 {
			if err := setTCPNotSentLowat(tcpConn, notSentLowat); err != nil {
				xl.Warnf("set TCP_NOTSENT_LOWAT of the connection to server failed: %v", err)
			}
		}
		return ctx, c, nil
	}
}

func DialHookCustomTLSHeadByte(enableTLS bool, disableCustomTLSHeadByte bool) libnet.AfterHookFunc {
	return func(ctx context.Context, c net.Conn, addr string) (context.Context, net.Conn, error) {
		if enableTLS && !disableCustomTLSHeadByte {
			_, err := c.Write([]byte{byte(FRPTLSHeadByte)})
			if err != nil {
				return nil, nil, err
			}
		}
		return ctx, c, nil
	}
}

func DialHookWebsocket(protocol string, host string) libnet.AfterHookFunc {
	return func(ctx context.Context, c net.Conn, addr string) (context.Context, net.Conn, error) {
		if protocol != "wss" {
			protocol = "ws"
		}
		if host == "" {
			host = addr
		}
		addr = protocol + "://" + host + FrpWebsocketPath
		uri, err := url.Parse(addr)
		if err != nil {
			return nil, nil, err
		}

		origin := "http://" + uri.Host
		cfg, err := websocket.NewConfig(addr, origin)
		if err != nil {
			return nil, nil, err
		}

		conn, err := websocket.NewClient(cfg, c)
		if err != nil {
			return nil, nil, err
		}
		// The tunnel payload is a raw byte stream (yamux), not UTF-8 text.
		// Send it as binary frames; otherwise RFC 6455-compliant intermediaries
		// (e.g. API gateways/reverse proxies) UTF-8-validate the default text
		// frames and close the connection on invalid bytes.
		conn.PayloadType = websocket.BinaryFrame
		return ctx, conn, nil
	}
}
