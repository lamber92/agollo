/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package http

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	url2 "net/url"
	"strings"
	"sync"
	"time"

	"github.com/apolloconfig/agollo/v4/component/log"
	"github.com/apolloconfig/agollo/v4/env"
	"github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/env/server"
	"github.com/apolloconfig/agollo/v4/extension"
	"github.com/apolloconfig/agollo/v4/perror"
	"github.com/apolloconfig/agollo/v4/utils"
)

var (
	// for on error retry
	onErrorRetryInterval = 2 * time.Second // 2s

	connectTimeout = 1 * time.Second // 1s

	// max retries connect apollo
	maxRetries = 5

	// defaultMaxConnsPerHost defines the maximum number of concurrent connections
	defaultMaxConnsPerHost = 512
	// defaultTimeoutBySecond defines the default timeout for http connections
	defaultTimeoutBySecond = 1 * time.Second
	// defaultKeepAliveSecond defines the connection time
	defaultKeepAliveSecond = 60 * time.Second
	// once for single http.Transport
	once sync.Once
	// defaultTransport http.Transport
	defaultTransport *http.Transport
)

func getDefaultTransport(insecureSkipVerify bool) *http.Transport {
	once.Do(func() {
		defaultTransport = &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			MaxIdleConns:        defaultMaxConnsPerHost,
			MaxIdleConnsPerHost: defaultMaxConnsPerHost,
			DialContext: (&net.Dialer{
				KeepAlive: defaultKeepAliveSecond,
				Timeout:   defaultTimeoutBySecond,
			}).DialContext,
		}
		if insecureSkipVerify {
			defaultTransport.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: insecureSkipVerify,
			}
		}
	})
	return defaultTransport
}

// CallBack 请求回调函数
type CallBack struct {
	SuccessCallBack   func([]byte, CallBack) (interface{}, error)
	NotModifyCallBack func() error
	AppConfigFunc     func() config.AppConfig
	Namespace         string
}

// Request 建立网络请求
func Request(ctx context.Context,
	requestURL string,
	connectionConfig *env.ConnectConfig,
	callBack *CallBack) (interface{}, error) {

	client := &http.Client{}
	// 如有设置自定义超时时间即使用
	if connectionConfig != nil && connectionConfig.Timeout != 0 {
		client.Timeout = connectionConfig.Timeout
	} else {
		client.Timeout = connectTimeout
	}
	var err error
	url, err := url2.Parse(requestURL)
	if err != nil {
		return nil, fmt.Errorf("invalid request URL: %s, err: %v", requestURL, err)
	}
	var insecureSkipVerify bool
	if strings.HasPrefix(url.Scheme, "https") {
		insecureSkipVerify = true
	}
	client.Transport = getDefaultTransport(insecureSkipVerify)
	retry := 0
	var retries = maxRetries
	if connectionConfig != nil && !connectionConfig.IsRetry {
		retries = 1
	}
	for {
		retry++
		if retry > retries {
			break
		}

		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, "GET", requestURL, nil)
		if req == nil || err != nil {
			// if error then sleep
			return nil, fmt.Errorf("generate request failed. url: %s, err: %s", requestURL, err)
		}

		// 增加header选项
		httpAuth := extension.GetHTTPAuth()
		if httpAuth != nil {
			headers := httpAuth.HTTPHeaders(requestURL, connectionConfig.AppID, connectionConfig.Secret)
			if len(headers) > 0 {
				req.Header = headers
			}
			host := req.Header.Get("Host")
			if len(host) > 0 {
				req.Host = host
			}
		}

		continueErr := errors.New("continue")
		doRequest := func() (interface{}, error) {
			var res *http.Response
			res, err = client.Do(req)
			if res != nil {
				defer res.Body.Close()
			}
			if res == nil || err != nil {
				log.Warnf("request failed. url: %s, err: %s", requestURL, err)
				// if error then sleep
				time.Sleep(onErrorRetryInterval)
				return nil, continueErr
			}

			// not modified break
			switch res.StatusCode {
			case http.StatusOK:
				responseBody, err := io.ReadAll(res.Body)
				if err != nil {
					log.Errorf("parse response body failed. try again. url: %s, err: %v", requestURL, err)
					// if error then sleep
					time.Sleep(onErrorRetryInterval)
					return nil, continueErr
				}
				if callBack != nil && callBack.SuccessCallBack != nil {
					return callBack.SuccessCallBack(responseBody, *callBack)
				}
				return nil, nil
			case http.StatusNotModified:
				log.Debugf("config not modified, err: %v", err)
				if callBack != nil && callBack.NotModifyCallBack != nil {
					return nil, callBack.NotModifyCallBack()
				}
				return nil, nil
			case http.StatusUnauthorized:
				// https://github.com/apolloconfig/apollo/issues/3652
				// During the long polling process, if the client is not directly connected to ApolloService
				// and passes through a gateway in the middle, it is likely to return http-status-code: 401.
				// Because the request timeout time set by the gateway is shorter than the suspension time
				// of apollo Notifications-API, the request is disconnected on the gateway side and retried.
				// At this time, the request time has been refreshed and does not match the signature of the
				// original request, which eventually causes apolloService to return 401.
				// Although this is not an elegant way, it is safest to handle it here for the time being.
				if strings.Contains(requestURL, "notifications/v2") {
					log.Debugf("config not modified, err: %v", err)
					if callBack != nil && callBack.NotModifyCallBack != nil {
						return nil, callBack.NotModifyCallBack()
					}
					return nil, nil
				}
				return nil, perror.ErrUnauthorized
			case http.StatusNotFound:
				return nil, perror.ErrNotFound
			default:
				log.Debugf("response return err. url: %s, http-status-code: %d", requestURL, res.StatusCode)
				// if error then sleep
				time.Sleep(onErrorRetryInterval)
				return nil, continueErr
			}
		}

		result, err := doRequest()
		if err != nil {
			if errors.Is(err, continueErr) {
				continue
			}
			return result, err
		}
		return result, err
	}

	log.Warnf("over max retry times still error. err: %v", err)
	if retry > retries {
		err = perror.ErrOverMaxRetryTimes
	}
	return nil, err
}

// RequestRecovery 可以恢复的请求
func RequestRecovery(ctx context.Context,
	appConfig config.AppConfig,
	connectConfig *env.ConnectConfig,
	callBack *CallBack) (response interface{}, err error) {

	for {
		host := loadBalance(appConfig)
		if host == "" {
			return nil, err
		}

		requestURL := fmt.Sprintf("%s%s", host, connectConfig.URI)
		response, err = Request(ctx, requestURL, connectConfig, callBack)
		if err == nil {
			return response, nil
		}

		server.SetDownNode(appConfig.GetHost(), host)

		if errors.Is(err, perror.ErrUnauthorized) ||
			errors.Is(err, perror.ErrOverMaxRetryTimes) ||
			errors.Is(err, perror.ErrNotFound) {
			return nil, err
		}
	}
}

func loadBalance(appConfig config.AppConfig) string {
	if !server.IsConnectDirectly(appConfig.GetHost()) {
		return appConfig.GetHost()
	}
	serverInfo := extension.GetLoadBalance().Load(server.GetServers(appConfig.GetHost()))
	if serverInfo == nil {
		return utils.Empty
	}
	return checkUrlSuffix(serverInfo.HomepageURL)
}

func checkUrlSuffix(url string) string {
	if !strings.HasSuffix(url, "/") {
		return url + "/"
	}
	return url
}
