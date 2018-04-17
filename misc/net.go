package misc

import (
"net"
"strings"
)

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


