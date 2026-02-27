package torbox

import (
	"net/http"
	"net/url"
	"time"

	"github.com/MunifTanjim/stremthru/core"
	"github.com/MunifTanjim/stremthru/internal/config"
	"github.com/MunifTanjim/stremthru/internal/request"
	"github.com/MunifTanjim/stremthru/store"
)

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

func (c APIClient) Request(method, path string, params request.Context, v ResponseEnvelop) (*http.Response, error) {
	if params == nil {
		params = &Ctx{}
	}

	// Capture the API key being used for this request
	reqAPIKey := params.GetAPIKey(c.apiKey)

	start := time.Now()
	req, err := params.NewRequest(c.BaseURL, method, path, c.reqHeader, c.reqQuery)
	if err != nil {
		log.Error("🔴 TORBOX REQUEST FAILED (create)",
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
		if res != nil {
			statusCode = res.StatusCode
		}
		log.Warn("🔴 TORBOX API ERROR",
			"method", method,
			"path", path,
			"status", statusCode,
			"keyPreview", keyPreview(reqAPIKey),
			"duration", duration,
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
		"method", method,
		"path", path,
		"status", res.StatusCode,
		"keyPreview", keyPreview(reqAPIKey),
		"duration", duration)

	return res, nil
}
