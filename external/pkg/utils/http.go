package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

type HttpClientWrapper struct {
	Domain     string
	client     *http.Client
	timeout    time.Duration
	retryTimes int
	retryDelay time.Duration
}

type Option func(*HttpClientWrapper)

func WithTimeout(timeout time.Duration) Option {
	return func(w *HttpClientWrapper) {
		w.timeout = timeout
	}
}

func WithRetry(times int, delay time.Duration) Option {
	return func(w *HttpClientWrapper) {
		w.retryTimes = times
		w.retryDelay = delay
	}
}

func NewHttpClientWrapper(domain string, opts ...Option) *HttpClientWrapper {
	wrapper := &HttpClientWrapper{
		Domain:     domain,
		timeout:    10 * time.Second,
		retryTimes: 3,
		retryDelay: time.Second,
	}

	for _, opt := range opts {
		opt(wrapper)
	}

	wrapper.client = &http.Client{
		Timeout: wrapper.timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 5 * time.Second,
			ForceAttemptHTTP2:   true,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}

	return wrapper
}

func (w *HttpClientWrapper) doWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i <= w.retryTimes; i++ {
		resp, err = w.client.Do(req)
		if err == nil {
			return resp, nil
		}

		if i < w.retryTimes {
			time.Sleep(w.retryDelay)
			continue
		}
	}

	return nil, fmt.Errorf("请求失败(重试%d次): %v", w.retryTimes, err)
}

func (w *HttpClientWrapper) Get(api string, header map[string]string, queryParams url.Values, ctx ...context.Context) (*http.Response, error) {
	return w.request(http.MethodGet, api, header, queryParams, nil, ctx...)
}

func (w *HttpClientWrapper) Post(api string, header map[string]string, queryParams url.Values, body []byte, ctx ...context.Context) (*http.Response, error) {
	return w.request(http.MethodPost, api, header, queryParams, body, ctx...)
}

func (w *HttpClientWrapper) request(method, api string, header map[string]string, queryParams url.Values, body []byte, ctx ...context.Context) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	apiURL := w.Domain + api
	if queryParams != nil {
		apiURL = apiURL + "?" + queryParams.Encode()
	}

	var req *http.Request
	var err error

	if len(ctx) > 0 {
		req, err = http.NewRequestWithContext(ctx[0], method, apiURL, reader)
	} else {
		req, err = http.NewRequest(method, apiURL, reader)
	}

	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置通用header
	req.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	GetLogger().Debugf("发送HTTP请求-method:%s url:%s body:%s", method, apiURL, string(body))
	return w.doWithRetry(req)
}

func HandleResponse[T any](response *http.Response) (body T, err error) {
	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}
	GetLogger().Debugf("url:%s,responseStatus:%d,responseBody: %s", response.Request.URL, response.StatusCode, string(bodyBytes))
	if response.StatusCode != http.StatusOK {
		err = errors.New(fmt.Sprintf("reponse:%s, status not 200,status:%d,body:%s", response.Request.URL.Path, response.StatusCode, string(bodyBytes)))
		return
	}
	err = json.Unmarshal(bodyBytes, &body)
	return
}

func DoRequest[ResponseStruct any](method, domain, api string, header map[string]string, queryParams url.Values, body []byte, expiredTime ...time.Duration) (resStruct ResponseStruct, err error) {
	wrapper := NewHttpClientWrapper(domain)
	if len(expiredTime) > 0 {
		wrapper.timeout = expiredTime[0]
	}

	ctx := context.Background()
	if len(expiredTime) > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, expiredTime[0])
		defer cancel()
	}

	resp, err := wrapper.request(method, api, header, queryParams, body, ctx)
	if err != nil {
		return
	}

	return HandleResponse[ResponseStruct](resp)
}
