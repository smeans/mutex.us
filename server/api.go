package main

import (
    "fmt"
    "log"
    "sync"
    "database/sql"
    "time"
    "errors"
    "sync/atomic"

    "github.com/google/uuid"
    "mutex/server/persist"
    "mutex/server/semaphore"
)

type ClientInfo struct {
    Email string `json:"email" db-pk:"true"`
    ClientID string `json:"clientID"`
}

type ClientResources struct {
    mu sync.RWMutex
    totalLocks *int32
    previousTotalLocks int32
    totalUnlocks *int32
    previousTotalUnlocks int32
    semaphoreMap map[string]*semaphore.Semaphore
}

var crmMutex sync.RWMutex
var clientResourceMap = map[string]*ClientResources{}
var purgeClientChannel chan string

func init() {
    uuid.EnableRandPool()

    clientResourceMap = make(map[string]*ClientResources)
    purgeClientChannel = make(chan string)

    go PurgeClientWorker()
    go PurgeClientDaemon()
}

func getClientResources(clientID string) (cr *ClientResources) {
    crmMutex.RLock()
    cr, ok := clientResourceMap[clientID]
    crmMutex.RUnlock()

    if ok {
        return cr
    }

    crmMutex.Lock()
    defer crmMutex.Unlock()

    cr = &ClientResources{
        totalLocks: new(int32),
        totalUnlocks: new(int32),
        semaphoreMap: make(map[string]*semaphore.Semaphore),
    }

    clientResourceMap[clientID] = cr

    return cr
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
    crmMutex.RLock()
    _, ok := clientResourceMap[clientID]
    crmMutex.RUnlock()

    if ok {
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

func LockSemaphore(clientID string, mutexIdentifier string, waitTimeoutMs time.Duration,
            done <-chan struct{}) error {
    cr := getClientResources(clientID)

    cr.mu.Lock()
    semaphoreInstance, ok := cr.semaphoreMap[mutexIdentifier]

    if !ok {
        semaphoreInstance = semaphore.NewSemaphore(1)
        cr.semaphoreMap[mutexIdentifier] = semaphoreInstance
    }
    cr.mu.Unlock()

    if err := semaphoreInstance.Lock(waitTimeoutMs, done); err != nil {
        return err
    }

    atomic.AddInt32(cr.totalLocks, 1)

    return nil
}

func UnlockSemaphore(clientID string, mutexIdentifier string) error {
    cr := getClientResources(clientID)
    cr.mu.RLock()
    defer cr.mu.RUnlock()

    if _, ok := cr.semaphoreMap[mutexIdentifier]; !ok {
        return errors.New(fmt.Sprintf("invalid mutex identifier '%s'", mutexIdentifier))
    }

    if !cr.semaphoreMap[mutexIdentifier].Unlock() {
        return errors.New(fmt.Sprintf("unable to unlock mutex '%s' (mismatched lock/unlock calls?)",
                mutexIdentifier))
    }

    atomic.AddInt32(cr.totalUnlocks, 1)

    return nil
}

func PurgeClientWorker() {
    log.Println("PurgeClientWorker started")
    for clientID := range purgeClientChannel {
        log.Printf("PurgeClientWorker: purging client %s", clientID)
        crmMutex.Lock()
        delete(clientResourceMap, clientID)
        crmMutex.Unlock()
    }
    log.Println("PurgeClientWorker exiting")
}

func PurgeClientDaemon() {
    log.Println("PurgeClientDaemon started")
    for {
        log.Printf("PurgeClientDaemon: sleeping %v", PurgeInterval)
        time.Sleep(PurgeInterval)
        log.Printf("PurgeClientDaemon: purging idle clients")
        PurgeIdleClients()
    }
}

func PurgeIdleClients() {
    crmMutex.RLock()
    for clientID, cr := range clientResourceMap {
        cr.mu.Lock()
        if *cr.totalLocks == cr.previousTotalLocks && *cr.totalUnlocks == cr.previousTotalUnlocks {
            if *cr.totalLocks != *cr.totalUnlocks {
                log.Printf("PurgeIdleClients: client %s: mutex(s) held too long", clientID)
            }

            purgeClientChannel <- clientID
        }

        cr.previousTotalLocks = *cr.totalLocks
        cr.previousTotalUnlocks = *cr.totalUnlocks

        cr.mu.Unlock()
    }
    crmMutex.RUnlock()
}
