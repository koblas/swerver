package handler

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Set map[string]struct{}

var hopHeaders = Set{
	"Connection":          {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"Te":                  {}, // canonicalized version of "TE"
	"Trailers":            {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

func copyHeader(dst, src http.Header, omitHeaders Set) {
	for k, vv := range src {
		if _, found := omitHeaders[k]; found {
			continue
		}
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func appendHostToXForwardHeader(header http.Header, host string) {
	// If we aren't the first proxy retain prior
	// X-Forwarded-For information as a comma+space
	// separated list and fold multiple headers into one.
	if prior, ok := header["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + host
	}
	header.Set("X-Forwarded-For", host)
}

type proxy struct {
	remote string
}

func NewProxy(remote string) http.Handler {
	u, err := url.Parse(remote)
	if err != nil {
		log.Fatal(err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		log.Fatal("Only http and https proxy supported")
	}

	return &proxy{remote: remote}
}

func (p *proxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	rctx := chi.RouteContext(req.Context())

	remote := p.remote
	for idx, key := range rctx.URLParams.Keys {
		value := rctx.URLParams.Values[idx]
		remote = strings.ReplaceAll(remote, key, value)
	}

	newreq, err := http.NewRequest(req.Method, remote, req.Body)
	if err != nil {
		http.Error(wr, "Server Error", http.StatusInternalServerError)
		log.Fatal("ServeHTTP:", err)

		return
	}
	copyHeader(newreq.Header, req.Header, Set{})

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		appendHostToXForwardHeader(newreq.Header, clientIP)
	}

	client := &http.Client{}
	resp, err := client.Do(newreq)
	if err != nil {
		http.Error(wr, "Server Error", http.StatusInternalServerError)
		log.Fatal("ServeHTTP:", err)
	}
	defer resp.Body.Close()

	copyHeader(wr.Header(), resp.Header, hopHeaders)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
}
