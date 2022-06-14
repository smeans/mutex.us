package main

import (
    "log"

    "github.com/google/uuid"
    "mutex/server/persist"
)

type ClientInfo struct {
    Email string `json:"email" db-pk:"true"`
    ClientID string `json:"clientID"`
}

func init() {
    uuid.EnableRandPool()
    err := persist.Init(*DbPath)

    if err != nil {
        log.Fatal(err)
    }
}

func RegisterClient(email string) (*ClientInfo, error) {
    clientInfo := &ClientInfo{
        ClientID: uuid.New().String(),
        Email: email,
    }

    err := persist.Insert(clientInfo)
    if err != nil {
        return nil, err
    }

    return clientInfo, err
}
