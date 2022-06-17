// Example static file server.
// https://github.com/valyala/fasthttp/blob/master/examples/fileserver/fileserver.go
package main

import (
	"log"
    "io/ioutil"
	"strings"

    "github.com/gomarkdown/markdown"
	"github.com/valyala/fasthttp"
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

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())
		pathParams := strings.Split(path, "/")

		switch {
        case path == "/":
            mainHandler(ctx)
		case strings.HasPrefix(path, "/api/client"):
			if len(pathParams) >= 6 && pathParams[4] == "mutex" {
				apiMutexHandler(ctx, pathParams[3], strings.Join(pathParams[5:], "/"))
			} else {
				apiClientHandler(ctx)
			}
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

var readmeHTML []byte

func mainHandler(ctx *fasthttp.RequestCtx) {
    if len(readmeHTML) == 0 {
        readmeData, _ := ioutil.ReadFile("../README.md")

        readmeHTML = markdown.ToHTML(readmeData, nil, nil)
    }
    ctx.SetContentType("text/html; charset=utf8")
    ctx.Write(readmeHTML)
}
