package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
func (wrapper *HttpClientWrapper) Get(api string, header map[string]string, queryParams url.Values) (*http.Response, error) {
	request, err := wrapper.assembleRequest(http.MethodGet, api, header, queryParams, nil)
	if err != nil {
		return nil, err
	}
	return wrapper.client.Do(request)
}
func (wrapper *HttpClientWrapper) Post(api string, header map[string]string, queryParams url.Values, body *[]byte) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(*body)
	}
	request, err := wrapper.assembleRequest(http.MethodPost, api, header, queryParams, reader)
	if err != nil {
		return nil, err
	}
	return wrapper.client.Do(request)
}
func (wrapper *HttpClientWrapper) assembleRequest(method string, api string, header map[string]string, queryParams url.Values, body io.Reader) (*http.Request, error) {
	apiUrl := wrapper.Domain + api
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
func HandleResponse[T any](response *http.Response) (body T, err error) {
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err = errors.New(fmt.Sprintf("reponse:%s, status not 200,status:%d", response.Request.URL.Path, response.StatusCode))
		return
	}
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(bodyBytes, &body)
	return
}
