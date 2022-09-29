/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"errors"
	"net"
	"runtime"
	"strconv"
	"time"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/face/impl"
	"github.com/named-data/YaNFD/ndn"
	"github.com/named-data/YaNFD/ndn/tlv"
)

// UnicastTCPTransport is a unicast TCP transport.
type UnicastTCPTransport struct {
	dialer     *net.Dialer
	conn       *net.TCPConn
	localAddr  net.TCPAddr
	remoteAddr net.TCPAddr
	transportBase
}

// MakeUnicastTCPTransport creates a new unicast TCP transport.
func MakeUnicastTCPTransport(remoteURI *ndn.URI, localURI *ndn.URI, persistency Persistency) (*UnicastTCPTransport, error) {
	// Validate URIs.
	if !remoteURI.IsCanonical() ||
		(remoteURI.Scheme() != "tcp4" && remoteURI.Scheme() != "tcp6") ||
		(localURI != nil && !localURI.IsCanonical()) ||
		(localURI != nil && remoteURI.Scheme() != localURI.Scheme()) {
		return nil, core.ErrNotCanonical
	}

	t := new(UnicastTCPTransport)
	// All persistencies are accepted.
	t.makeTransportBase(remoteURI, localURI, persistency, ndn.NonLocal, ndn.PointToPoint, tlv.MaxNDNPacketSize)
	t.expirationTime = new(time.Time)
	*t.expirationTime = time.Now().Add(tcpLifetime)

	// Set scope
	ip := net.ParseIP(remoteURI.Path())
	if ip.IsLoopback() {
		t.scope = ndn.Local
	} else {
		t.scope = ndn.NonLocal
	}

	// Set local and remote addresses
	if localURI != nil {
		t.localAddr.IP = net.ParseIP(localURI.Path())
		t.localAddr.Port = int(localURI.Port())
	} else {
		t.localAddr.Port = int(TCPUnicastPort)
	}
	t.remoteAddr.IP = net.ParseIP(remoteURI.Path())
	t.remoteAddr.Port = int(remoteURI.Port())

	// Attempt to "dial" remote URI
	var err error
	// Configure dialer so we can allow address reuse
	t.dialer = &net.Dialer{LocalAddr: &t.localAddr, Control: impl.SyscallReuseAddr}
	conn, err := t.dialer.Dial(t.remoteURI.Scheme(), net.JoinHostPort(t.remoteURI.Path(), strconv.Itoa(int(t.remoteURI.Port()))))
	if err != nil {
		return nil, errors.New("Unable to connect to remote endpoint: " + err.Error())
	}
	t.conn = conn.(*net.TCPConn)

	if localURI == nil {
		t.localAddr = *t.conn.LocalAddr().(*net.TCPAddr)
		t.localURI = ndn.DecodeURIString("tcp://" + t.localAddr.String())
	}

	t.changeState(ndn.Up)

	go t.expirationHandler()

	return t, nil
}

