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
	"strings"

	"github.com/eric135/YaNFD/core"
)

// URIType represents the type of the URI.
type uriType int

//const uriPattern = "^([0-9A-Za-z]+)://([0-9A-Za-z:-\\[\\]%\\.]+)(:([0-9]+))?$"
const ethernetPattern = "^(?P<scheme>eth)://\\[(?P<mac>(([0-9a-fA-F]){2}:){5}([0-9a-fA-F]){2}(?P<zone>\\%[A-Za-z0-9])*)\\]$"
const ipv4Pattern = "^((25[0-4]|2[0-4][0-9]|1[0-9][0-9]|[0-9][0-9]|[0-9])\\.){3}(25[0-4]|2[0-4][0-9]|1[0-9][0-9]|[0-9][0-9]|[0-9])$"
const macPattern = "^(([0-9a-fA-F]){2}:){5}([0-9a-fA-F]){2}$"
const nullPattern = "^(null)://$"
const udpPattern = "^(?P<scheme>udp[46]?)://\\[?(?P<host>[0-9A-Za-z\\:\\.\\-]+)\\]?:(?P<port>[0-9]+)$"
const unixPattern = "^(unix)://([/\\\\A-Za-z0-9\\.\\-_]+)$"

const (
	unknownURI  uriType = iota
	nullURI     uriType = iota
	ethernetURI uriType = iota
	udpURI      uriType = iota
	unixURI     uriType = iota
)

// URI represents a URI for a face.
type URI struct {
	uriType uriType
	scheme  string
	path    string
	port    uint16
}

// MakeNullFaceURI constructs a null face URI.
func MakeNullFaceURI() URI {
	return URI{nullURI, "null", "", 0}
}

// MakeEthernetFaceURI constructs a URI for an Ethernet face.
func MakeEthernetFaceURI(mac net.HardwareAddr) URI {
	return URI{ethernetURI, "eth", mac.String(), 0}
}

// MakeUDPFaceURI constructs a URI for a UDP face.
func MakeUDPFaceURI(ipVersion int, host string, port uint16) URI {
	return URI{udpURI, "udp" + strconv.Itoa(ipVersion), host, port}
}

// MakeUnixFaceURI constructs a URI for a Unix face.
func MakeUnixFaceURI(path string) URI {
	return URI{unixURI, "unix", path, 0}
}

// DecodeURIString decodes a URI from a string.
func DecodeURIString(str string) (u URI) {
	u.uriType = unknownURI
	u.scheme = "unknown"
	schemeSplit := strings.SplitN(str, ":", 2)
	if len(schemeSplit) < 2 {
		// No scheme
		return u
	}

	if strings.EqualFold("null", schemeSplit[0]) {
		u.uriType = nullURI
		u.scheme = "null"
	} else if strings.EqualFold("eth", schemeSplit[0]) {
		u.uriType = ethernetURI
		u.scheme = "eth"

		regex, err := regexp.Compile(ethernetPattern)
		if err != nil {
			return u
		}

		matches := regex.FindStringSubmatch(str)
		if len(matches) <= regex.SubexpIndex("mac") {
			return u
		}
		u.path = matches[regex.SubexpIndex("mac")]
	} else if strings.EqualFold("udp", schemeSplit[0]) || strings.EqualFold("udp4", schemeSplit[0]) || strings.EqualFold("udp6", schemeSplit[0]) {
		u.uriType = udpURI
		u.scheme = "udp"

		regex, err := regexp.Compile(udpPattern)
		if err != nil {
			return u
		}

		matches := regex.FindStringSubmatch(str)
		if len(matches) <= regex.SubexpIndex("host") || len(matches) <= regex.SubexpIndex("port") {
			return u
		}
		u.path = matches[regex.SubexpIndex("host")]
		port, err := strconv.Atoi(matches[regex.SubexpIndex("port")])
		if err != nil || port <= 0 || port > 65535 {
			return u
		}
		u.port = uint16(port)
	} else if strings.EqualFold("unix", schemeSplit[0]) {
		u.uriType = unixURI
		u.scheme = "unix"

		regex, err := regexp.Compile(unixPattern)
		if err != nil {
			return u
		}

		matches := regex.FindStringSubmatch(str)
		if len(matches) != 3 {
			return u
		}
		u.path = matches[2]
	}

	// Canonize, if possible
	u.Canonize()
	return u
}

// GetURIType returns the type of the face URI.
func (u *URI) getType() uriType {
	return u.uriType
}

