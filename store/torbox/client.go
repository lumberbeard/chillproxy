package torbox

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/store"
)

// Global request counter for debugging key invalidation
var torboxRequestSeq atomic.Int64

var DefaultHTTPClient = config.DefaultHTTPClient

type APIClientConfig struct {
	BaseURL    string // default: https://api.torbox.app
	APIKey     string
	HTTPClient *http.Client
	UserAgent  string
}

type APIClient struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
	apiKey     string
	agent      string
	reqQuery   func(query *url.Values, params request.Context)
	reqHeader  func(query *http.Header, params request.Context)
}

func NewAPIClient(conf *APIClientConfig) *APIClient {
	if conf.UserAgent == "" {
		conf.UserAgent = "stremthru"
	}

	if conf.BaseURL == "" {
		conf.BaseURL = "https://api.torbox.app"
	}

	if conf.HTTPClient == nil {
		conf.HTTPClient = DefaultHTTPClient
	}

	c := &APIClient{}

	baseUrl, err := url.Parse(conf.BaseURL)
	if err != nil {
		panic(err)
	}

	c.BaseURL = baseUrl
	c.HTTPClient = conf.HTTPClient
	c.apiKey = conf.APIKey
	c.agent = conf.UserAgent

	c.reqQuery = func(query *url.Values, params request.Context) {}

	c.reqHeader = func(header *http.Header, params request.Context) {
		// TorBox API expects "Bearer <api_key>" format
		apiKey := params.GetAPIKey(c.apiKey)
		authValue := "Bearer " + apiKey
		header.Add("Authorization", authValue)
		header.Add("User-Agent", c.agent)
	}

	return c
}

type Ctx = request.Ctx

func keyPreview(key string) string {
	if len(key) > 8 {
		return key[:8]
	}
	return key
}

// formatResponseHeaders extracts interesting headers from a TorBox API response
func formatResponseHeaders(res *http.Response) string {
	if res == nil {
		return "nil"
	}
	interesting := []string{}
	for name, values := range res.Header {
		lower := strings.ToLower(name)
		// Capture rate limit, retry, auth, and any torbox-specific headers
		if strings.Contains(lower, "rate") ||
			strings.Contains(lower, "limit") ||
			strings.Contains(lower, "retry") ||
			strings.Contains(lower, "cooldown") ||
			strings.Contains(lower, "x-") ||
			strings.Contains(lower, "cf-") {
			interesting = append(interesting, fmt.Sprintf("%s=%s", name, strings.Join(values, ",")))
		}
	}
	if len(interesting) == 0 {
		return "none"
	}
	return strings.Join(interesting, "; ")
}

func (c APIClient) Request(method, path string, params request.Context, v ResponseEnvelop) (*http.Response, error) {
	if params == nil {
		params = &Ctx{}
	}

	// Capture the API key being used for this request
	reqAPIKey := params.GetAPIKey(c.apiKey)
	seq := torboxRequestSeq.Add(1)

	start := time.Now()
	req, err := params.NewRequest(c.BaseURL, method, path, c.reqHeader, c.reqQuery)
	if err != nil {
		log.Error("🔴 TORBOX REQUEST FAILED (create)",
			"seq", seq,
			"method", method,
			"path", path,
			"keyPreview", keyPreview(reqAPIKey),
			"error", err)
		error := core.NewStoreError("failed to create request")
		error.StoreName = string(store.StoreNameTorBox)
		error.Cause = err
		return nil, error
	}
	res, err := params.DoRequest(c.HTTPClient, req)
	duration := time.Since(start)
	err = request.ProcessResponseBody(res, err, v)
	if err != nil {
		statusCode := 0
		headers := "nil"
		if res != nil {
			statusCode = res.StatusCode
			headers = formatResponseHeaders(res)
		}
		log.Warn("🔴 TORBOX API ERROR",
			"seq", seq,
			"method", method,
			"path", path,
			"status", statusCode,
			"keyPreview", keyPreview(reqAPIKey),
			"duration", duration,
			"respHeaders", headers,
			"error", err)
		err := UpstreamErrorWithCause(err)
		err.InjectReq(req)
		if res != nil {
			err.StatusCode = res.StatusCode
		}
		if err.StatusCode <= http.StatusBadRequest {
			err.StatusCode = http.StatusBadRequest
		}
		err.Pack(req)
		return res, err
	}

	log.Info("🟢 TORBOX API OK",
		"seq", seq,
		"method", method,
		"path", path,
		"status", res.StatusCode,
		"keyPreview", keyPreview(reqAPIKey),
		"duration", duration,
		"respHeaders", formatResponseHeaders(res))

	return res, nil
}
