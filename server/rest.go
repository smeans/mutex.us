// REST API handlers and support routines.
package main

import (
    "fmt"
    "bytes"
    "strings"
	"encoding/json"
    "math"
    "strconv"
    "time"

    "github.com/valyala/fasthttp"
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

func WriteJSON(ctx *fasthttp.RequestCtx, o interface{}) {
    ctx.SetContentType("application/json; charset=utf8")

    data, _ := JSONMarshal(o)

	ctx.Write(data)
}

// Set the HTTP status code and return an error JSON payload to the client.
func reportError(ctx *fasthttp.RequestCtx, statusCode int, errorMessage string) {
    // !!!TBD!!! wsm - consider logging errors here
	ctx.SetStatusCode(statusCode)

	errorBody := &HttpError{
		StatusCode: statusCode,
		ErrorMessage: errorMessage,
	}

    WriteJSON(ctx, errorBody)
}

func statsHandler(ctx *fasthttp.RequestCtx) {
    args := ctx.QueryArgs()
    adminId := string(args.Peek("adminId"))
    if adminId != *AdminID {
        reportError(ctx, 401, fmt.Sprintf("adminId '%s' is invalid", adminId))

        return
    }

    ctx.SetContentType("application/json; charset=utf8")
	ctx.WriteString("{")
    ctx.WriteString(fmt.Sprintf(`"totalClients": %d`, len(clientResourceMap)))
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
    ctx.WriteString(fmt.Sprintf(`, "totalLocks": %d, "totalUnlocks": %d, "totalIdleClients": %d`,
            totalLocks, totalUnlocks, totalIdleClients))
    ctx.WriteString("}")
}

func apiClientHandler(ctx *fasthttp.RequestCtx) {
	if !ctx.IsPost() {
		reportError(ctx, fasthttp.StatusBadRequest, "use POST to register a new client")
	}

    args := ctx.QueryArgs()
    if !args.Has("register") || !args.Has("email") {
        reportError(ctx, fasthttp.StatusBadRequest, "usage: /api/client?register&email=[ValidEmailAddress]")

        return
    }

    email := string(args.Peek("email"))

    clientInfo, err := RegisterClient(email)
    if err != nil {
        if (strings.HasPrefix(err.Error(), "UNIQUE constraint failed")) {
            reportError(ctx, 400, fmt.Sprintf("email '%s' is already in use", email))
        } else {
            reportError(ctx, 500, err.Error())
        }

        return
    }

    WriteJSON(ctx, clientInfo)
}

func apiMutexHandler(ctx *fasthttp.RequestCtx, clientID string, mutexIdentifier string) {
    if !VerifyClient(clientID) {
        reportError(ctx, 401, fmt.Sprintf("client id '%s' is invalid", clientID))

        return
    }

    args := ctx.QueryArgs()

    switch {
        case args.Has("lock"):
            waitTimeoutMs := GetMaxWaitTimeout(clientID)
            if args.Has("waitTimeoutMs") {
                waitArgString := string(args.Peek("waitTimeoutMs"))
                if waitArg, err := strconv.Atoi(waitArgString); err == nil {
                    waitTimeoutMs = time.Duration(math.Min(float64(waitTimeoutMs),
                            float64(time.Duration(waitArg) * time.Millisecond)))
                }
            }

            if err := LockSemaphore(clientID, mutexIdentifier, waitTimeoutMs); err != nil {
                reportError(ctx, 409, err.Error())

                return
            }

            success := &HttpSuccess{
                StatusCode: 200,
            }

            ctx.SetStatusCode(200)
            WriteJSON(ctx, success)
        case args.Has("unlock"):
            if err := UnlockSemaphore(clientID, mutexIdentifier); err != nil {
                reportError(ctx, 409, err.Error())

                return
            }

            success := &HttpSuccess{
                StatusCode: 200,
            }

            ctx.SetStatusCode(200)
            WriteJSON(ctx, success)
    }
}
