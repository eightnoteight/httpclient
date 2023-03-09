package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	stdClient            *http.Client
	staticRequestConfig  StaticRequestConfig
	runtimeRequestConfig RuntimeRequestConfig
}

func NewHTTPClientFromConfig(cfg Config) (*Client, error) {
	if cfg.ClientConfig.Client != nil && cfg.ClientConfig.Transport != nil {
		return nil, fmt.Errorf("cannot set both client and transport")
	}
	var stdClient *http.Client
	if cfg.ClientConfig.Client != nil {
		stdClient = cfg.ClientConfig.Client
	} else {
		transport := buildDefaultTransport()
		if cfg.ClientConfig.Transport != nil {
			transport = cfg.ClientConfig.Transport
		}
		stdClient = &http.Client{
			Transport: transport,
			Timeout:   cfg.ClientConfig.ClientTimeout,
		}
	}

	return &Client{
		stdClient:            stdClient,
		staticRequestConfig:  cfg.StaticRequestConfig,
		runtimeRequestConfig: cfg.RuntimeRequestConfig,
	}, nil
}

func NewHTTPClient(co ConfigOptions) (*Client, error) {
	cfg, err := NewConfig(co)
	if err != nil {
		return nil, err
	}
	return NewHTTPClientFromConfig(cfg)
}

func (c *Client) Send(ctx context.Context, opts ...RuntimeRequestOption) error {
	rrc, err := buildRuntimeRequestConfig(c.runtimeRequestConfig, opts...)
	if err != nil {
		return err
	}

	// validate that the response destination is given
	if rrc.ResponseJSON == nil && rrc.ResponseBody == nil {
		return fmt.Errorf("no response destination given")
	} else if rrc.ResponseJSON != nil && rrc.ResponseBody != nil {
		return fmt.Errorf("cannot set both response json and response body")
	}

	req, err := c.buildRequest(ctx, rrc)
	if err != nil {
		return err
	}

	resp, err := c.stdClient.Do(req)
	if err != nil {
		return err // todo: figure out how to extract url from request and add to error
	}
	defer resp.Body.Close()
	// if no error is returned, the response will contain a non-nil resp.Body which the user is expected to close.
	// so we can read the body here and return the error if any
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("not ok error code: %d", resp.StatusCode) // todo: error context with urlStr
	}
	// todo: response body contract validations
	// todo: request body contract validations
	if c.runtimeRequestConfig.ResponseJSON == nil {
		_, err := io.Copy(c.runtimeRequestConfig.ResponseBody, bytes.NewReader(body))
		if err != nil {
			return err
		}
	} else {
		err := json.Unmarshal(body, c.runtimeRequestConfig.ResponseJSON)
		if err != nil {
			return err
		}
	}
	return nil
}
func buildRuntimeRequestConfig(rrc RuntimeRequestConfig, opts ...RuntimeRequestOption) (RuntimeRequestConfig, error) {
	for _, opt := range opts {
		var err error
		rrc, err = opt(rrc)
		if err != nil {
			return RuntimeRequestConfig{}, err
		}
	}
	return rrc, nil
}

func (c *Client) buildRequest(ctx context.Context, rrc RuntimeRequestConfig) (*http.Request, error) {
	var rqBody io.Reader = nil
	if rrc.Body != nil {
		rqBody = bytes.NewReader(rrc.Body)
	}

	urlStr := buildURLString(c.staticRequestConfig, rrc)

	rq, err := http.NewRequestWithContext(
		ctx, string(c.runtimeRequestConfig.Method),
		urlStr,
		rqBody,
	)
	if err != nil {
		return nil, err // todo: error context with urlStr
	}
	return rq, nil
}

func buildURLString(staticRequestConfig StaticRequestConfig, runtimeRequestConfig RuntimeRequestConfig) string {
	var userinfo *url.Userinfo
	if staticRequestConfig.User != "" && staticRequestConfig.Password != "" {
		userinfo = url.UserPassword(staticRequestConfig.User, staticRequestConfig.Password)
	} else if staticRequestConfig.User != "" {
		userinfo = url.User(staticRequestConfig.User)
	}
	u := url.URL{
		Scheme:   staticRequestConfig.Scheme,
		User:     userinfo,
		Host:     staticRequestConfig.Host,
		Path:     runtimeRequestConfig.Path,
		RawQuery: runtimeRequestConfig.Query.Encode(),
	}
	return u.String()
}

// // todo: have some other api where response consumption is very easy and intuitive
// func (c *Client) SendRaw(ctx context.Context, opts ...RuntimeRequestOption) (*http.Response, error) {
// }