func AcceptUnicastTCPTransport(remoteConn net.Conn, localURI *ndn.URI, persistency Persistency) (*UnicastTCPTransport, error) {
	// Construct remote URI
	var remoteURI *ndn.URI
	remoteAddr := remoteConn.RemoteAddr()
	host, port, err := net.SplitHostPort(remoteAddr.String())
	if err != nil {
		core.LogWarn("UnicastTCPTransport", "Unable to create face from ", remoteAddr, ": could not split host from port")
		return nil, err
	}
	portInt, _ := strconv.ParseUint(port, 10, 16)
	if err != nil {
		core.LogWarn("UnicastTCPTransport", "Unable to create face from ", remoteAddr, ": could not split host from port")
		return nil, err
	}
	remoteURI = ndn.MakeTCPFaceURI(4, host, uint16(portInt))
	remoteURI.Canonize()
	if !remoteURI.IsCanonical() {
		core.LogWarn("UnicastTCPTransport", "Unable to create face from ", remoteURI, ": remote URI is not canonical")
		return nil, err
	}

	t := new(UnicastTCPTransport)
	// All persistencies are accepted.
	t.makeTransportBase(remoteURI, localURI, persistency, ndn.NonLocal, ndn.PointToPoint, tlv.MaxNDNPacketSize)
	t.expirationTime = new(time.Time)
	*t.expirationTime = time.Now().Add(tcpLifetime)

	// Set scope
	ip := net.ParseIP(remoteURI.Path())
	if ip.IsLoopback() {
		t.scope = ndn.Local
	} else {
		t.scope = ndn.NonLocal
	}

	// Set local and remote addresses
	if localURI != nil {
		t.localAddr.IP = net.ParseIP(localURI.Path())
		t.localAddr.Port = int(localURI.Port())
	} else {
		t.localAddr.Port = int(TCPUnicastPort)
	}
	t.remoteAddr.IP = net.ParseIP(remoteURI.Path())
	t.remoteAddr.Port = int(remoteURI.Port())

	var success bool
	t.conn, success = remoteConn.(*net.TCPConn)
	if !success {
		core.LogError("UnicastTCPTransport", "Specified connection ", remoteConn, " is not a net.TCPConn")
		return nil, errors.New("Specified connection is not a net.TCPConn")
	}

	if localURI == nil {
		t.localAddr = *t.conn.LocalAddr().(*net.TCPAddr)
		t.localURI = ndn.DecodeURIString("tcp://" + t.localAddr.String())
	}

	t.changeState(ndn.Up)

	go t.expirationHandler()

	return t, nil
}

func (t *UnicastTCPTransport) String() string {
	return "UnicastTCPTransport, FaceID=" + strconv.FormatUint(t.faceID, 10) + ", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String()
}

// SetPersistency changes the persistency of the face.
func (t *UnicastTCPTransport) SetPersistency(persistency Persistency) bool {
	if persistency == t.persistency {
		return true
	}

	t.persistency = persistency
	return true
}

// GetSendQueueSize returns the current size of the send queue.
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
	default:
		t.changeState(ndn.Down)
	}
}

// expirationHandler checks if the face should expire (if on demand)
func (t *UnicastTCPTransport) expirationHandler() {
	for {
		time.Sleep(time.Duration(10) * time.Second)
		if t.state == ndn.Down {
			break
		}
		if t.persistency == PersistencyOnDemand && (t.expirationTime.Before(time.Now()) || t.expirationTime.Equal(time.Now())) {
			core.LogInfo(t, "Face expired")
			t.changeState(ndn.Down)
			break
		}
	}
}

func (t *UnicastTCPTransport) sendFrame(frame []byte) {
	if len(frame) > t.MTU() {
		core.LogWarn(t, "Attempted to send frame larger than MTU - DROP")
		return
	}

	core.LogDebug(t, "Sending frame of size ", len(frame))
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
	if lockThreadsToCores {
		runtime.LockOSThread()
	}

	recvBuf := make([]byte, tlv.MaxNDNPacketSize)
	for {
		readSize, err := t.conn.Read(recvBuf)
		if err != nil {
			if err.Error() == "EOF" {
				core.LogDebug(t, "EOF - Face DOWN")
			} else {
				core.LogWarn(t, "Unable to read from socket (", err, ") - DROP")
				t.onTransportFailure(true)
			}
			t.changeState(ndn.Down)
			break
		}

		core.LogTrace(t, "Receive of size ", readSize)
		t.nInBytes += uint64(readSize)
		*t.expirationTime = time.Now().Add(tcpLifetime)

		if readSize > tlv.MaxNDNPacketSize {
			core.LogWarn(t, "Received too much data without valid TLV block - DROP")
			continue
		}

		// Send up to link service
		t.linkService.handleIncomingFrame(recvBuf[:readSize])
	}
}

func (t *UnicastTCPTransport) changeState(new ndn.State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "state: ", t.state, " -> ", new)
	t.state = new

	if t.state != ndn.Up {
		core.LogInfo(t, "Closing TCP socket")
		t.hasQuit <- true
		t.conn.Close()

		// Stop link service
		t.linkService.tellTransportQuit()

		FaceTable.Remove(t.faceID)
	}
}
