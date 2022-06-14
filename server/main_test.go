package main

import (
    "fmt"
    "testing"
    "net/http"
    "io/ioutil"
)

var baseURL string

func init() {
    if *AddrTLS != "" {
        baseURL = "https://" + *AddrTLS
    } else {
        baseURL = "http://" + *Addr
    }
}

func TestRegister(t *testing.T) {
    clientURL := fmt.Sprintf("%s/api/client", baseURL)
    res, _ := http.Get(clientURL)

    body, _ := ioutil.ReadAll(res.Body)
    if res.StatusCode != 400 {
        t.Errorf("GET %s: expected 400: received: %d\n%s", clientURL,
                res.StatusCode, body)
    }

    registerURL := fmt.Sprintf("%s/api/client?register&email=test@mutex.us", baseURL)
    res, _ = http.PostForm(registerURL, nil)

    body, _ = ioutil.ReadAll(res.Body)
    if res.StatusCode != 200 {
        t.Errorf("POST %s: expected 200: received: %d\n%s", clientURL,
                res.StatusCode, body)
    }
}
