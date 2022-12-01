package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func Request(method, path string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(method, path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}
