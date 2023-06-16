package common

import (
	"fmt"
	"net/url"
	"strings"

	operatorProxy "github.com/operator-framework/operator-lib/proxy"
)

type ProxyInfo struct {
	host string
	port string
}

func (pi *ProxyInfo) Host() string {
	return pi.host
}

func (pi *ProxyInfo) Port() string {
	return pi.port
}

func NewProxyInfo() *ProxyInfo {
	proxy := ProxyInfo{}
	proxy.host, proxy.port = getProxyInfo()
	return &proxy

}

func getProxyInfo() (string, string) {
	if len(operatorProxy.ReadProxyVarsFromEnv()) > 0 {
		p := operatorProxy.ReadProxyVarsFromEnv()
		for _, v := range p {
			switch v.Name {
			case "HTTP_PROXY":
				return proxyHostPort(v.Value)
			case "HTTPS_PROXY":
				return proxyHostPort(v.Value)
			}
		}
	}
	return "", ""
}

func proxyHostPort(proxy string) (string, string) {
	url, _ := url.Parse(proxy)
	s := url.Scheme
	h := url.Host
	u := url.User.String()
	port := ""

	if s != "" {
		s = fmt.Sprintf("%s://", s)
	}

	if strings.Contains(url.Host, ":") {
		host := strings.Split(url.Host, ":")
		h = host[0]
		port = host[1]
	}
	if u != "" {
		u = fmt.Sprintf("%s@", url.User)
	}

	return fmt.Sprintf("%s%s%s", s, u, h), port
}
