package utils

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
)

type HttpClientWrapper struct {
	client *http.Client
	Domain string
}

func NewHttpClientWrapper(domain string) *HttpClientWrapper {
	return &HttpClientWrapper{
		Domain: domain,
		client: http.DefaultClient,
	}
}
func (wrapper *HttpClientWrapper) Get(api string, queryParams url.Values) (*http.Response, error) {
	request, err := wrapper.assembleRequest(http.MethodGet, api, queryParams, nil)
	if err != nil {
		return nil, err
	}
	return wrapper.client.Do(request)
}
func (wrapper *HttpClientWrapper) Post(api string, queryParams url.Values, body *[]byte) (*http.Response, error) {
	request, err := wrapper.assembleRequest(http.MethodPost, api, queryParams, bytes.NewReader(*body))
	if err != nil {
		return nil, err
	}
	return wrapper.client.Do(request)
}
func (wrapper *HttpClientWrapper) assembleRequest(method string, api string, queryParams url.Values, body io.Reader) (*http.Request, error) {
	apiUrl := wrapper.Domain + api
	if queryParams != nil {
		apiUrl = apiUrl + "?" + queryParams.Encode()
	}
	return http.NewRequest(method, apiUrl, body)
}
