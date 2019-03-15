package reader_provider

import (
	"io"
	"net"
	"net/http"
	"time"
)

type ReaderProvider interface {
	GetReader() (r io.ReadCloser, err error)
}

type MediaReaderProvider interface {
	ReaderProvider
	GetName() string
}

type Named struct {
	ReaderProvider
	Name string
}

func (n *Named) GetName() string {
	return n.Name
}

type HTTPReaderProvider struct {
	URL     string
	Timeout time.Duration
	Get     func(client *http.Client, url string) (r *http.Response, err error)
}

func (p *HTTPReaderProvider) GetReader() (r io.ReadCloser, err error) {
	timeout := p.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	var (
		netTransport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: timeout,
			}).DialContext,
			TLSHandshakeTimeout: timeout,
		}
		netClient = &http.Client{
			Timeout:   timeout,
			Transport: netTransport,
		}

		response *http.Response
		get      = p.Get
	)

	if get == nil {
		get = func(client *http.Client, url string) (r *http.Response, err error) {
			return netClient.Get(url)
		}
	}
	if response, err = get(netClient, p.URL); err != nil {
		return
	}
	return &HTTPReader{response.Body, netClient}, nil
}

type HTTPReader struct {
	io.ReadCloser
	client *http.Client
}

func (r *HTTPReader) Client() *http.Client {
	return r.client
}
