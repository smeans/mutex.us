// REST API handlers and support routines.
package main

import (
    "bytes"
	"encoding/json"

    "github.com/valyala/fasthttp"
)

type HttpError struct {
	StatusCode int `json:"statusCode"`
	ErrorMessage string `json:"errorMessage"`
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
	ctx.SetStatusCode(statusCode)

	errorBody := &HttpError{
		StatusCode: statusCode,
		ErrorMessage: errorMessage,
	}

    WriteJSON(ctx, errorBody)
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
        reportError(ctx, 400, err.Error())

        return
    }

    WriteJSON(ctx, clientInfo)
}
