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
	ndn_defn "github.com/named-data/YaNFD/ndn_defn"
)

// UnicastUDPTransport is a unicast UDP transport.
type UnicastUDPTransport struct {
	dialer     *net.Dialer
	conn       *net.UDPConn
	localAddr  net.UDPAddr
	remoteAddr net.UDPAddr
	transportBase
}

// MakeUnicastUDPTransport creates a new unicast UDP transport.
func MakeUnicastUDPTransport(
	remoteURI *ndn_defn.URI, localURI *ndn_defn.URI, persistency Persistency,
) (*UnicastUDPTransport, error) {
	// Validate URIs
	if !remoteURI.IsCanonical() || (remoteURI.Scheme() != "udp4" && remoteURI.Scheme() != "udp6") ||
		(localURI != nil && !localURI.IsCanonical()) || (localURI != nil && remoteURI.Scheme() != localURI.Scheme()) {
		return nil, core.ErrNotCanonical
	}

	t := new(UnicastUDPTransport)
	// All persistencies are accepted
	t.makeTransportBase(remoteURI, localURI, persistency, ndn_defn.NonLocal, ndn_defn.PointToPoint, ndn_defn.MaxNDNPacketSize)
	t.expirationTime = new(time.Time)
	*t.expirationTime = time.Now().Add(udpLifetime)

	// Set scope
	ip := net.ParseIP(remoteURI.Path())
	if ip.IsLoopback() {
		t.scope = ndn_defn.Local
	} else {
		t.scope = ndn_defn.NonLocal
	}

	// Set local and remote addresses
	if localURI != nil {
		t.localAddr.IP = net.ParseIP(localURI.Path())
		t.localAddr.Port = int(localURI.Port())
	} else {
		t.localAddr.Port = int(UDPUnicastPort)
	}
	t.remoteAddr.IP = net.ParseIP(remoteURI.Path())
	t.remoteAddr.Port = int(remoteURI.Port())

	// Attempt to "dial" remote URI
	var err error
	// Configure dialer so we can allow address reuse
	t.dialer = &net.Dialer{LocalAddr: &t.localAddr, Control: impl.SyscallReuseAddr}
	conn, err := t.dialer.Dial(t.remoteURI.Scheme(),
		net.JoinHostPort(t.remoteURI.Path(), strconv.Itoa(int(t.remoteURI.Port()))))
	if err != nil {
		return nil, errors.New("Unable to connect to remote endpoint: " + err.Error())
	}
	t.conn = conn.(*net.UDPConn)

	if localURI == nil {
		t.localAddr = *t.conn.LocalAddr().(*net.UDPAddr)
		t.localURI = ndn_defn.DecodeURIString("udp://" + t.localAddr.String())
	}

	t.changeState(ndn_defn.Up)

	go t.expirationHandler()

	return t, nil
}

func (t *UnicastUDPTransport) String() string {
	return "UnicastUDPTransport, FaceID=" + strconv.FormatUint(t.faceID, 10) +
		", RemoteURI=" + t.remoteURI.String() + ", LocalURI=" + t.localURI.String()
}

// SetPersistency changes the persistency of the face.
func (t *UnicastUDPTransport) SetPersistency(persistency Persistency) bool {
	if persistency == t.persistency {
		return true
	}

	t.persistency = persistency
	return true
}

// GetSendQueueSize returns the current size of the send queue.
func (t *UnicastUDPTransport) GetSendQueueSize() uint64 {
	rawConn, err := t.conn.SyscallConn()
	if err != nil {
		core.LogWarn(t, "Unable to get raw connection to get socket length: ", err)
	}
	return impl.SyscallGetSocketSendQueueSize(rawConn)
}

func (t *UnicastUDPTransport) onTransportFailure(fromReceive bool) {
	switch t.persistency {
	case PersistencyPermanent:
		// Restart socket
		t.conn.Close()
		var err error
		conn, err := t.dialer.Dial(t.remoteURI.Scheme(),
			net.JoinHostPort(t.remoteURI.Path(), strconv.Itoa(int(t.remoteURI.Port()))))
		if err != nil {
			core.LogError(t, "Unable to connect to remote endpoint: ", err)
		}
		t.conn = conn.(*net.UDPConn)

		if fromReceive {
			t.runReceive()
		} else {
			// Old receive thread will error out, so we need to replace it
			go t.runReceive()
		}
	default:
		t.changeState(ndn_defn.Down)
	}
}

// expirationHandler checks if the face should expire (if on demand)
func (t *UnicastUDPTransport) expirationHandler() {
	for {
		time.Sleep(time.Duration(10) * time.Second)
		if t.state == ndn_defn.Down {
			break
		}
		if t.persistency == PersistencyOnDemand && (t.expirationTime.Before(time.Now()) ||
			t.expirationTime.Equal(time.Now())) {
			core.LogInfo(t, "Face expired")
			t.changeState(ndn_defn.Down)
			break
		}
	}
}

func (t *UnicastUDPTransport) sendFrame(frame []byte) {
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
	*t.expirationTime = time.Now().Add(udpLifetime)
}

func (t *UnicastUDPTransport) runReceive() {
	if lockThreadsToCores {
		runtime.LockOSThread()
	}

	recvBuf := make([]byte, ndn_defn.MaxNDNPacketSize)
	for {
		readSize, err := t.conn.Read(recvBuf)
		if err != nil {
			if err.Error() == "EOF" {
				core.LogDebug(t, "EOF")
			} else {
				core.LogWarn(t, "Unable to read from socket (", err, ") - DROP")
				t.onTransportFailure(true)
			}
			break
		}

		core.LogTrace(t, "Receive of size ", readSize)
		t.nInBytes += uint64(readSize)
		*t.expirationTime = time.Now().Add(udpLifetime)

		if readSize > ndn_defn.MaxNDNPacketSize {
			core.LogWarn(t, "Received too much data without valid TLV block - DROP")
			continue
		}

		// Send up to link service
		t.linkService.handleIncomingFrame(recvBuf[:readSize])
	}
}

func (t *UnicastUDPTransport) changeState(new ndn_defn.State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "state: ", t.state, " -> ", new)
	t.state = new

	if t.state != ndn_defn.Up {
		core.LogInfo(t, "Closing UDP socket")
		t.hasQuit <- true
		t.conn.Close()

		// Stop link service
		t.linkService.tellTransportQuit()

		FaceTable.Remove(t.faceID)
	}
}
