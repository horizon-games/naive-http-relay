package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"moul.io/http2curl"
)

func main() {
	var port uint16
	if len(os.Args) >= 2 {
		number, err := strconv.ParseUint(os.Args[1], 0, 16)
		if err == nil {
			port = uint16(number)
		} else {
			fmt.Printf("error: '%v' is not a port number\n", os.Args[1])
		}
	}
	if port == 0 {
		rand.Seed(time.Now().Unix())
		port = uint16(49152 + rand.Intn(65535-49152+1))
	}

	fmt.Printf("listening on http://0.0.0.0:%v\n\n", port)
	fmt.Printf("error: %v\n", http.ListenAndServe(fmt.Sprintf(":%v", port), &Relay{}))
}

type Relay struct {
	client http.Client
}

func (r *Relay) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, err = writer.Write([]byte(err.Error()))
		if err != nil {
			fmt.Printf("error: %v\n\n", err)
		}
		return
	}

	url := strings.TrimPrefix(request.RequestURI, "/")
	rawRequest, err := http.NewRequest(request.Method, url, bytes.NewReader(body))
	rawRequest.Header = request.Header.Clone()

	curlRequest := rawRequest.Clone(context.Background())
	curlRequest.Body = io.NopCloser(bytes.NewReader(body))
	curlRequest.Header.Del("Accept-Encoding")
	curlRequest.Header.Add("Accept-Encoding", "identity")

	command, err := http2curl.GetCurlCommand(curlRequest)
	if err == nil {
		fmt.Printf("%v\n\n", command)
	} else {
		fmt.Printf("error: %v\n\n", err)
	}

	response, err := r.client.Do(rawRequest)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, err = writer.Write([]byte(err.Error()))
		if err != nil {
			fmt.Printf("error: %v\n\n", err)
		}
		return
	}

	defer func() {
		err := response.Body.Close()
		if err != nil {
			fmt.Printf("error: %v\n\n", err)
		}
	}()

	body, err = io.ReadAll(response.Body)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, err = writer.Write([]byte(err.Error()))
		if err != nil {
			fmt.Printf("error: %v\n\n", err)
		}
		return
	}

	fmt.Printf("%v\n", response.Status)
	for name, values := range response.Header {
		fmt.Printf("%v: %v\n", name, strings.Join(values, ","))
	}
	if containsString(response.Header["Content-Encoding"], "gzip") {
		decompressor, err := gzip.NewReader(bytes.NewReader(body))
		if err == nil {
			body, err := io.ReadAll(decompressor)
			if err == nil {
				fmt.Printf("%v\n\n", string(body))
			} else {
				fmt.Printf("error: %v\n", err)
			}
		} else {
			fmt.Printf("error: %v\n", err)
		}
	} else {
		fmt.Printf("%v\n\n", string(body))
	}

	for name, values := range response.Header {
		for _, value := range values {
			writer.Header().Add(name, value)
		}
	}
	writer.WriteHeader(response.StatusCode)
	_, err = writer.Write(body)
	if err != nil {
		fmt.Printf("error: %v\n\n", err)
	}
}

func containsString(haystack []string, needle string) bool {
	for _, straw := range haystack {
		if straw == needle {
			return true
		}
	}

	return false
}
