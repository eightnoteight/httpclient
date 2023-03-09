package httpclient

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

type ClientConfig struct {
	Transport     *http.Transport
	Client        *http.Client
	ClientTimeout time.Duration
}

type StaticRequestConfig struct {
	Scheme   string
	User     string
	Password string
	Host     string
	Headers  http.Header
}

func (c StaticRequestConfig) Clone() StaticRequestConfig {
	return StaticRequestConfig{
		Scheme:  c.Scheme,
		User:    c.User,
		Host:    c.Host,
		Headers: c.Headers.Clone(),
	}
}

type RuntimeRequestConfig struct {
	Headers      http.Header
	Body         []byte
	ResponseJSON interface{}
	ResponseBody io.Writer
	Method       HttpMethod
	Path         string
	Query        url.Values
}

type Config struct {
	ClientConfig         ClientConfig
	StaticRequestConfig  StaticRequestConfig
	RuntimeRequestConfig RuntimeRequestConfig
}

type ClientOption func(c ClientConfig) (ClientConfig, error)
type StaticRequestOption func(c StaticRequestConfig) (StaticRequestConfig, error)
type RuntimeRequestOption func(c RuntimeRequestConfig) (RuntimeRequestConfig, error)

type ConfigOptions struct {
	ClientOptions         []ClientOption
	StaticRequestOptions  []StaticRequestOption
	RuntimeRequestOptions []RuntimeRequestOption
}

func NewConfig(co ConfigOptions) (Config, error) {
	c := Config{
		ClientConfig: ClientConfig{
			ClientTimeout: defaultTimeout,
		},
	}

	// apply client options
	for _, opt := range co.ClientOptions {
		var err error
		c.ClientConfig, err = opt(c.ClientConfig)
		if err != nil {
			return c, err // todo: error wrapping
		}
	}
	// apply static request options
	for _, opt := range co.StaticRequestOptions {
		var err error
		c.StaticRequestConfig, err = opt(c.StaticRequestConfig)
		if err != nil {
			return c, err // todo: error wrapping
		}
	}
	// apply runtime request options
	for _, opt := range co.RuntimeRequestOptions {
		var err error
		c.RuntimeRequestConfig, err = opt(c.RuntimeRequestConfig)
		if err != nil {
			return c, err // todo: error wrapping
		}
	}

	return c, nil
}

// generate config from yaml node or json node. this will be helpful in standardizing config across services config.yaml file
