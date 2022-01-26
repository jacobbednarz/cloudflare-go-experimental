package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

// RouteType is a custom type for denoting the ownership level of a resource.
type RouteType string

const (
	APIHostname string = "api.cloudflare.com"
	APIBasePath string = "/client/v4"

	AccountRouteType RouteType = "accounts"
	ZoneRouteType    RouteType = "zones"

	testAccountID    string = "01a7362d577a6c3019a474fd6f485823"
	testZoneID       string = "d56084adb405e0b7e32c52321bf07be6"
	testCertPackUUID string = "a77f8bd7-3b47-46b4-a6f1-75cf98109948"
)

var (
	Key            string
	Email          string
	UserServiceKey string
	Token          string
	Version        string = "dev"
)

type RetryPolicy struct {
	MaxRetries    int
	MinRetryDelay time.Duration
	MaxRetryDelay time.Duration
}

type Logger interface {
	Printf(format string, v ...interface{})
}

type APIClient struct {
	ClientParams
}

type ClientParams struct {
	Key            string
	Email          string
	UserServiceKey string
	Token          string
	Hostname       string
	BasePath       string
	UserAgent      string
	Headers        http.Header
	HTTPClient     *http.Client
	RateLimiter    *rate.Limiter
	RetryPolicy    RetryPolicy
	Logger         Logger
}

// ResponseInfo contains a code and message returned by the API as errors or
// informational messages inside the response.
type ResponseInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Response is a template. There will also be a result struct. There will be a
// unique response type for each response, which will include this type.
type Response struct {
	Success  bool           `json:"success"`
	Errors   []ResponseInfo `json:"errors"`
	Messages []ResponseInfo `json:"messages"`
}

// ResultInfoCursors contains information about cursors.
type ResultInfoCursors struct {
	Before string `json:"before"`
	After  string `json:"after"`
}

// ResultInfo contains metadata about the Response.
type ResultInfo struct {
	Page       int               `json:"page"`
	PerPage    int               `json:"per_page"`
	TotalPages int               `json:"total_pages"`
	Count      int               `json:"count"`
	Total      int               `json:"total_count"`
	Cursor     string            `json:"cursor"`
	Cursors    ResultInfoCursors `json:"cursors"`
}

func (api *APIClient) Call(ctx context.Context, method, path string, payload interface{}) ([]byte, error) {
	return api.makeRequest(ctx, method, path, payload, nil)
}

func (api *APIClient) CallWithHeaders(ctx context.Context, method, path string, payload interface{}, headers http.Header) ([]byte, error) {
	return api.makeRequest(ctx, method, path, payload, headers)
}

// New creates a new instance of the API client by merging ClientParams with the
// default values.
func New(config *ClientParams) (*APIClient, error) {
	silentLogger := log.New(ioutil.Discard, "", log.LstdFlags)

	defaultAPI := &ClientParams{
		Hostname:    APIHostname,
		BasePath:    APIBasePath,
		UserAgent:   "cloudflare-go/" + Version,
		HTTPClient:  http.DefaultClient,
		Headers:     make(http.Header),
		RateLimiter: rate.NewLimiter(rate.Limit(4), 1), // 4rps equates to default api limit (1200 req/5 min)
		RetryPolicy: RetryPolicy{
			MaxRetries:    3,
			MinRetryDelay: time.Duration(1) * time.Second,
			MaxRetryDelay: time.Duration(30) * time.Second,
		},
		Logger: silentLogger,
	}

	if err := mergo.Merge(config, defaultAPI); err != nil {
		return nil, fmt.Errorf("failed to merge API configuration with defaults: %w", err)
	}

	// Take the global values and override them in the instantiated client if they
	// exist. This ensures a global authentication method has precedence over a
	// local one.
	if Key != "" {
		config.Key = Key
		config.Email = Email
	}

	if Token != "" {
		config.Token = Token
	}

	if UserServiceKey != "" {
		config.UserServiceKey = UserServiceKey
	}

	return &APIClient{*config}, nil
}

