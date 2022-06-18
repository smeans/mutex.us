package main

import (
    "fmt"
    "testing"
    "net/http"
    "io/ioutil"
    "time"
    "log"
    "encoding/json"
)

var baseURL string
var testEmail string
var clientID string

func init() {
    if *AddrTLS != "" {
        baseURL = "https://" + *AddrTLS
    } else {
        baseURL = "http://" + *Addr
    }

    testEmail = fmt.Sprintf("test-%d@mutex.us", time.Now().Unix())

    log.Printf("mutex.us unit test initialized")
}

func TestRegister(t *testing.T) {
    clientURL := fmt.Sprintf("%s/api/client", baseURL)
    res, _ := http.Get(clientURL)

    body, _ := ioutil.ReadAll(res.Body)
    if res.StatusCode != 400 {
        t.Errorf("GET %s: expected 400: received: %d\n%s", clientURL,
                res.StatusCode, body)
    }

    registerURL := fmt.Sprintf("%s/api/client?register&email=%s", baseURL, testEmail)
    res, _ = http.PostForm(registerURL, nil)

    body, _ = ioutil.ReadAll(res.Body)
    if res.StatusCode != 200 {
        bodyText := string(body)
        t.Errorf("POST %s: expected 200: received: %d\n%s", clientURL,
                res.StatusCode, bodyText)
        return
    }

    var clientInfo ClientInfo

    err := json.Unmarshal(body, &clientInfo)
    if err != nil {
        t.Errorf("TestRegister: unable to unmarshal ClientInfo")
        return
    }

    clientID = clientInfo.ClientID
}

func TestLock(t *testing.T) {
    mutexURL := fmt.Sprintf("%s/api/client/%s/mutex/testmutex", baseURL,
            clientID)

    lockURL := fmt.Sprintf("%s?lock&waitTimeoutMs=3000", mutexURL)
    res, _ := http.PostForm(lockURL, nil)

    body, _ := ioutil.ReadAll(res.Body)
    bodyText := string(body)
    if res.StatusCode != 200 {
        t.Errorf("POST %s: expected 200: received: %d\n%s", lockURL,
                res.StatusCode, bodyText)
    }

    res, _ = http.PostForm(lockURL, nil)

    body, _ = ioutil.ReadAll(res.Body)
    if res.StatusCode != 409 {
        t.Errorf("POST %s: expected 400: received: %d\n%s", lockURL,
                res.StatusCode, bodyText)
    }

    unlockURL := fmt.Sprintf("%s?unlock", mutexURL)
    res, _ = http.PostForm(unlockURL, nil)

    body, _ = ioutil.ReadAll(res.Body)
    bodyText = string(body)
    if res.StatusCode != 200 {
        t.Errorf("POST %s: expected 200: received: %d\n%s", unlockURL,
                res.StatusCode, bodyText)
    }

    res, _ = http.PostForm(unlockURL, nil)

    body, _ = ioutil.ReadAll(res.Body)
    bodyText = string(body)
    if res.StatusCode != 409 {
        t.Errorf("POST %s: expected 409: received: %d\n%s", unlockURL,
                res.StatusCode, bodyText)
    }
}
