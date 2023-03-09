package httpclient

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/wI2L/jsondiff"
)

func TestWithInsecureSkipVerify(t *testing.T) {
	// given
	// when
	opt := WithInsecureSkipVerify(true)

	cfg := ClientConfig{}
	_, err := opt(cfg)
	if err == nil {
		t.Errorf("expected error to be set as this option only works along with the transport")
		return
	}

	cfg2 := ClientConfig{
		Transport: buildDefaultTransport(),
	}
	cfg2, err = opt(cfg2)
	if err != nil {
		t.Errorf("expected error to be nil, got %v", err)
		return
	}
	if cfg2.Transport.TLSClientConfig.InsecureSkipVerify != true {
		t.Errorf("expected %v, got %v", true, cfg2.Transport.TLSClientConfig.InsecureSkipVerify)
		return
	}

	cfg3 := ClientConfig{
		Transport: buildDefaultTransport(),
	}
	opt2 := WithInsecureSkipVerify(false)
	cfg3, err = opt2(cfg3)
	if err != nil {
		t.Errorf("expected error to be nil, got %v", err)
		return
	}
	if cfg3.Transport.TLSClientConfig.InsecureSkipVerify != false {
		t.Errorf("expected %v, got %v", false, cfg3.Transport.TLSClientConfig.InsecureSkipVerify)
		return
	}
}

func TestWithScheme(t *testing.T) {
	// given
	// when
	opt := WithScheme("http")

	cfg := StaticRequestConfig{}
	cfg, err := opt(cfg)
	if err != nil {
		t.Errorf("expected error to be nil, got %v", err)
		return
	}
	if cfg.Scheme != "http" {
		t.Errorf("expected %v, got %v", "http", cfg.Scheme)
		return
	}

	opt2 := WithScheme("https")
	cfg2 := StaticRequestConfig{}
	cfg2, err = opt2(cfg2)
	if err != nil {
		t.Errorf("expected error to be nil, got %v", err)
		return
	}
	if cfg2.Scheme != "https" {
		t.Errorf("expected %v, got %v", "https", cfg2.Scheme)
		return
	}

	opt3 := WithScheme("ftp")
	cfg3 := StaticRequestConfig{}
	_, err = opt3(cfg3)
	if err == nil {
		t.Errorf("expected error to be set as scheme is not http or https")
		return
	}

	// test empty scheme
	opt4 := WithScheme("")
	cfg4 := StaticRequestConfig{}
	_, err = opt4(cfg4)
	if err == nil {
		t.Errorf("expected error to be set as scheme is empty")
		return
	}
}

func TestWithHostPort(t *testing.T) {
	// given
	// when
	opt := WithHostPort("localhost:8080")

	cfg := StaticRequestConfig{}
	cfg, err := opt(cfg)
	if err != nil {
		t.Errorf("expected error to be nil, got %v", err)
		return
	}
	if cfg.Host != "localhost:8080" {
		t.Errorf("expected %v, got %v", "localhost:8080", cfg.Host)
		return
	}

	// test empty host
	opt2 := WithHostPort("")
	cfg2 := StaticRequestConfig{}
	_, err = opt2(cfg2)
	if err == nil {
		t.Errorf("expected error to be set as host is empty")
		return
	}

	// test without port as the port is mandatory
	opt3 := WithHostPort("localhost")
	cfg3 := StaticRequestConfig{}
	_, err = opt3(cfg3)
	if err == nil {
		t.Errorf("expected error to be set as port is missing")
		return
	}

	// test with invalid port
	opt4 := WithHostPort("localhost:abc")
	cfg4 := StaticRequestConfig{}
	_, err = opt4(cfg4)
	if err == nil {
		t.Errorf("expected error to be set as port is invalid")
		return
	}

	// test with invalid host
	opt5 := WithHostPort("localhost:8080:8080")
	cfg5 := StaticRequestConfig{}
	_, err = opt5(cfg5)
	if err == nil {
		t.Errorf("expected error to be set as host is invalid")
		return
	}

	// test with empty host but with non-empty port
	opt6 := WithHostPort(":8080")
	cfg6 := StaticRequestConfig{}
	_, err = opt6(cfg6)
	if err == nil {
		t.Errorf("expected error to be set as host is empty")
		return
	}
}

