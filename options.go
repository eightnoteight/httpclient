package httpclient

import (
	"encoding/json"
	"errors" // todo: replace with self errors library that adds stack trace and passing context
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eightnoteight/httpclient/gstrings"
	"github.com/eightnoteight/httpclient/integers"
)

// transport / client configuration options
func WithInsecureSkipVerify(val bool) ClientOption {
	return func(c ClientConfig) (ClientConfig, error) {
		// check whether necessary variables are set
		if c.Transport == nil {
			return c, errors.New("transport is not set")
		}
		if c.Transport.TLSClientConfig == nil {
			return c, errors.New("tls client config is not set")
		}

		// set insecure skip verify
		c.Transport.TLSClientConfig.InsecureSkipVerify = val
		return c, nil
	}
}

// static request configuration options
func WithScheme(scheme string) StaticRequestOption {
	return func(c StaticRequestConfig) (StaticRequestConfig, error) {
		err := validateScheme(scheme)
		if err != nil {
			return c, err
		}
		c.Scheme = scheme

		return c, nil
	}
}

func validateScheme(scheme string) error {
	if scheme == "" {
		return errors.New("scheme is empty")
	}
	if scheme != "http" && scheme != "https" {
		return errors.New("scheme is not http or https")
	}
	return nil
}

// WithHostPort sets the host and port of the request.
// It validates the host and port, and returns an error if the host or port is invalid.
// note that the port is compulsory and it doesn't default to 80 or 443.
func WithHostPort(hostPort string) StaticRequestOption {
	return func(c StaticRequestConfig) (StaticRequestConfig, error) {
		err := validateHostPort(hostPort)
		if err != nil {
			return c, err
		}
		c.Host = hostPort
		return c, nil
	}
}

func validateHostPort(hostPort string) error {
	host, port, err := net.SplitHostPort(hostPort)
	if err != nil {
		return fmt.Errorf("hostPort is not a valid host:port. input: %s, err: %w", hostPort, err)
	}
	if err := validateHost(host); err != nil {
		return err
	}
	if err := validatePort(port); err != nil {
		return err
	}
	return nil
}

func validateHost(host string) error {
	if host == "" {
		return errors.New("host is empty")
	}
	return nil
}

// https://www.sciencedirect.com/topics/computer-science/registered-port
// better to enforce both on client and server side
const maxValidConnectPort = 49152

func validatePort(portStr string) error {
	port, err := strconv.ParseUint(portStr, 10, 32)
	if err != nil {
		return fmt.Errorf("port is not a valid uint16. input: %s, err: %w", portStr, err)
	}
	if port < 1 || port > maxValidConnectPort {
		return fmt.Errorf("port is not in range 1-%d", maxValidConnectPort)
	}
	return nil
}

func WithHostHeader(hostHeader string) StaticRequestOption {
	return func(c StaticRequestConfig) (StaticRequestConfig, error) {
		err := validateHostHeader(hostHeader)
		if err != nil {
			return c, err
		}
		c = c.Clone()
		if c.Headers == nil {
			c.Headers = make(http.Header)
		}
		c.Headers.Set("Host", hostHeader)
		return c, nil
	}
}

func validateHostHeader(hostHeader string) error {
	if hostHeader == "" {
		return errors.New("hostHeader is empty")
	}
	return nil
}

func WithURL(u string) StaticRequestOption {
	uobj, err := url.Parse(u)
	opts := []StaticRequestOption{}
	if err == nil {
		if uobj.Scheme != "" {
			opts = append(opts, WithScheme(uobj.Scheme))
		}
		if uobj.Host != "" {
			opts = append(opts, WithHostPort(uobj.Host))
		}
		// todo:p0: implement
		// if uobj.Path != "" {
		// 	return error in this case as static option shouldn't support sending non-empty path
		// }
		// if uobj.RawQuery != "" {
		// 	return error in this case as static option shouldn't support sending non-empty raw query
		// }
		// if uobj.Fragment != "" {
		// 	return error in this case as static option shouldn't support sending non-empty fragment
		// }
	}

	return func(c StaticRequestConfig) (StaticRequestConfig, error) {
		if err != nil {
			return c, err
		}
		for _, opt := range opts {
			if c, err = opt(c); err != nil {
				return c, err
			}
		}
		return c, nil
	}
}

type RetryOnCode string

const (
	RetryOn5xx                     RetryOnCode = "5xx"
	RetryOnGatewayError            RetryOnCode = "gateway-error"
	RetryOnReset                   RetryOnCode = "reset"
	RetryOnConnectFailure          RetryOnCode = "connect-failure"
	RetryOnEnvoyRatelimited        RetryOnCode = "envoy-ratelimited"
	RetryOnRetriable4xx            RetryOnCode = "retriable-4xx"
	RetryOnRefusedStream           RetryOnCode = "refused-stream"
	RetryOnHTTP3PostConnectFailure RetryOnCode = "http3-post-connect-failure"

	// reserved for internal use if RetriableStatusCodes or RetriableHeaders fields are set
	retryOnRetriableStatusCodes RetryOnCode = "retriable-status-codes"
	retryOnRetriableHeaders     RetryOnCode = "retriable-headers"
)

