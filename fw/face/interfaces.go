/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"bytes"
	"net"
)

// InterfaceByMAC gets an interface by its MAC address.
func InterfaceByMAC(mac net.HardwareAddr) (*net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		if bytes.Equal(iface.HardwareAddr, mac) {
			return &iface, nil
		}
	}

	return nil, nil
}

// InterfaceByIP gets an interface by its IP address.
func InterfaceByIP(ip net.IP) (*net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		// If there's an error getting addresses, just skip
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			hostAddr := addr.(*net.IPNet)
			if hostAddr.IP.Equal(ip) {
				return &iface, nil
			}
		}
	}

	return nil, nil
}
