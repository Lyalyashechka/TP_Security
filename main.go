package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
)

func copyHeader(t, f http.Header) {
	for h, vv := range f {
		for _, v := range vv {
			t.Add(h, v)
		}
	}
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

func main() {
	fmt.Println("Run server")
	http.HandleFunc("/", handler)
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err)
	}
}