func (api *APIClient) makeRequest(ctx context.Context, method, uri string, params interface{}, headers http.Header) ([]byte, error) {
	var reqBody io.Reader
	var err error

	if params != nil {
		if r, ok := params.(io.Reader); ok {
			reqBody = r
		} else if paramBytes, ok := params.([]byte); ok {
			reqBody = bytes.NewReader(paramBytes)
		} else {
			var jsonBody []byte
			jsonBody, err = json.Marshal(params)
			if err != nil {
				return nil, errors.Wrap(err, "error marshalling params to JSON")
			}
			reqBody = bytes.NewReader(jsonBody)
		}
	}

	var resp *http.Response
	var respErr error
	var respBody []byte
	for i := 0; i <= api.RetryPolicy.MaxRetries; i++ {
		if i > 0 {
			// expect the backoff introduced here on errored requests to dominate the effect of rate limiting
			// don't need a random component here as the rate limiter should do something similar
			// nb time duration could truncate an arbitrary float. Since our inputs are all ints, we should be ok
			sleepDuration := time.Duration(math.Pow(2, float64(i-1)) * float64(api.RetryPolicy.MinRetryDelay))

			if sleepDuration > api.RetryPolicy.MaxRetryDelay {
				sleepDuration = api.RetryPolicy.MaxRetryDelay
			}
			// useful to do some simple logging here, maybe introduce levels later
			api.Logger.Printf("sleeping %s before retry attempt number %d for request %s %s", sleepDuration.String(), i, method, uri)

			select {
			case <-time.After(sleepDuration):
			case <-ctx.Done():
				return nil, fmt.Errorf("operation aborted during backoff: %w", ctx.Err())
			}
		}

		err = api.RateLimiter.Wait(ctx)
		if err != nil {
			return nil, fmt.Errorf("error caused by request rate limiting: %w", err)
		}

		resp, respErr = api.request(ctx, method, uri, reqBody, headers)

		// retry if the server is rate limiting us or if it failed
		// assumes server operations are rolled back on failure
		if respErr != nil || resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			// if we got a valid http response, try to read body so we can reuse the connection
			// see https://golang.org/pkg/net/http/#Client.Do
			if respErr == nil {
				respBody, err = ioutil.ReadAll(resp.Body)
				resp.Body.Close()

				respErr = errors.Wrap(err, "could not read response body")

				api.Logger.Printf("Request: %s %s got an error response %d: %s\n", method, uri, resp.StatusCode,
					strings.Replace(strings.Replace(string(respBody), "\n", "", -1), "\t", "", -1))
			} else {
				api.Logger.Printf("Error performing request: %s %s : %s \n", method, uri, respErr.Error())
			}
			continue
		} else {
			respBody, err = ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
			if err != nil {
				return nil, errors.Wrap(err, "could not read response body")
			}
			break
		}
	}
	if respErr != nil {
		return nil, respErr
	}

	if resp.StatusCode >= http.StatusBadRequest {
		if strings.HasSuffix(resp.Request.URL.Path, "/filters/validate-expr") {
			return nil, errors.Errorf("%s", respBody)
		}

		if resp.StatusCode > http.StatusInternalServerError {
			return nil, errors.Errorf("HTTP status %d: service failure", resp.StatusCode)
		}

		errBody := &Response{}
		err = json.Unmarshal(respBody, &errBody)
		if err != nil {
			return nil, errors.Wrap(err, errUnmarshalErrorBody)
		}

		return nil, &APIRequestError{
			StatusCode: resp.StatusCode,
			Errors:     errBody.Errors,
			RayID:      resp.Header.Get("cf-ray"),
		}
	}

	return respBody, nil
}

// request makes a HTTP request to the given API endpoint, returning the raw
// *http.Response, or an error if one occurred. The caller is responsible for
// closing the response body.
func (api *APIClient) request(ctx context.Context, method, uri string, reqBody io.Reader, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, api.GetBaseURL()+uri, reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "HTTP request creation failed")
	}

	combinedHeaders := make(http.Header)
	copyHeader(combinedHeaders, api.Headers)
	copyHeader(combinedHeaders, headers)
	req.Header = combinedHeaders

	if api.Key == "" && api.Email == "" && api.Token == "" && api.UserServiceKey == "" {
		return nil, errors.New("no user credentials provided")
	}

	if api.Key != "" {
		req.Header.Set("X-Auth-Key", api.Key)
		req.Header.Set("X-Auth-Email", api.Email)
	}

	if api.UserServiceKey != "" {
		req.Header.Set("X-Auth-User-Service-Key", api.UserServiceKey)
	}

	if api.Token != "" {
		req.Header.Set("Authorization", "Bearer "+api.Token)
	}

	if api.UserAgent != "" {
		req.Header.Set("User-Agent", api.UserAgent)
	}

	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "HTTP request failed")
	}

	return resp, nil
}

func (api *APIClient) GetBaseURL() string {
	return "https://" + api.Hostname + api.BasePath
}

// copyHeader copies all headers for `source` and sets them on `target`.
// based on https://godoc.org/github.com/golang/gddo/httputil/header#Copy
func copyHeader(target, source http.Header) {
	for k, vs := range source {
		target[k] = vs
	}
}

func isHTTPWriteMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

func normalizeURL(url string) string {
	// All paths include a leading slash, so to keep logs pretty, trim a
	// trailing slash on the URL.
	url = strings.TrimSuffix(url, "/")

	// For a long time we had the `/v1` suffix as part of a configured URL
	// rather than in the per-package URLs throughout the library. Continue
	// to support this for the time being by stripping one that's been
	// passed for better backwards compatibility.
	url = strings.TrimSuffix(url, "/v1")

	return url
}
