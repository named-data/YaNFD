/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"net"
	"os"
	"regexp"
	"strconv"

	"github.com/eric135/YaNFD/core"
)

// URIType represents the type of the URI
type uriType int

const ethernetPattern = "^\\[?([0-9A-Fa-f][-:]){5}([0-9A-Fa-f]\\]?$"

//const ip4Pattern = "^((25[0-4]|2[0-4][0-9]|1[0-9][0-9]|[0-9][0-9]|[0-9])\\.){3}(25[0-4]|2[0-4][0-9]|1[0-9][0-9]|[0-9][0-9]|[0-9])$"

const (
	nullURI     uriType = iota
	ethernetURI uriType = iota
	udpURI      uriType = iota
	unixURI     uriType = iota
)

// URI represents a URI for a face
type URI struct {
	uriType uriType
	scheme  string
	path    string
	port    uint16
}

// MakeNullFaceURI constructs a null face URI
func MakeNullFaceURI() URI {
	return URI{nullURI, "null", "", 0}
}

// MakeEthernetFaceURI constructs a URI for an Ethernet face
func MakeEthernetFaceURI(address string) URI {
	return URI{ethernetURI, "eth", address, 0}
}

// MakeUDPFaceURI constructs a URI for a UDP face
func MakeUDPFaceURI(ipVersion int, host string, port uint16) URI {
	return URI{udpURI, "udp" + strconv.Itoa(ipVersion), host, port}
}

// MakeUnixFaceURI constructs a URI for a Unix face
func MakeUnixFaceURI(path string) URI {
	return URI{unixURI, "unix", path, 0}
}

// GetURIType returns the type of the face URI
func (u *URI) getType() uriType {
	return u.uriType
}

// Scheme returns the scheme of the face URI
func (u *URI) Scheme() string {
	return u.scheme
}

// Path returns the path of the face URI
func (u *URI) Path() string {
	return u.path
}

// Port returns the port of the face URI
func (u *URI) Port() uint16 {
	return u.port
}

// IsCanonical returns whether the face URI is canonical
func (u *URI) IsCanonical() bool {
	// Must pass type-specific checks
	switch u.uriType {
	case nullURI:
		return u.scheme == "null" && u.path == "" && u.port == 0
	case ethernetURI:
		isEthernet, _ := regexp.MatchString(ethernetPattern, u.path)
		return u.scheme == "eth" && isEthernet && u.port == 0
	case udpURI:
		ip := net.ParseIP(u.path)
		// Port number is implicitly limited to <= 65535 by type uint16
		return ip != nil && ((u.scheme == "udp4" && ip.To4() != nil) || (u.scheme == "udp6" && ip.To16() != nil)) && u.port > 0
	case unixURI:
		// Check whether file exists
		_, err := os.Stat(u.path)
		return u.scheme == "unix" && err == nil && u.port == 0
	default:
		// Of unknown type
		return false
	}
}

// Canonize attempts to canonize the URI, if not already canonical.
func (u *URI) Canonize() error {
	if u.IsCanonical() {
		return nil
	}

	if u.uriType == ethernetURI {
		// TODO
	} else if u.uriType == udpURI {
		ip := net.ParseIP(u.path)
		if ip == nil {
			// TODO
		}

		if ip.To4() != nil {
			u.scheme = "udp4"
		} else if ip.To16() != nil {
			u.scheme = "udp6"
		} else {
			return core.ErrNotCanonical
		}
	} else if u.uriType == unixURI {
		// TODO
	}

	return nil
}

// Scope returns the scope of the URI.
func (u *URI) Scope() Scope {
	if !u.IsCanonical() {
		return Unknown
	}

	if u.uriType == ethernetURI {
		return NonLocal
	} else if u.uriType == udpURI {
		if net.ParseIP(u.path).IsLoopback() {
			return Local
		}
		return NonLocal
	} else if u.uriType == unixURI {
		return Local
	}

	// Only type left is null, which is by definition local
	return Local
}

func (u *URI) String() string {
	if u.uriType == ethernetURI {
		return u.scheme + "://[" + u.path + "]"
	} else if u.uriType == udpURI {
		if u.scheme == "udp4" {
			return u.scheme + "://" + u.path + ":" + strconv.FormatUint(uint64(u.port), 10)
		} else if u.scheme == "udp6" {
			return u.scheme + "://[" + u.path + "]:" + strconv.FormatUint(uint64(u.port), 10)
		} else {
			return "null://"
		}
	} else if u.uriType == unixURI {
		return u.scheme + "://" + u.path
	} else {
		return "unknown://"
	}
}
