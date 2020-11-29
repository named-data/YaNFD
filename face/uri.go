/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"errors"
	"net"
	"os"
	"regexp"
	"strconv"
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

// NewNullFaceURI constructs an empty face URI
func NewNullFaceURI() URI {
	return URI{nullURI, "", "", 0}
}

// NewEthernetFaceURI constructs a URI for an Ethernet face
func NewEthernetFaceURI(address string) URI {
	return URI{ethernetURI, "eth", address, 0}
}

// NewUDPFaceURI constructs a URI for a UDP face
func NewUDPFaceURI(ipVersion int, host string, port uint16) URI {
	return URI{udpURI, "udp" + strconv.Itoa(ipVersion), host, port}
}

// NewUnixFaceURI constructs a URI for a Unix face
func NewUnixFaceURI(path string) URI {
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
		return ip != nil && ((u.scheme == "udp4" && ip.To4() != nil) || (u.scheme == "udp6" && ip.To16() != nil)) &&
			u.port > 0 && u.port <= 65535
	case unixURI:
		// Check whether file exists
		_, err := os.Stat(u.path)
		return u.scheme == "unix" && err == nil && u.port == 0
	default:
		// Of unknown type
		return false
	}
}

// Canonize attempts to canonize the URI, if not already canonical
func (u *URI) Canonize() error {
	if u.IsCanonical() {
		return nil
	}
	// TODO
	return errors.New("URI could not be canonized")
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
		return "null://"
	}
}
