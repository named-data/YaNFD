/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/ndn"
)

// WebSocketListenerConfig contains WebSocketListener configuration.
type WebSocketListenerConfig struct {
	Bind       string
	Port       uint16
	TLSEnabled bool
	TLSCert    string
	TLSKey     string
}

// URL returns server URL.
func (cfg WebSocketListenerConfig) URL() *url.URL {
	addr := net.JoinHostPort(cfg.Bind, strconv.FormatUint(uint64(cfg.Port), 10))
	u := &url.URL{
		Scheme: "ws",
		Host:   addr,
	}
	if cfg.TLSEnabled {
		u.Scheme = "wss"
	}
	return u
}

func (cfg WebSocketListenerConfig) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "WebSocket listener at %s", cfg.URL())
	if cfg.TLSEnabled {
		fmt.Fprintf(&b, " with TLS cert %s and key %s", cfg.TLSCert, cfg.TLSKey)
	}
	return b.String()
}

// NewWebSocketListener constructs a WebSocketListener.
func NewWebSocketListener(cfg WebSocketListenerConfig) (*WebSocketListener, error) {
	localURI := cfg.URL()
	server := http.Server{Addr: localURI.Host}
	if cfg.TLSEnabled {
		cert, e := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
		if e != nil {
			return nil, fmt.Errorf("tls.LoadX509KeyPair(%s %s): %w", cfg.TLSCert, cfg.TLSKey, e)
		}
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		localURI.Scheme = "wss"
	}
	return &WebSocketListener{
		server: server,
		upgrader: websocket.Upgrader{
			WriteBufferPool: &sync.Pool{},
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		localURI: ndn.MakeWebSocketServerFaceURI(localURI),
	}, nil
}

// WebSocketListener listens for incoming WebSockets connections.
type WebSocketListener struct {
	server   http.Server
	upgrader websocket.Upgrader
	localURI *ndn.URI
}

var _ Listener = &WebSocketListener{}

func (l *WebSocketListener) String() string {
	return "WebSocketListener, " + l.localURI.String()
}

// Run starts the WebSocket listener.
func (l *WebSocketListener) Run() {
	l.server.Handler = http.HandlerFunc(l.handler)

	var e error
	if l.server.TLSConfig == nil {
		e = l.server.ListenAndServe()
	} else {
		e = l.server.ListenAndServeTLS("", "")
	}
	if !errors.Is(e, http.ErrServerClosed) {
		core.LogFatal(l, "Unable to start listener: ", e)
	}
}

func (l *WebSocketListener) handler(w http.ResponseWriter, r *http.Request) {
	c, e := l.upgrader.Upgrade(w, r, nil)
	if e != nil {
		return
	}

	t := NewWebSocketTransport(l.localURI, c)
	linkService := MakeNDNLPLinkService(t, MakeNDNLPLinkServiceOptions())

	core.LogInfo(l, "Accepting new WebSocket face ", t.RemoteURI())
	FaceTable.Add(linkService)
	go linkService.Run(nil)
}

// Close closes the WebSocketListener.
func (l *WebSocketListener) Close() {
	core.LogInfo(l, "Stopping listener")
	l.server.Shutdown(context.TODO())
}
