package misc

import (
	"net"
	"strconv"
	"strings"
)

// Faster version of net.SplitHostPort which performs no checks, and returns
// the host part only of an hostport pair.
func LooselyGetHost(hostport string) string {
	hoststart, hostend := 0, 0
	if len(hostport) >= 1 && hostport[0] == '[' {
		hoststart = 1
		hostend = strings.IndexByte(hostport, ']')
	} else {
		hostend = strings.IndexByte(hostport, ':')
	}
	if hostend < 0 {
		hostend = len(hostport)
	}
	return hostport[hoststart:hostend]
}

// Like net.JoinHostPort, but the port is added only if != 0.
func OptionallyJoinHostPort(host string, port int) string {
	is_ipv6 := strings.IndexByte(host, ':') >= 0
	has_port := port > 0
	if is_ipv6 {
		host = "[" + host + "]"
	}
	if has_port {
		host += ":" + strconv.Itoa(port)
	}
	return host
}

// Like net.SplitHostPort, but the port is optional.
// If no port is specified, an empty string will be returned.
func SplitHostPort(hostport string) (host, port string, err error) {
	addrErr := func(addr, why string) (host, port string, err error) {
		return "", "", &net.AddrError{Err: why, Addr: addr}
	}

	hoststart, hostend := 0, 0
	portstart := len(hostport)
	if len(hostport) >= 1 && hostport[0] == '[' {
		hoststart = 1
		hostend = strings.IndexByte(hostport, ']')
		if hostend < 0 {
			return addrErr(hostport, "missing ']' in address")
		}
		portstart = hostend + 1
	} else {
		hostend = strings.IndexByte(hostport, ':')
		if hostend < 0 {
			hostend = len(hostport)
		}
		portstart = hostend
	}
	if portstart < len(hostport) {
		if hostport[portstart] != ':' {
			return addrErr(hostport, "invalid character at the end of address, expecting ':'")
		}
		portstart += 1
	}

	port = hostport[portstart:]
	host = hostport[hoststart:hostend]

	if strings.IndexByte(port, ':') >= 0 {
		return addrErr(hostport, "too many colons in suspected port number")
	}
	if strings.IndexByte(port, ']') >= 0 {
		return addrErr(hostport, "unexpected ']' in port")
	}
	if strings.IndexByte(port, '[') >= 0 {
		return addrErr(hostport, "unexpected '[' in port")
	}
	if strings.IndexByte(host, '[') >= 0 {
		return addrErr(hostport, "unexpected '[' in host")
	}
	if strings.IndexByte(host, ']') >= 0 {
		return addrErr(hostport, "unexpected ']' in host")
	}

	return host, port, nil
}
