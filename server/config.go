package main

import (
    "os"
    "flag"
    "bytes"
)

var (
    flagSet = flag.NewFlagSet("mutex", flag.ContinueOnError)
	Addr = flagSet.String("addr", "localhost:8080", "Server listen address and port")
	AddrTLS = flagSet.String("addrTLS", "", "TCP address to listen to TLS (aka SSL or HTTPS) requests. Leave empty to disable TLS")
	CertFile = flagSet.String("certFile", "./ssl-cert.pem", "Path to TLS certificate file")
	Compress = flagSet.Bool("compress", false, "Enables transparent response compression if set to true")
	Dir = flagSet.String("dir", "./static", "Directory to serve static files from")
	GenerateIndexPages = flagSet.Bool("generateIndexPages", false, "Whether to generate directory index pages")
	KeyFile = flagSet.String("keyFile", "./ssl-cert.key", "Path to TLS key file")
	Vhost = flagSet.Bool("vhost", false, "Enables virtual hosting by prepending the requested path with the requested hostname")
    ConfigError error
    ConfigErrorText string
)

func init() {
    var writer bytes.Buffer
    flagSet.SetOutput(&writer)
	ConfigError = flagSet.Parse(os.Args[1:])
    ConfigErrorText = writer.String()
}