func TestWithHostHeader(t *testing.T) {
	// given
	// when
	opt := WithHostHeader("localhost")

	cfg := StaticRequestConfig{}
	cfg, err := opt(cfg)
	if err != nil {
		t.Errorf("expected error to be nil, got %v", err)
		return
	}
	hostHeaderSet := false
	for k, v := range cfg.Headers {
		if k != "Host" {
			continue
		}
		hostHeaderSet = true
		if v[0] != "localhost" {
			t.Errorf("expected %v, got %v", "localhost", v[0])
			return
		}
	}
	if !hostHeaderSet {
		t.Errorf("expected host header to be set")
		return
	}

	// test empty host
	opt2 := WithHostHeader("")
	cfg2 := StaticRequestConfig{}
	_, err = opt2(cfg2)
	if err == nil {
		t.Errorf("expected error to be set as host is empty")
		return
	}
}

func TestDefaultEnvoyRetryPolicy(t *testing.T) {
	// since the default retry policy is pretty basic, we just need to test that it is set to some non nil value
	retryPolicy := DefaultEnvoyRetryPolicy()
	emptyObj := EnvoyRetryPolicy{}
	if reflect.DeepEqual(retryPolicy, emptyObj) {
		t.Errorf("expected retry policy to be set")
		return
	}
}

