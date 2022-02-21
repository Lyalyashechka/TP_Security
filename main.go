package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

func copyHeader(t, f http.Header) {
	for h, vv := range f {
		for _, v := range vv {
			t.Add(h, v)
		}
	}
}

type Handler struct {
}

func (p *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		handlerHTTPS(w, r)
	} else {
		handler(w, r)
	}
}

func connectHandshake(w http.ResponseWriter, r *http.Request) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return nil, err
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Connection established"))

	return conn, nil
}

func connectHijacker(w http.ResponseWriter) (net.Conn, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return nil, errors.New("hijacking not supported")
	}

	conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, err
	}

	return conn, nil
}

func transfer(dest io.WriteCloser, src io.ReadCloser) {
	defer dest.Close()
	defer src.Close()

	io.Copy(dest, src)
}

func handler(w http.ResponseWriter, req *http.Request) {
	req.RequestURI = ""
	req.Header.Del("Proxy-Connection")

	reqString, err := httputil.DumpRequest(req, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Request:\n" + string(reqString))

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	respString, err := httputil.DumpResponse(resp, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Response:\n" + string(respString))

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func handlerHTTPS(w http.ResponseWriter, r *http.Request) {

	destConn, err := connectHandshake(w, r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal("connectHandshake:", err)
		return
	}

	srcConn, err := connectHijacker(w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatal("connectHijacker:", err)
		return
	}

	go transfer(destConn, srcConn)
	go transfer(srcConn, destConn)
}

func main() {
	h := &Handler{}

	server := http.Server{
		Addr:    ":8080",
		Handler: h,
	}

	fmt.Println("Run server")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf(err.Error())
	}
}