// Scheme returns the scheme of the face URI.
func (u *URI) Scheme() string {
	return u.scheme
}

// Path returns the path of the face URI.
func (u *URI) Path() string {
	return u.path
}

// PathHost returns the host component of the path of the face URI.
func (u *URI) PathHost() string {
	pathComponents := strings.Split(u.path, "%")
	if len(pathComponents) < 1 {
		return ""
	}
	return pathComponents[0]
}

// PathZone returns the zone component of the path of the face URI.
func (u *URI) PathZone() string {
	pathComponents := strings.Split(u.path, "%")
	if len(pathComponents) < 2 {
		return ""
	}
	return pathComponents[1]
}

// Port returns the port of the face URI.
func (u *URI) Port() uint16 {
	return u.port
}

// IsCanonical returns whether the face URI is canonical.
func (u *URI) IsCanonical() bool {
	// Must pass type-specific checks
	switch u.uriType {
	case nullURI:
		return u.scheme == "null" && u.path == "" && u.port == 0
	case ethernetURI:
		isEthernet, _ := regexp.MatchString(macPattern, u.path)
		return u.scheme == "eth" && isEthernet && u.port == 0
	case udpURI:
		// Split off zone, if any
		ip := net.ParseIP(u.PathHost())
		// Port number is implicitly limited to <= 65535 by type uint16
		// We have to test whether To16() && not IPv4 because the Go net library considers IPv4 addresses to be valid IPv6 addresses
		isIPv4, _ := regexp.MatchString(ipv4Pattern, u.PathHost())
		return ip != nil && ((u.scheme == "udp4" && ip.To4() != nil) || (u.scheme == "udp6" && ip.To16() != nil && !isIPv4)) && u.port > 0
	case unixURI:
		// Check whether file exists
		fileInfo, err := os.Stat("/" + u.path)
		return u.scheme == "unix" && ((err == nil && !fileInfo.IsDir()) || os.IsNotExist(err)) && u.port == 0
	default:
		// Of unknown type
		return false
	}
}

// Canonize attempts to canonize the URI, if not already canonical.
func (u *URI) Canonize() error {
	if u.uriType == ethernetURI {
		mac, err := net.ParseMAC(strings.Trim(u.path, "[]"))
		if err != nil {
			return core.ErrNotCanonical
		}
		u.scheme = "eth"
		u.path = mac.String()
		u.port = 0
	} else if u.uriType == udpURI {
		path := u.path
		if strings.Contains(u.path, "%") {
			// Has zone, so separate out
			path = u.PathHost()
		}
		ip := net.ParseIP(strings.Trim(path, "[]"))
		if ip == nil {
			// Resolve DNS
			resolvedIPs, err := net.LookupHost(path)
			if err != nil || len(resolvedIPs) == 0 {
				return core.ErrNotCanonical
			}
			ip = net.ParseIP(resolvedIPs[0])
			if ip == nil {
				return core.ErrNotCanonical
			}
		}

		if ip.To4() != nil {
			u.scheme = "udp4"
			u.path = ip.String()
		} else if ip.To16() != nil {
			u.scheme = "udp6"
			u.path = ip.String()
		} else {
			return core.ErrNotCanonical
		}
	} else if u.uriType == unixURI {
		u.scheme = "unix"
		fileInfo, err := os.Stat("/" + u.path)
		if err != nil && !os.IsNotExist(err) {
			// File couldn't be opened, but not just becaue it doesn't exit
			return core.ErrNotCanonical
		} else if err == nil && fileInfo.IsDir() {
			// File is a directory
			return core.ErrNotCanonical
		}
		u.port = 0
	} else {
		return core.ErrNotCanonical
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
	if u.uriType == nullURI {
		return "null://"
	} else if u.uriType == ethernetURI {
		return u.scheme + "://[" + u.path + "]"
	} else if u.uriType == udpURI {
		if u.scheme == "udp4" {
			return u.scheme + "://" + u.path + ":" + strconv.FormatUint(uint64(u.port), 10)
		} else if u.scheme == "udp6" {
			return u.scheme + "://[" + u.path + "]:" + strconv.FormatUint(uint64(u.port), 10)
		} else {
			return u.scheme + "://" + u.path + ":" + strconv.FormatUint(uint64(u.port), 10)
		}
	} else if u.uriType == unixURI {
		return u.scheme + "://" + u.path
	} else {
		return "unknown://"
	}
}