type EnvoyRetryPolicy struct {
	MaxRetries           uint16
	TotalTimeout         time.Duration
	PerTryTimeout        time.Duration
	RetryOn              []RetryOnCode
	RetriableStatusCodes []uint16
	RetriableHeaders     []string
}

func DefaultEnvoyRetryPolicy() EnvoyRetryPolicy {
	retryOn := []RetryOnCode{
		RetryOnConnectFailure,
		RetryOnEnvoyRatelimited,
		RetryOnRetriable4xx,
		RetryOnRefusedStream,
		retryOnRetriableStatusCodes,
	}
	return EnvoyRetryPolicy{
		MaxRetries:    3,
		TotalTimeout:  31 * time.Second,
		PerTryTimeout: 10 * time.Second,
		RetryOn:       retryOn,
		RetriableStatusCodes: []uint16{
			http.StatusTooManyRequests,    // 429
			http.StatusBadGateway,         // 502
			http.StatusServiceUnavailable, // 503
		},
	}
}

func WithDefaultEnvoyRetryPolicy() StaticRequestOption {
	return WithEnvoyRetryPolicy(DefaultEnvoyRetryPolicy())
}

// WithEnvoyRetryPolicy sets the retry policy for the request.
// It validates the retry policy, and returns an error if the retry policy is invalid.
// if max retries is set or non-zero then total timeout and per try timeout is compulsory.
func WithEnvoyRetryPolicy(retryPolicy EnvoyRetryPolicy) StaticRequestOption {
	return func(c StaticRequestConfig) (StaticRequestConfig, error) {
		err := validateEnvoyRetryPolicy(retryPolicy)
		if err != nil {
			return c, err
		}
		c = c.Clone()
		if c.Headers == nil {
			c.Headers = make(http.Header)
		}
		c.Headers.Set("x-envoy-max-retries", strconv.FormatUint(uint64(retryPolicy.MaxRetries), 10))
		if retryPolicy.TotalTimeout > 0 {
			c.Headers.Set("x-envoy-upstream-rq-timeout-ms", strconv.FormatUint(uint64(retryPolicy.TotalTimeout/time.Millisecond), 10))
		}
		if retryPolicy.PerTryTimeout > 0 {
			c.Headers.Set("x-envoy-upstream-rq-per-try-timeout-ms", strconv.FormatUint(uint64(retryPolicy.PerTryTimeout/time.Millisecond), 10))
		}
		if len(retryPolicy.RetriableStatusCodes) > 0 {
			c.Headers.Set("x-envoy-retriable-status-codes", strings.Join(integers.ToStrings(retryPolicy.RetriableStatusCodes), ","))
		}
		if len(retryPolicy.RetriableHeaders) > 0 {
			c.Headers.Set("x-envoy-retriable-headers", gstrings.Join(retryPolicy.RetriableHeaders, ","))
		}
		if len(retryPolicy.RetryOn) > 0 {
			c.Headers.Set("x-envoy-retry-on", gstrings.Join(retryPolicy.RetryOn, ","))
		}
		return c, nil
	}
}

func validateEnvoyRetryPolicy(rp EnvoyRetryPolicy) error {
	// check if max retries is set or non-zero then total timeout and per try timeout is compulsory.
	if rp.MaxRetries > 0 && (rp.TotalTimeout == 0 || rp.PerTryTimeout == 0) {
		return fmt.Errorf("maxRetries is set but totalTimeout(%d) or perTryTimeout(%d) is not set", rp.TotalTimeout, rp.PerTryTimeout)
	}
	// validate that total timeout is greater than per try timeout
	if rp.TotalTimeout < rp.PerTryTimeout {
		return fmt.Errorf("totalTimeout(%d) is less than perTryTimeout(%d)", rp.TotalTimeout, rp.PerTryTimeout)
	}
	// validate that total timeout is greater than per try timeout+1 * max retries is less than total timeout
	if rp.TotalTimeout < (rp.PerTryTimeout+1)*time.Duration(rp.MaxRetries) {
		return fmt.Errorf("totalTimeout(%d) is less than (perTryTimeout(%d) + 1) * maxRetries(%d)", rp.TotalTimeout, rp.PerTryTimeout, rp.MaxRetries)
	}

	for _, statusCode := range rp.RetriableStatusCodes {
		if statusCode < 100 || statusCode > 599 {
			return fmt.Errorf("invalid status code %d passed in RetriableStatusCodes", statusCode)
		}
	}
	for _, header := range rp.RetriableHeaders {
		if len(header) == 0 {
			return fmt.Errorf("empty header passed in RetriableHeaders")
		}
	}

	// todo: setup validation packs where users can opt into default validation pack or extend the default validation pack with their own custom validation packs
	// if rp.MaxRetries > 10 {
	// if rp.TotalTimeout > 60*time.Second {
	// if rp.PerTryTimeout < 0 {
	return nil
}

