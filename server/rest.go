// REST API handlers and support routines.
package main

import (
    "log"
    "fmt"
    "bytes"
    "strings"
	"encoding/json"
    "math"
    "strconv"
    "time"
    "net/http"
)

type HttpError struct {
	StatusCode int `json:"statusCode"`
	ErrorMessage string `json:"errorMessage"`
}

type HttpSuccess struct {
    StatusCode int `json:"statusCode"`
}

// Marshal JSON without escaping <, >, and & characters.
func JSONMarshal(t interface{}) ([]byte, error) {
    buffer := &bytes.Buffer{}
    encoder := json.NewEncoder(buffer)
    encoder.SetEscapeHTML(false)
    err := encoder.Encode(t)
    return buffer.Bytes(), err
}

func WriteJSON(w http.ResponseWriter, req *http.Request, o interface{}) bool {
    w.Header().Set("Content-Type", "application/json; charset=utf8")

    data, _ := JSONMarshal(o)

	_, err := w.Write(data)

    if err != nil {
        log.Printf("WriteJSON error %v", err)
    }
    return err == nil
}

// Set the HTTP status code and return an error JSON payload to the client.
func reportError(w http.ResponseWriter, req *http.Request, statusCode int, errorMessage string) {
    // !!!TBD!!! wsm - consider logging errors here
    log.Printf("reportError: %s: %s", req.RemoteAddr, errorMessage)
	w.WriteHeader(statusCode)

	errorBody := &HttpError{
		StatusCode: statusCode,
		ErrorMessage: errorMessage,
	}

    WriteJSON(w, req, errorBody)
}

func statsHandler(w http.ResponseWriter, req *http.Request) {
    args := req.URL.Query()

    if adminID := string(args.Get("adminID")); adminID != *AdminID {
        reportError(w, req, 401, fmt.Sprintf("adminID '%s' is invalid", adminID))

        return
    }

    w.Header().Set("Content-Type", "application/json; charset=utf8")
    w.Write([]byte(fmt.Sprintf(`"{totalClients": %d`, len(clientResourceMap))))
    var totalLocks int64
    var totalUnlocks int64
    var totalIdleClients int64

    for _, cr := range clientResourceMap {
        totalLocks += int64(*cr.totalLocks)
        totalUnlocks += int64(*cr.totalUnlocks)
        if *cr.totalLocks == cr.previousTotalLocks && *cr.totalUnlocks == cr.previousTotalUnlocks {
            totalIdleClients += 1
        }
    }
    w.Write([]byte(fmt.Sprintf(`, "totalLocks": %d, "totalUnlocks": %d, "totalIdleClients": %d`,
            totalLocks, totalUnlocks, totalIdleClients)))
    w.Write([]byte("}"))
}

func apiClientHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		reportError(w, req, 400, "use POST to register a new client")
	}

    args := req.URL.Query()
    if !args.Has("register") || !args.Has("email") {
        reportError(w, req, 400, "usage: /api/client?register&email=[ValidEmailAddress]")

        return
    }

    email := string(args.Get("email"))

    clientInfo, err := RegisterClient(email)
    if err != nil {
        if (strings.HasPrefix(err.Error(), "UNIQUE constraint failed")) {
            reportError(w, req, 400, fmt.Sprintf("email '%s' is already in use", email))
        } else {
            reportError(w, req, 500, err.Error())
        }

        return
    }

    WriteJSON(w, req, clientInfo)
}

func apiMutexHandler(w http.ResponseWriter, req *http.Request, clientID string, mutexIdentifier string) {
    if !VerifyClient(clientID) {
        reportError(w, req, 401, fmt.Sprintf("client id '%s' is invalid", clientID))

        return
    }

    args := req.URL.Query()

    switch {
        case args.Has("lock"):
            // conn := GetConn(req)
            waitTimeoutMs := GetMaxWaitTimeout(clientID)
            if args.Has("waitTimeoutMs") {
                waitArgString := string(args.Get("waitTimeoutMs"))
                if waitArg, err := strconv.Atoi(waitArgString); err == nil {
                    waitTimeoutMs = time.Duration(math.Min(float64(waitTimeoutMs),
                            float64(time.Duration(waitArg) * time.Millisecond)))
                }
            }

            if err := LockSemaphore(clientID, mutexIdentifier, waitTimeoutMs,
                        req.Context().Done()); err != nil {
                reportError(w, req, 409, err.Error())

                return
            }

            success := &HttpSuccess{
                StatusCode: 200,
            }

            w.WriteHeader(200)
            WriteJSON(w, req, success)
        case args.Has("unlock"):
            if err := UnlockSemaphore(clientID, mutexIdentifier); err != nil {
                reportError(w, req, 409, err.Error())

                return
            }

            success := &HttpSuccess{
                StatusCode: 200,
            }

            w.WriteHeader(200)
            WriteJSON(w, req, success)
    }
}
