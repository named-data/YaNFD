/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package ndn

import (
	"net"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/named-data/YaNFD/core"
)

// URIType represents the type of the URI.
type URIType int

// const uriPattern = `^([0-9A-Za-z]+)://([0-9A-Za-z:-\[\]%\.]+)(:([0-9]+))?$“
const devPattern = `^(?P<scheme>dev)://(?P<ifname>[A-Za-z0-9\-]+)$`
const fdPattern = `^(?P<scheme>fd)://(?P<fd>[0-9]+)$`
const ipv4Pattern = `^((25[0-4]|2[0-4][0-9]|1[0-9][0-9]|[0-9][0-9]|[0-9])\.){3}(25[0-4]|2[0-4][0-9]|1[0-9][0-9]|[0-9][0-9]|[0-9])$`
const macPattern = `^(([0-9a-fA-F]){2}:){5}([0-9a-fA-F]){2}$`
const udpPattern = `^(?P<scheme>udp[46]?)://\[?(?P<host>[0-9A-Za-z\:\.\-]+)(%(?P<zone>[A-Za-z0-9\-]+))?\]?:(?P<port>[0-9]+)$`
const tcpPattern = `^(?P<scheme>tcp[46]?)://\[?(?P<host>[0-9A-Za-z\:\.\-]+)(%(?P<zone>[A-Za-z0-9\-]+))?\]?:(?P<port>[0-9]+)$`
const unixPattern = `^(?P<scheme>unix)://(?P<path>[/\\A-Za-z0-9\:\.\-_]+)$`

const (
	unknownURI URIType = iota
	devURI
	fdURI
	internalURI
	nullURI
	udpURI
	tcpURI
	unixURI
	wsURI
	wsclientURI
)

// URI represents a URI for a face.
type URI struct {
	uriType URIType
	scheme  string
	path    string
	port    uint16
}

// MakeDevFaceURI constucts a URI for a network interface.
func MakeDevFaceURI(ifname string) *URI {
	uri := new(URI)
	uri.uriType = devURI
	uri.scheme = "dev"
	uri.path = ifname
	uri.port = 0
	uri.Canonize()
	return uri
}

// MakeFDFaceURI constructs a file descriptor URI.
func MakeFDFaceURI(fd int) *URI {
	uri := new(URI)
	uri.uriType = fdURI
	uri.scheme = "fd"
	uri.path = strconv.Itoa(fd)
	uri.port = 0
	uri.Canonize()
	return uri
}

// MakeInternalFaceURI constructs an internal face URI.
func MakeInternalFaceURI() *URI {
	uri := new(URI)
	uri.uriType = internalURI
	uri.scheme = "internal"
	uri.path = ""
	uri.port = 0
	return uri
}

// MakeNullFaceURI constructs a null face URI.
func MakeNullFaceURI() *URI {
	uri := new(URI)
	uri.uriType = nullURI
	uri.scheme = "null"
	uri.path = ""
	uri.port = 0
	uri.Canonize()
	return uri
}

// MakeUDPFaceURI constructs a URI for a UDP face.
func MakeUDPFaceURI(ipVersion int, host string, port uint16) *URI {
	/*path := host
	if zone != "" {
		path += "%" + zone
	}*/
	uri := new(URI)
	uri.uriType = udpURI
	uri.scheme = "udp" + strconv.Itoa(ipVersion)
	uri.path = host
	uri.port = port
	uri.Canonize()
	return uri
}

// MakeTCPFaceURI constructs a URI for a TCP face.
func MakeTCPFaceURI(ipVersion int, host string, port uint16) *URI {
	uri := new(URI)
	uri.uriType = tcpURI
	uri.scheme = "tcp" + strconv.Itoa(ipVersion)
	uri.path = host
	uri.port = port
	uri.Canonize()
	return uri
}

// MakeUnixFaceURI constructs a URI for a Unix face.
func MakeUnixFaceURI(path string) *URI {
	uri := new(URI)
	uri.uriType = unixURI
	uri.scheme = "unix"
	uri.path = path
	uri.port = 0
	uri.Canonize()
	return uri
}

// MakeWebSocketServerFaceURI constructs a URI for a WebSocket server.
func MakeWebSocketServerFaceURI(u *url.URL) *URI {
	port, _ := strconv.ParseUint(u.Port(), 10, 16)
	return &URI{
		uriType: wsURI,
		scheme:  u.Scheme,
		path:    u.Hostname(),
		port:    uint16(port),
	}
}

