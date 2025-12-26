package torbox

import (
	"net/http"
	"net/url"

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

		// DEBUG: Log what's being sent (first 50 chars only for security)
		truncated := authValue
		if len(authValue) > 50 {
			truncated = authValue[:50] + "..."
		}
		// Note: Using fmt.Println since we don't have logger here
		// This will show in Docker logs
		if len(apiKey) == 0 {
			println("🚨 TORBOX: Empty API key!")
		} else if len(apiKey) != 36 {
			println("🚨 TORBOX: Invalid API key length:", len(apiKey), "expected 36")
		} else {
			println("✅ TORBOX: Authorization header set:", truncated)
		}
	}

	return c
}

type Ctx = request.Ctx

func (c APIClient) Request(method, path string, params request.Context, v ResponseEnvelop) (*http.Response, error) {
	if params == nil {
		params = &Ctx{}
	}
	req, err := params.NewRequest(c.BaseURL, method, path, c.reqHeader, c.reqQuery)
	if err != nil {
		error := core.NewStoreError("failed to create request")
		error.StoreName = string(store.StoreNameTorBox)
		error.Cause = err
		return nil, error
	}
	res, err := params.DoRequest(c.HTTPClient, req)
	err = request.ProcessResponseBody(res, err, v)
	if err != nil {
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
	return res, nil
}
