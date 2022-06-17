package main

import (
    "fmt"
    "log"
    "sync"
    "database/sql"
    "time"
    "errors"

    "github.com/google/uuid"
    "mutex/server/persist"
    "mutex/server/semaphore"
)

type ClientInfo struct {
    Email string `json:"email" db-pk:"true"`
    ClientID string `json:"clientID"`
}

type ClientResources struct {
    mu sync.Mutex
    semaphoreMap map[string]*semaphore.Semaphore
}

var crmMutex sync.Mutex
var clientResourceMap = map[string]*ClientResources{}

func init() {
    uuid.EnableRandPool()

    clientResourceMap = make(map[string]*ClientResources)

    err := persist.Init(*DbPath)
    if err != nil {
        log.Fatal(err)
    }
}

func getClientResources(clientID string) *ClientResources {
    if _, ok := clientResourceMap[clientID]; !ok {
        crmMutex.Lock()
        defer crmMutex.Unlock()

        cr := &ClientResources{
            semaphoreMap: make(map[string]*semaphore.Semaphore),
        }

        clientResourceMap[clientID] = cr
    }

    return clientResourceMap[clientID]
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

    // instantiate the map now to avoid another DB lookup later
    _ = getClientResources(clientInfo.ClientID)

    return clientInfo, err
}

func VerifyClient(clientID string) bool {
    if _, ok := clientResourceMap[clientID]; ok {
        return true
    }

    var clientInfoFound ClientInfo
    persist.Find(&ClientInfo{
        ClientID: clientID,
    }, func (rows *sql.Rows) {
        rows.Scan(&clientInfoFound.Email, &clientInfoFound.ClientID)
    })

    if clientInfoFound.ClientID != clientID {
        return false
    }

    _ = getClientResources(clientID)

    return true
}

func GetMaxWaitTimeout(clientID string) time.Duration {
    // !!!TBD!!! wsm -- this is where clients could have "premium"
    // accounts with longer timeouts
    return time.Duration(3) * time.Minute
}

func LockSemaphore(clientID string, mutexIdentifier string, waitTimeoutMs time.Duration) error {
    cr := getClientResources(clientID)
    if _, ok := cr.semaphoreMap[mutexIdentifier]; !ok {
        cr.mu.Lock()
        defer cr.mu.Unlock()
        cr.semaphoreMap[mutexIdentifier] = semaphore.NewSemaphore(1)
    }

    if !cr.semaphoreMap[mutexIdentifier].Lock(waitTimeoutMs) {
        return errors.New(fmt.Sprintf("unable to lock mutex '%s': timeout expired",
                mutexIdentifier))
    }

    return nil
}

func UnlockSemaphore(clientID string, mutexIdentifier string) error {
    cr := getClientResources(clientID)
    if _, ok := cr.semaphoreMap[mutexIdentifier]; !ok {
        return errors.New(fmt.Sprintf("invalid mutex identifier '%s'", mutexIdentifier))
    }

    if !cr.semaphoreMap[mutexIdentifier].Unlock() {
        return errors.New(fmt.Sprintf("unable to unlock mutex '%s' (mismatched lock/unlock calls?)",
                mutexIdentifier))
    }

    return nil
}