func TestWithEnvoyRetryPolicy(t *testing.T) {
	// test with just max retries set to 0
	opt := WithEnvoyRetryPolicy(EnvoyRetryPolicy{
		MaxRetries: 0,
	})

	cfg := StaticRequestConfig{}
	cfg, err := opt(cfg)
	if err != nil {
		t.Errorf("expected error to be nil, got %v", err)
		return
	}
	if v := cfg.Headers.Values("x-envoy-max-retries"); len(v) == 0 {
		t.Errorf("expected x-envoy-max-retries header to be set")
		return
	} else if len(v) > 1 {
		t.Errorf("expected only one value for x-envoy-max-retries header, got %v", len(v))
		return
	} else if v[0] != "0" {
		t.Errorf("expected %v, got %v", "0", v[0])
		return
	}

	// test with just max retries set to 1
	opt2 := WithEnvoyRetryPolicy(EnvoyRetryPolicy{
		MaxRetries: 1,
	})
	cfg2 := StaticRequestConfig{}
	_, err = opt2(cfg2)
	if err == nil {
		t.Errorf("expected error to be set as retry policy is invalid")
		return
	}

	// test with invalid total timeout
	opt3 := WithEnvoyRetryPolicy(EnvoyRetryPolicy{
		MaxRetries:    0,
		PerTryTimeout: 2 * time.Second,
		TotalTimeout:  time.Second,
	})
	cfg3 := StaticRequestConfig{}
	_, err = opt3(cfg3)
	if err == nil {
		t.Errorf("expected error to be set as retry policy is invalid")
		return
	}

	// test with max retries set but not per try timeout and total timeout
	opt4 := WithEnvoyRetryPolicy(EnvoyRetryPolicy{
		MaxRetries: 1,
	})
	cfg4 := StaticRequestConfig{}
	_, err = opt4(cfg4)
	if err == nil {
		t.Errorf("expected error to be set as retry policy is invalid")
		return
	}

	// test with invalid retryiable status code
	opts1 := []StaticRequestOption{
		WithEnvoyRetryPolicy(EnvoyRetryPolicy{
			MaxRetries:    0,
			PerTryTimeout: 2 * time.Second,
			TotalTimeout:  2 * time.Second,
			RetriableStatusCodes: []uint16{
				433,
				0,
			},
		}),
		WithEnvoyRetryPolicy(EnvoyRetryPolicy{
			MaxRetries:    0,
			PerTryTimeout: 2 * time.Second,
			TotalTimeout:  2 * time.Second,
			RetriableStatusCodes: []uint16{
				433,
				93,
			},
		}),
	}
	for idx, opt := range opts1 {
		t.Run("test with invalid retryiable status code", func(t *testing.T) {
			cfg := StaticRequestConfig{}
			_, err := opt(cfg)
			if err == nil {
				t.Errorf("expected error to be set as retry policy is invalid for test case %v", idx)
				return
			}
			// check if the error is the one we expect (users shouldn't depend on this as a public contract)
			if !strings.Contains(err.Error(), "invalid status code") {
				t.Errorf("expected error to be set as retry policy is invalid for test case %v but got %v", idx, err.Error())
				return
			}
		})
	}

	// test with valid retryiable status code
	opts2 := []struct {
		opt      StaticRequestOption
		expected string
	}{
		{
			opt: WithEnvoyRetryPolicy(EnvoyRetryPolicy{
				MaxRetries:    0,
				PerTryTimeout: 2 * time.Second,
				TotalTimeout:  2 * time.Second,
				RetriableStatusCodes: []uint16{
					433,
					567,
				},
			}),
			expected: "433,567",
		},
		{
			opt: WithEnvoyRetryPolicy(EnvoyRetryPolicy{
				MaxRetries:    0,
				PerTryTimeout: 2 * time.Second,
				TotalTimeout:  2 * time.Second,
				RetriableStatusCodes: []uint16{
					503,
					429,
				},
			}),
			expected: "503,429",
		},
	}
	for idx, testcase := range opts2 {
		t.Run("test with valid retryiable status code", func(t *testing.T) {
			cfg := StaticRequestConfig{}
			cfg, err := testcase.opt(cfg)
			if err != nil {
				t.Errorf("expected error to be nil for test case %v", idx)
				return
			}
			if v := cfg.Headers.Values("x-envoy-retriable-status-codes"); len(v) == 0 {
				t.Errorf("expected x-envoy-retriable-status-codes header to be set for test case %v", idx)
				return
			} else if len(v) > 1 {
				t.Errorf("expected only one value for x-envoy-retriable-status-codes header, got %v for test case %v", len(v), idx)
				return
			} else if v[0] != testcase.expected {
				t.Errorf("expected %v, got %v for test case %v", testcase.expected, v[0], idx)
				return
			}
		})
	}

	// test with invalid retriable header
	opt5 := WithEnvoyRetryPolicy(EnvoyRetryPolicy{
		MaxRetries:    0,
		PerTryTimeout: 2 * time.Second,
		TotalTimeout:  2 * time.Second,
		RetriableHeaders: []string{
			"",
		},
	})
	cfg5 := StaticRequestConfig{}
	_, err = opt5(cfg5)
	if err == nil {
		t.Errorf("expected error to be set as retry policy is invalid")
		return
	}
}

// runtime options test
func TestWithJSONBody(t *testing.T) {
	// make a test object to send and receive
	type testStruct struct {
		Name      string `json:"name"`
		Something string `json:"something"`
	}
	ts := testStruct{
		Name:      "test",
		Something: "something2",
	}

	// launch http test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		expectedInput, err := json.Marshal(ts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		patch, err := jsondiff.CompareJSON(b, expectedInput)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		if len(patch) != 0 {
			t.Errorf("expected no difference between the two jsons, got %v", patch)
			return
		}
	}))
	defer server.Close()

	opt := WithRequestJSON(ts)

	hc, err := NewHTTPClient(ConfigOptions{
		ClientOptions: []ClientOption{
			WithGoStdClient(server.Client()),
		},
		StaticRequestOptions: []StaticRequestOption{
			WithURL(server.URL),
		},
		RuntimeRequestOptions: []RuntimeRequestOption{opt},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	tobj1 := struct{}{}

	err = hc.Send(
		context.Background(),
		WithResponseJSON(tobj1),
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
}
