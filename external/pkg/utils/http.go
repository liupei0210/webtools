package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
)

type HttpClientWrapper struct {
	Domain string
}

func NewHttpClientWrapper(domain string) *HttpClientWrapper {
	return &HttpClientWrapper{
		Domain: domain,
	}
}
func (wrapper *HttpClientWrapper) Get(api string, header map[string]string, queryParams url.Values, ctx ...context.Context) (*http.Response, error) {
	return request(http.MethodGet, wrapper.Domain, api, header, queryParams, nil, ctx...)
}
func (wrapper *HttpClientWrapper) Post(api string, header map[string]string, queryParams url.Values, body []byte, ctx ...context.Context) (*http.Response, error) {
	return request(http.MethodPost, wrapper.Domain, api, header, queryParams, body, ctx...)
}
func (wrapper *HttpClientWrapper) Put(api string, header map[string]string, queryParams url.Values, body []byte, ctx ...context.Context) (*http.Response, error) {
	return request(http.MethodPut, wrapper.Domain, api, header, queryParams, body, ctx...)
}
func HandleResponse[T any](response *http.Response) (body T, err error) {
	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}
	if log.GetLevel() == log.DebugLevel {
		requestBody, _ := io.ReadAll(response.Request.Body)
		log.Debugf("url:%s,requestBody:%s,responseStatus:%d,responseBody: %s", response.Request.URL, string(requestBody), response.StatusCode, string(bodyBytes))
	}
	if response.StatusCode != http.StatusOK {
		err = errors.New(fmt.Sprintf("reponse:%s, status not 200,status:%d,body:%s", response.Request.URL.Path, response.StatusCode, string(bodyBytes)))
		return
	}
	err = json.Unmarshal(bodyBytes, &body)
	return
}

var assembleRequest = func(method, domain, api string, header map[string]string, queryParams url.Values, body io.Reader) (*http.Request, error) {
	apiUrl := domain + api
	if queryParams != nil {
		apiUrl = apiUrl + "?" + queryParams.Encode()
	}
	request, err := http.NewRequest(method, apiUrl, body)
	if err != nil {
		return nil, err
	}
	for key, value := range header {
		request.Header.Set(key, value)
	}
	return request, nil
}
var assembleRequestWithContext = func(ctx context.Context, method, domain, api string, header map[string]string, queryParams url.Values, body io.Reader) (*http.Request, error) {
	apiUrl := domain + api
	if queryParams != nil {
		apiUrl = apiUrl + "?" + queryParams.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, method, apiUrl, body)
	if err != nil {
		return nil, err
	}
	for key, value := range header {
		request.Header.Set(key, value)
	}
	return request, nil
}
var request = func(method, domain, api string, header map[string]string, queryParams url.Values, body []byte, ctx ...context.Context) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	var req *http.Request
	var err error
	if len(ctx) > 0 {
		req, err = assembleRequestWithContext(ctx[0], method, domain, api, header, queryParams, reader)
	} else {
		req, err = assembleRequest(method, domain, api, header, queryParams, reader)
	}
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

func DoRequest[ResponseStruct any](method, domain, api string, header map[string]string, queryParams url.Values, body []byte, ctx ...context.Context) (resStruct ResponseStruct, err error) {
	response, err := request(method, domain, api, header, queryParams, body, ctx...)
	if err != nil {
		return
	}
	return HandleResponse[ResponseStruct](response)
}
