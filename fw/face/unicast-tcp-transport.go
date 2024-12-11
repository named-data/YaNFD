/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/named-data/YaNFD/core"
	defn "github.com/named-data/YaNFD/defn"
	"github.com/named-data/YaNFD/face/impl"
	"github.com/zjkmxy/go-ndn/pkg/utils"
)

// UnicastTCPTransport is a unicast TCP transport.
type UnicastTCPTransport struct {
	dialer     *net.Dialer
	conn       *net.TCPConn
	localAddr  net.TCPAddr
	remoteAddr net.TCPAddr
	transportBase
}

// Makes an outgoing unicast TCP transport.
func MakeUnicastTCPTransport(
	remoteURI *defn.URI,
	localURI *defn.URI,
	persistency Persistency,
) (*UnicastTCPTransport, error) {
	// Validate URIs.
	if !remoteURI.IsCanonical() ||
		(remoteURI.Scheme() != "tcp4" && remoteURI.Scheme() != "tcp6") {
		return nil, core.ErrNotCanonical
	}
	if localURI != nil {
		return nil, errors.New("do not specify localURI for TCP")
	}

	// Construct transport
	t := new(UnicastTCPTransport)
	t.makeTransportBase(remoteURI, localURI, persistency, defn.NonLocal, defn.PointToPoint, defn.MaxNDNPacketSize)
	t.expirationTime = utils.IdPtr(time.Now().Add(tcpLifetime))

	// Set scope
	ip := net.ParseIP(remoteURI.Path())
	if ip.IsLoopback() {
		t.scope = defn.Local
	} else {
		t.scope = defn.NonLocal
	}

	// Set local and remote addresses
	t.localAddr.Port = int(TCPUnicastPort)
	t.remoteAddr.IP = net.ParseIP(remoteURI.Path())
	t.remoteAddr.Port = int(remoteURI.Port())

	// Configure dialer so we can allow address reuse
	// Fix: for TCP we shouldn't specify the local address. Instead, we should obtain it from system.
	// Though it succeeds in Windows and MacOS, Linux does not allow this.
	t.dialer = &net.Dialer{Control: impl.SyscallReuseAddr}
	remote := net.JoinHostPort(t.remoteURI.Path(), strconv.Itoa(int(t.remoteURI.Port())))

	conn, err := t.dialer.Dial(t.remoteURI.Scheme(), remote)
	if err != nil {
		return nil, errors.New("Unable to connect to remote endpoint: " + err.Error())
	}

	t.conn = conn.(*net.TCPConn)
	t.running.Store(true)

	t.localAddr = *t.conn.LocalAddr().(*net.TCPAddr)
	t.localURI = defn.DecodeURIString("tcp://" + t.localAddr.String())

	return t, nil
}

// Accept an incoming unicast TCP transport.
func AcceptUnicastTCPTransport(
	remoteConn net.Conn,
	localURI *defn.URI,
	persistency Persistency,
) (*UnicastTCPTransport, error) {
	// Construct remote URI
	var remoteURI *defn.URI
	remoteAddr := remoteConn.RemoteAddr()
	host, port, err := net.SplitHostPort(remoteAddr.String())
	if err != nil {
		core.LogWarn("UnicastTCPTransport", "Unable to create face from ", remoteAddr, ": could not split host from port")
		return nil, err
	}
	portInt, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		core.LogWarn("UnicastTCPTransport", "Unable to create face from ", remoteAddr, ": could not split host from port")
		return nil, err
	}
	remoteURI = defn.MakeTCPFaceURI(4, host, uint16(portInt))
	remoteURI.Canonize()
	if !remoteURI.IsCanonical() {
		core.LogWarn("UnicastTCPTransport", "Unable to create face from ", remoteURI, ": remote URI is not canonical")
		return nil, err
	}

	// Construct transport
	t := new(UnicastTCPTransport)
	t.makeTransportBase(remoteURI, localURI, persistency, defn.NonLocal, defn.PointToPoint, defn.MaxNDNPacketSize)
	t.expirationTime = utils.IdPtr(time.Now().Add(tcpLifetime))

	var success bool
	t.conn, success = remoteConn.(*net.TCPConn)
	if !success {
		core.LogError("UnicastTCPTransport", "Specified connection ", remoteConn, " is not a net.TCPConn")
		return nil, errors.New("specified connection is not a net.TCPConn")
	}
	t.running.Store(true)

	// Set scope
	ip := net.ParseIP(remoteURI.Path())
	if ip.IsLoopback() {
		t.scope = defn.Local
	} else {
		t.scope = defn.NonLocal
	}

	// Set local and remote addresses
	if localURI != nil {
		t.localAddr.IP = net.ParseIP(localURI.Path())
		t.localAddr.Port = int(localURI.Port())
	} else {
		t.localAddr = *t.conn.LocalAddr().(*net.TCPAddr)
		t.localURI = defn.DecodeURIString(fmt.Sprintf("tcp://%s", &t.localAddr))
	}

	t.remoteAddr.IP = net.ParseIP(remoteURI.Path())
	t.remoteAddr.Port = int(remoteURI.Port())

	return t, nil
}

func (t *UnicastTCPTransport) String() string {
	return fmt.Sprintf("UnicastTCPTransport, FaceID=%d, RemoteURI=%s, LocalURI=%s", t.faceID, t.remoteURI, t.localURI)
}

func (t *UnicastTCPTransport) SetPersistency(persistency Persistency) bool {
	t.persistency = persistency
	return true
}

func (t *UnicastTCPTransport) GetSendQueueSize() uint64 {
	rawConn, err := t.conn.SyscallConn()
	if err != nil {
		core.LogWarn(t, "Unable to get raw connection to get socket length: ", err)
	}
	return impl.SyscallGetSocketSendQueueSize(rawConn)
}

// onTransportFailure modifies the state of the UnicastTCPTransport to indicate
// a failure in transmission.
func (t *UnicastTCPTransport) onTransportFailure(fromReceive bool) {
	// TODO: fully broke
	switch t.persistency {
	case PersistencyPermanent:
		// Restart socket
		t.conn.Close()
		var err error
		conn, err := t.dialer.Dial(t.remoteURI.Scheme(), net.JoinHostPort(t.remoteURI.Path(),
			strconv.Itoa(int(t.remoteURI.Port()))))
		if err != nil {
			core.LogError(t, "Unable to connect to remote endpoint: ", err)
		}
		t.conn = conn.(*net.TCPConn)

		if fromReceive {
			t.runReceive()
		} else {
			// Old receive thread will error out, so we need to replace it
			go t.runReceive()
		}
	}
}

func (t *UnicastTCPTransport) sendFrame(frame []byte) {
	if !t.running.Load() {
		return
	}

	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	_, err := t.conn.Write(frame)
	if err != nil {
		core.LogWarn(t, "Unable to send on socket - DROP")
		t.onTransportFailure(false)
		return
	}

	t.nOutBytes += uint64(len(frame))
	*t.expirationTime = time.Now().Add(tcpLifetime)
}

func (t *UnicastTCPTransport) runReceive() {
	defer t.Close()

	err := readStreamTransport(t.conn, func(b []byte) {
		t.nInBytes += uint64(len(b))
		t.linkService.handleIncomingFrame(b)
	})
	if err != nil {
		core.LogWarn(t, "Unable to read from socket (", err, ") - Face DOWN")
	}
}

func (t *UnicastTCPTransport) Close() {
	if t.running.Swap(false) {
		t.conn.Close()
	}
}
