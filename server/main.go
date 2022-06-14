// Example static file server.
// https://github.com/valyala/fasthttp/blob/master/examples/fileserver/fileserver.go
package main

import (
	"expvar"
	"log"
    "io/ioutil"

    "github.com/gomarkdown/markdown"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/expvarhandler"
)

func main() {
	if ConfigError != nil {
		log.Fatalf(ConfigErrorText)
	}

	// Setup FS handler
	fs := &fasthttp.FS{
		Root:               *Dir,
		IndexNames:         []string{"../README.md"},
		GenerateIndexPages: *GenerateIndexPages,
		Compress:           *Compress,
	}
	if *Vhost {
		fs.PathRewrite = fasthttp.NewVHostPathRewriter(0)
	}
	fsHandler := fs.NewRequestHandler()

	// Create RequestHandler serving server stats on /stats and files
	// on other requested paths.
	// /stats output may be filtered using regexps. For example:
	//
	//   * /stats?r=fs will show only stats (expvars) containing 'fs'
	//     in their names.
	requestHandler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/stats":
			expvarhandler.ExpvarHandler(ctx)
        case "/":
            mainHandler(ctx)
		case "/api/client":
			apiClientHandler(ctx)
		default:
			fsHandler(ctx)
			updateFSCounters(ctx)
		}
	}

	// Start HTTP server.
	if len(*Addr) > 0 {
		log.Printf("Starting HTTP server on %q", *Addr)
		go func() {
			if err := fasthttp.ListenAndServe(*Addr, requestHandler); err != nil {
				log.Fatalf("error in ListenAndServe: %v", err)
			}
		}()
	}

	// Start HTTPS server.
	if len(*AddrTLS) > 0 {
		log.Printf("Starting HTTPS server on %q", *AddrTLS)
		go func() {
			if err := fasthttp.ListenAndServeTLS(*AddrTLS, *CertFile, *KeyFile, requestHandler); err != nil {
				log.Fatalf("error in ListenAndServeTLS: %v", err)
			}
		}()
	}

	log.Printf("Serving files from directory %q", *Dir)
	log.Printf("See stats at http://%s/stats", *Addr)

	// Wait forever.
	select {}
}

func updateFSCounters(ctx *fasthttp.RequestCtx) {
	// Increment the number of fsHandler calls.
	fsCalls.Add(1)

	// Update other stats counters
	resp := &ctx.Response
	switch resp.StatusCode() {
	case fasthttp.StatusOK:
		fsOKResponses.Add(1)
		fsResponseBodyBytes.Add(int64(resp.Header.ContentLength()))
	case fasthttp.StatusNotModified:
		fsNotModifiedResponses.Add(1)
	case fasthttp.StatusNotFound:
		fsNotFoundResponses.Add(1)
	default:
		fsOtherResponses.Add(1)
	}
}

// Various counters - see https://golang.org/pkg/expvar/ for details.
var (
	// Counter for total number of fs calls
	fsCalls = expvar.NewInt("fsCalls")

	// Counters for various response status codes
	fsOKResponses          = expvar.NewInt("fsOKResponses")
	fsNotModifiedResponses = expvar.NewInt("fsNotModifiedResponses")
	fsNotFoundResponses    = expvar.NewInt("fsNotFoundResponses")
	fsOtherResponses       = expvar.NewInt("fsOtherResponses")

	// Total size in bytes for OK response bodies served.
	fsResponseBodyBytes = expvar.NewInt("fsResponseBodyBytes")
)

var readmeHTML []byte

func mainHandler(ctx *fasthttp.RequestCtx) {
    if len(readmeHTML) == 0 {
        readmeData, _ := ioutil.ReadFile("../README.md")

        readmeHTML = markdown.ToHTML(readmeData, nil, nil)
    }
    ctx.SetContentType("text/html; charset=utf8")
    ctx.Write(readmeHTML)
}