// useful for setting static headers that will be useful for setting secrets or any other custom
// authentication headers that apply for all requests
func WithStaticHeaders(headers map[string]string) StaticRequestOption {
	return func(c StaticRequestConfig) (StaticRequestConfig, error) {
		err := validateStaticHeaders(headers)
		if err != nil {
			return c, err
		}
		for k, v := range headers {
			c.Headers.Set(k, v)
		}
		return c, nil
	}
}

func WithStaticHeader(key, value string) StaticRequestOption {
	return func(c StaticRequestConfig) (StaticRequestConfig, error) {
		if key == "" || value == "" {
			return c, errors.New("key or value is empty")
		}
		c.Headers.Set(key, value)
		return c, nil
	}
}

func validateStaticHeaders(headers map[string]string) error {
	if headers == nil {
		return errors.New("headers is nil")
	}
	return nil
}

// runtime request configuration options

// useful for setting runtime headers that are request specific. (use proper tracing middlewares for setting these)
// only for exceptional cases where you want to set a header that couldn't be abstracted into a middleware use this
func WithRuntimeHeaders(headers map[string]string) RuntimeRequestOption {
	return func(c RuntimeRequestConfig) (RuntimeRequestConfig, error) {
		err := validateRuntimeHeaders(headers)
		if err != nil {
			return c, err
		}
		for k, v := range headers {
			c.Headers.Set(k, v)
		}
		return c, nil
	}
}

func WithRuntimeHeader(key, value string) RuntimeRequestOption {
	return func(c RuntimeRequestConfig) (RuntimeRequestConfig, error) {
		if key == "" || value == "" {
			return c, errors.New("key or value is empty")
		}
		c.Headers.Set(key, value)
		return c, nil
	}
}

func validateRuntimeHeaders(headers map[string]string) error {
	if headers == nil {
		return errors.New("headers is nil")
	}
	return nil
}

func WithGoStdClient(client *http.Client) ClientOption {
	return func(c ClientConfig) (ClientConfig, error) {
		c.Client = client
		return c, nil
	}
}

func WithRequestJSON(body interface{}) RuntimeRequestOption {
	return func(c RuntimeRequestConfig) (RuntimeRequestConfig, error) {
		// todo: allow custom json encoders in the static config
		bts, err := json.Marshal(body)
		if err != nil {
			return c, err
		}
		c.Body = bts
		if c.Headers == nil {
			c.Headers = make(http.Header)
		}
		c.Headers.Set("Content-Type", "application/json")
		return c, nil
	}
}

func WithResponseJSON(response interface{}) RuntimeRequestOption {
	return func(c RuntimeRequestConfig) (RuntimeRequestConfig, error) {
		c.ResponseJSON = response
		return c, nil
	}
}

func WithResponseBody(response io.Writer) RuntimeRequestOption {
	return func(c RuntimeRequestConfig) (RuntimeRequestConfig, error) {
		c.ResponseBody = response
		return c, nil
	}
}

type HttpMethod string

const (
	MethodGet     HttpMethod = http.MethodGet
	MethodHead    HttpMethod = http.MethodHead
	MethodPost    HttpMethod = http.MethodPost
	MethodPut     HttpMethod = http.MethodPut
	MethodPatch   HttpMethod = http.MethodPatch
	MethodDelete  HttpMethod = http.MethodDelete
	MethodConnect HttpMethod = http.MethodConnect
	MethodOptions HttpMethod = http.MethodOptions
	MethodTrace   HttpMethod = http.MethodTrace
)

func WithMethod(method HttpMethod) RuntimeRequestOption {
	return func(c RuntimeRequestConfig) (RuntimeRequestConfig, error) {
		c.Method = method
		return c, nil
	}
}

func WithPath(path string) RuntimeRequestOption {
	return func(c RuntimeRequestConfig) (RuntimeRequestConfig, error) {
		c.Path = path
		return c, nil
	}
}

func WithQueryParam(key, value string) RuntimeRequestOption {
	return func(c RuntimeRequestConfig) (RuntimeRequestConfig, error) {
		if key == "" || value == "" {
			return c, errors.New("key or value is empty")
		}
		if c.Query == nil {
			c.Query = url.Values{}
		}
		c.Query.Set(key, value)
		return c, nil
	}
}

func WithQueryParams(params map[string]string) RuntimeRequestOption {
	return func(c RuntimeRequestConfig) (RuntimeRequestConfig, error) {
		err := validateQueryParams(params)
		if err != nil {
			return c, err
		}
		if c.Query == nil {
			c.Query = url.Values{}
		}
		for k, v := range params {
			c.Query.Set(k, v)
		}
		return c, nil
	}
}

func validateQueryParams(params map[string]string) error {
	if params == nil {
		return errors.New("params is nil")
	}
	return nil
}
