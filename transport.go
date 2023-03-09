package httpclient

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

const (
	defaultMaxIdleConnsPerHost = 10
	defaultIdleConnTimeout     = 10 * time.Second
	defaultDialTimeout         = 2 * time.Second
	defaultTLSHandshakeTimeout = 5 * time.Second
	defaultTimeout             = 2 * time.Second
)

func buildDefaultTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConnsPerHost: defaultMaxIdleConnsPerHost,
		IdleConnTimeout:     defaultIdleConnTimeout,
		Dial: (&net.Dialer{
			Timeout: defaultDialTimeout,
		}).Dial,
		TLSHandshakeTimeout: defaultTLSHandshakeTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}
}
