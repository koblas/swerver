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

var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
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
	client := &http.Client{}

	// delHopHeaders(req.Header)

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		appendHostToXForwardHeader(newreq.Header, clientIP)
	}

	resp, err := client.Do(newreq)
	if err != nil {
		http.Error(wr, "Server Error", http.StatusInternalServerError)
		log.Fatal("ServeHTTP:", err)
	}
	defer resp.Body.Close()

	delHopHeaders(resp.Header)

	copyHeader(wr.Header(), resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
}
