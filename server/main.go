// Example static file server.
// https://github.com/valyala/fasthttp/blob/master/examples/fileserver/fileserver.go
package main

import (
	"log"
    "io/ioutil"
	"strings"
	"net/http"

	"mutex/server/persist"

    "github.com/gomarkdown/markdown"
)

func main() {
	if ConfigError != nil {
		log.Fatalf(ConfigErrorText)
	}

	log.Printf("sqlite database path: %s", *DbPath)
    err := persist.Init(*DbPath)
    if err != nil {
        log.Fatal(err)
    }

	log.Printf("adminID is %s", *AdminID)

	mux := http.NewServeMux()

	mux.HandleFunc("/stats", statsHandler)
	mux.HandleFunc("/api/client/", func (w http.ResponseWriter, req *http.Request) {
		path := req.URL.Path
		pathParams := strings.Split(path, "/")
		if len(pathParams) >= 6 && pathParams[4] == "mutex" {
			apiMutexHandler(w, req, pathParams[3], strings.Join(pathParams[5:], "/"))
		} else {
			w.WriteHeader(404)
		}
	})
	mux.HandleFunc("/api/client", apiClientHandler)
	mux.HandleFunc("/", mainHandler)

	if len(*Addr) > 0 {
		server := &http.Server{
			Addr: *Addr,
			Handler: mux,
		}

		log.Printf("Starting HTTP server on %q", *Addr)
		go func() {
			log.Fatal(server.ListenAndServe())
		}()
	}

	// Start HTTPS server.
	if len(*AddrTLS) > 0 {
		server := &http.Server{
			Addr: *AddrTLS,
			Handler: mux,
		}

		log.Printf("Starting HTTPS server on %q", *AddrTLS)
		go func() {
			log.Fatal(server.ListenAndServeTLS(*CertFile, *KeyFile))
		}()
	}

	// Wait forever.
	select {}
}

var readmeHTML []byte

func mainHandler(w http.ResponseWriter, req *http.Request) {
    if len(readmeHTML) == 0 {
        readmeData, _ := ioutil.ReadFile("../README.md")

        readmeHTML = markdown.ToHTML(readmeData, nil, nil)
    }
    w.Header().Set("Content-Type", "text/html; charset=utf8")
    w.Write([]byte(readmeHTML))
}
