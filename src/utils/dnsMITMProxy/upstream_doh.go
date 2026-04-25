package dnsMITMProxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"
)

const maxDoHResponseSize = 65535

type dohUpstream struct {
	client *http.Client
	url    string
}

func newDoHUpstream(url string, maxIdleConns uint, timeout time.Duration, skipVerify bool, serverName string) *dohUpstream {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: skipVerify,
	}
	if serverName != "" {
		tlsCfg.ServerName = serverName
	}
	return &dohUpstream{
		url: url,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConnsPerHost: int(maxIdleConns),
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
				ForceAttemptHTTP2:   true,
				TLSClientConfig:    tlsCfg,
				DisableKeepAlives:   false,
			},
		},
	}
}

func (u *dohUpstream) Query(ctx context.Context, req []byte, _ string) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.url, bytes.NewReader(req))
	if err != nil {
		return nil, fmt.Errorf("DoH request creation failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/dns-message")
	httpReq.Header.Set("Accept", "application/dns-message")

	resp, err := u.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("DoH request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected DoH response status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDoHResponseSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read DoH response: %w", err)
	}

	return body, nil
}

func (u *dohUpstream) Close() error {
	u.client.CloseIdleConnections()
	return nil
}