// MakeWebSocketClientFaceURI constructs a URI for a WebSocket server.
func MakeWebSocketClientFaceURI(addr net.Addr) *URI {
	host, portStr, _ := net.SplitHostPort(addr.String())
	port, _ := strconv.ParseUint(portStr, 10, 16)
	return &URI{
		uriType: wsclientURI,
		scheme:  "wsclient",
		path:    host,
		port:    uint16(port),
	}
}

// DecodeURIString decodes a URI from a string.
func DecodeURIString(str string) *URI {
	u := new(URI)
	u.uriType = unknownURI
	u.scheme = "unknown"
	schemeSplit := strings.SplitN(str, ":", 2)
	if len(schemeSplit) < 2 {
		// No scheme
		return u
	}

	switch {
	case strings.EqualFold("dev", schemeSplit[0]):
		u.uriType = devURI
		u.scheme = "dev"

		regex, err := regexp.Compile(devPattern)
		if err != nil {
			return u
		}

		matches := regex.FindStringSubmatch(str)
		if regex.SubexpIndex("ifname") < 0 || len(matches) <= regex.SubexpIndex("ifname") {
			return u
		}

		ifname := matches[regex.SubexpIndex("ifname")]
		// Pure function is not allowed to have side effect
		// _, err = net.InterfaceByName(ifname)
		// if err != nil {
		// 	return u
		// }
		u.path = ifname
	case strings.EqualFold("fd", schemeSplit[0]):
		u.uriType = fdURI
		u.scheme = "fd"

		regex, err := regexp.Compile(fdPattern)
		if err != nil {
			return u
		}

		matches := regex.FindStringSubmatch(str)
		// fmt.Println(matches, len(matches), regex.SubexpIndex("fd"))
		if regex.SubexpIndex("fd") < 0 || len(matches) <= regex.SubexpIndex("fd") {
			return u
		}
		u.path = matches[regex.SubexpIndex("fd")]
	case strings.EqualFold("internal", schemeSplit[0]):
		u.uriType = internalURI
		u.scheme = "internal"
	case strings.EqualFold("null", schemeSplit[0]):
		u.uriType = nullURI
		u.scheme = "null"
	case strings.EqualFold("udp", schemeSplit[0]),
		strings.EqualFold("udp4", schemeSplit[0]),
		strings.EqualFold("udp6", schemeSplit[0]):
		u.uriType = udpURI
		u.scheme = "udp"

		regex, err := regexp.Compile(udpPattern)
		if err != nil {
			return u
		}

		matches := regex.FindStringSubmatch(str)
		if regex.SubexpIndex("host") < 0 || len(matches) <= regex.SubexpIndex("host") || regex.SubexpIndex("port") < 0 || len(matches) <= regex.SubexpIndex("port") {
			return u
		}
		u.path = matches[regex.SubexpIndex("host")]
		if regex.SubexpIndex("zone") < 0 || len(matches) >= regex.SubexpIndex("zone") && matches[regex.SubexpIndex("zone")] != "" {
			u.path += "%" + matches[regex.SubexpIndex("zone")]
		}
		port, err := strconv.ParseUint(matches[regex.SubexpIndex("port")], 10, 16)
		if err != nil || port <= 0 || port > 65535 {
			return u
		}
		u.port = uint16(port)
	case strings.EqualFold("tcp", schemeSplit[0]),
		strings.EqualFold("tcp4", schemeSplit[0]),
		strings.EqualFold("tcp6", schemeSplit[0]):
		u.uriType = tcpURI
		u.scheme = "tcp"

		regex, err := regexp.Compile(tcpPattern)
		if err != nil {
			return u
		}

		matches := regex.FindStringSubmatch(str)
		if regex.SubexpIndex("host") < 0 || len(matches) <= regex.SubexpIndex("host") || regex.SubexpIndex("port") < 0 || len(matches) <= regex.SubexpIndex("port") {
			return u
		}
		u.path = matches[regex.SubexpIndex("host")]
		if regex.SubexpIndex("zone") < 0 || len(matches) >= regex.SubexpIndex("zone") && matches[regex.SubexpIndex("zone")] != "" {
			u.path += "%" + matches[regex.SubexpIndex("zone")]
		}
		port, err := strconv.ParseUint(matches[regex.SubexpIndex("port")], 10, 16)
		if err != nil || port <= 0 || port > 65535 {
			return u
		}
		u.port = uint16(port)
	case strings.EqualFold("unix", schemeSplit[0]):
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
	case strings.EqualFold("ws", schemeSplit[0]),
		strings.EqualFold("wss", schemeSplit[0]):
		uri, e := url.Parse(str)
		if e != nil || uri.User != nil || strings.TrimLeft(uri.Path, "/") != "" ||
			uri.RawQuery != "" || uri.Fragment != "" {
			return nil
		}
		return MakeWebSocketServerFaceURI(uri)
	case strings.EqualFold("wsclient", schemeSplit[0]):
		addr, e := net.ResolveTCPAddr("tcp", strings.Trim(schemeSplit[1], "/"))
		if e != nil {
			return nil
		}
		return MakeWebSocketClientFaceURI(addr)
	}

	// Canonize, if possible
	u.Canonize()
	return u
}

