package util

import (
	"net"
	"net/http"
	"strings"

	"github.com/TwilyName/website/config"
)

func IsWhitelistedProxy(req *http.Request) bool {
	// request is from reverse proxy?
	_, ok := req.Header["X-Real-Ip"]
	if !ok {
		_, ok = req.Header["X-Forwarded-For"]
	}
	if !ok {
		// request does not contains headers specific for reverse proxy
		return true
	}

	ipstr, _, _ := net.SplitHostPort(req.RemoteAddr)
	ip := net.ParseIP(ipstr)
	for _, wl := range config.Get().ReverseProxy.Whitelist {
		if config.IsAllowedByACL(ip, wl) {
			return true
		}
	}
	return false
}

func GetRemoteAddr(req *http.Request) net.IP {
	ipstr, _, _ := net.SplitHostPort(req.RemoteAddr)
	ip := net.ParseIP(ipstr)

	if !IsWhitelistedProxy(req) {
		return ip
	}

	if val, ok := req.Header["X-Real-Ip"]; ok {
		return net.ParseIP(val[0])
	}

	if val, ok := req.Header["X-Forwarded-For"]; ok {
		vals := strings.Split(val[0], ",")
		return net.ParseIP(vals[0])
	}

	return ip
}
