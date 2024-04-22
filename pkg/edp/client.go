package edp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	log "github.com/kyma-project/kyma-metrics-collector/pkg/logger"
)

type Client struct {
	HttpClient *http.Client
	Config     *Config
	Logger     *zap.SugaredLogger
}

const (
	edpPathFormat          = "%s/namespaces/%s/dataStreams/%s/%s/dataTenants/%s/%s/events"
	contentType            = "application/json;charset=utf-8"
	userAgentKMC           = "kyma-metrics-collector"
	userAgentKeyHeader     = "User-Agent"
	contentTypeKeyHeader   = "Content-Type"
	authorizationKeyHeader = "Authorization"
	clientName             = "edp-client"
	tenantIdPlaceholder    = "<subAccountId>"
	retryInterval          = 10 * time.Second
)

func NewClient(config *Config, logger *zap.SugaredLogger) *Client {
	httpClient := &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   config.Timeout,
	}
	return &Client{
		HttpClient: httpClient,
		Logger:     logger,
		Config:     config,
	}
}

func (c Client) NewRequest(dataTenant string) (*http.Request, error) {
	edpURL := c.getEDPURL(dataTenant)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, edpURL, bytes.NewBuffer([]byte{}))
	if err != nil {
		return nil, fmt.Errorf("failed generate request for EDP, %d: %v", http.StatusBadRequest, err)
	}

	req.Header.Set(userAgentKeyHeader, userAgentKMC)
	req.Header.Add(contentTypeKeyHeader, contentType)
	req.Header.Add(authorizationKeyHeader, fmt.Sprintf("Bearer %s", c.Config.Token))

	return req, nil
}

func (c Client) getEDPURL(dataTenant string) string {
	return fmt.Sprintf(edpPathFormat,
		c.Config.URL,
		c.Config.Namespace,
		c.Config.DataStreamName,
		c.Config.DataStreamVersion,
		dataTenant,
		c.Config.DataStreamEnv,
	)
}

func (c Client) Send(req *http.Request, payload []byte) (*http.Response, error) {
	// define retry policy.
	retryOptions := []retry.Option{
		retry.Attempts(uint(c.Config.EventRetry)),
		retry.Delay(retryInterval),
	}

	resp, err := retry.DoWithData(
		func() (*http.Response, error) {
			reqStartTime := time.Now()
			// send request.
			req.Body = io.NopCloser(bytes.NewReader(payload))
			resp, err := c.HttpClient.Do(req)
			duration := time.Since(reqStartTime)
			// check result.
			if err != nil {
				urlErr := err.(*url.Error)
				responseCode := http.StatusBadRequest
				if urlErr.Timeout() {
					responseCode = http.StatusRequestTimeout
				}
				// record metric.
				recordEDPLatency(duration, responseCode, c.getEDPURL(tenantIdPlaceholder))
				// log error.
				c.namedLogger().Debugf("req: %v", req)
				c.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).
					With(log.KeyRetry, log.ValueTrue).Warn("send event stream to EDP")
				return resp, err
			}

			// defer to close response body.
			defer func() {
				if err := resp.Body.Close(); err != nil {
					c.namedLogger().Warn(err)
				}
			}()

			// set error object if status is not StatusCreated.
			if resp.StatusCode != http.StatusCreated {
				err = fmt.Errorf("failed to send event stream as EDP returned HTTP: %d", resp.StatusCode)
				c.namedLogger().With(log.KeyError, err.Error()).With(log.KeyRetry, log.ValueTrue).
					Warn("send event stream as EDP")
			}

			// record metric.
			// the request URL is recorded without the actual tenant id to avoid having multiple histograms.
			recordEDPLatency(duration, resp.StatusCode, c.getEDPURL(tenantIdPlaceholder))
			return resp, err
		},
		retryOptions...,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to POST event to EDP")
	}

	c.namedLogger().Debugf("sent an event to '%s' with eventstream: '%s'", req.URL.String(), string(payload))
	return resp, nil
}

func (c *Client) namedLogger() *zap.SugaredLogger {
	return c.Logger.Named(clientName).With("component", "EDP")
}
