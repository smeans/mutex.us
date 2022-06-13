package main

import (
    "testing"
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
    
}