// URIType returns the type of the face URI.
func (u *URI) URIType() URIType {
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
	case devURI:
		return u.scheme == "dev" && u.path != "" && u.port == 0
	case fdURI:
		fd, err := strconv.Atoi(u.path)
		return u.scheme == "fd" && err == nil && fd >= 0 && u.port == 0
	case internalURI:
		return u.scheme == "internal" && u.path == "" && u.port == 0
	case nullURI:
		return u.scheme == "null" && u.path == "" && u.port == 0
	case udpURI:
		// Split off zone, if any
		ip := net.ParseIP(u.PathHost())
		// Port number is implicitly limited to <= 65535 by type uint16
		// We have to test whether To16() && not IPv4 because the Go net library considers IPv4 addresses to be valid IPv6 addresses
		isIPv4, _ := regexp.MatchString(ipv4Pattern, u.PathHost())
		return ip != nil && ((u.scheme == "udp4" && ip.To4() != nil) || (u.scheme == "udp6" && ip.To16() != nil && !isIPv4)) && u.port > 0
	case tcpURI:
		// Split off zone, if any
		ip := net.ParseIP(u.PathHost())
		// Port number is implicitly limited to <= 65535 by type uint16
		// We have to test whether To16() && not IPv4 because the Go net library considers IPv4 addresses to be valid IPv6 addresses
		isIPv4, _ := regexp.MatchString(ipv4Pattern, u.PathHost())
		return ip != nil && u.port > 0 && ((u.scheme == "tcp4" && ip.To4() != nil) ||
			(u.scheme == "tcp6" && ip.To16() != nil && !isIPv4))
	case unixURI:
		// Do not check whether file exists, because it may fail due to lack of priviledge in testing environment
		return u.scheme == "unix" && u.path != "" && u.port == 0
	default:
		// Of unknown type
		return false
	}
}

// Canonize attempts to canonize the URI, if not already canonical.
func (u *URI) Canonize() error {
	switch u.uriType {
	case devURI, fdURI:
		// Nothing to do to canonize these
	case udpURI, tcpURI:
		path := u.path
		zone := ""
		if strings.Contains(u.path, "%") {
			// Has zone, so separate out
			path = u.PathHost()
			zone = "%" + u.PathZone()
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
			if u.uriType == udpURI {
				u.scheme = "udp4"
			} else {
				u.scheme = "tcp4"
			}
			u.path = ip.String() + zone
		} else if ip.To16() != nil {
			if u.uriType == udpURI {
				u.scheme = "udp6"
			} else {
				u.scheme = "tcp6"
			}
			u.path = ip.String() + zone
		} else {
			return core.ErrNotCanonical
		}
	case unixURI:
		u.scheme = "unix"
		testPath := "/" + u.path
		if runtime.GOOS == "windows" {
			testPath = u.path
		}
		fileInfo, err := os.Stat(testPath)
		if err != nil && !os.IsNotExist(err) {
			// File couldn't be opened, but not just because it doesn't exist
			return core.ErrNotCanonical
		} else if err == nil && fileInfo.IsDir() {
			// File is a directory
			return core.ErrNotCanonical
		}
		u.port = 0
	default:
		return core.ErrNotCanonical
	}

	return nil
}

// Scope returns the scope of the URI.
func (u *URI) Scope() Scope {
	if !u.IsCanonical() {
		return Unknown
	}

	switch u.uriType {
	case devURI:
		return NonLocal
	case fdURI:
		return Local
	case nullURI:
		return NonLocal
	case udpURI:
		if net.ParseIP(u.path).IsLoopback() {
			return Local
		}
		return NonLocal
	case unixURI:
		return Local
	}

	// Only valid types left is internal, which is by definition local
	return Local
}

func (u *URI) String() string {
	switch u.uriType {
	case devURI:
		return "dev://" + u.path
	case fdURI:
		return "fd://" + u.path
	case internalURI:
		return "internal://"
	case nullURI:
		return "null://"
	case udpURI, tcpURI, wsURI, wsclientURI:
		return u.scheme + "://" + net.JoinHostPort(u.path, strconv.FormatUint(uint64(u.port), 10))
	case unixURI:
		return u.scheme + "://" + u.path
	default:
		return "unknown://"
	}
}
