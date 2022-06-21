package main

import (
    "os"
    "flag"
    "bytes"
    "log"
    "time"

    "github.com/google/uuid"
)

var (
    flagSet = flag.NewFlagSet("mutex", flag.ContinueOnError)
    DbPath = flagSet.String("dbPath", "./mutex_site.db", "Path to site SQLite database file")
	Addr = flagSet.String("addr", "localhost:8080", "Server listen address and port")
	AddrTLS = flagSet.String("addrTLS", "", "TCP address to listen to TLS (aka SSL or HTTPS) requests. Leave empty to disable TLS")
	CertFile = flagSet.String("certFile", "./ssl-cert.pem", "Path to TLS certificate file")
	Compress = flagSet.Bool("compress", false, "Enables transparent response compression if set to true")
	Dir = flagSet.String("dir", "./static", "Directory to serve static files from")
	GenerateIndexPages = flagSet.Bool("generateIndexPages", false, "Whether to generate directory index pages")
	KeyFile = flagSet.String("keyFile", "./ssl-cert.key", "Path to TLS key file")
	Vhost = flagSet.Bool("vhost", false, "Enables virtual hosting by prepending the requested path with the requested hostname")
    AdminID = flagSet.String("adminID", "", "Secret admin identifier to access privileged functions")
    MaxWaitDurationString = flagSet.String("maxWaitDuration", "3s", "Maximum allowed wait duration for lock operation")
    MaxWaitDuration time.Duration
    PurgeIntervalString = flagSet.String("purgeInterval", "3m", "Time duration between purge cycles")
    PurgeInterval time.Duration
    ConfigError error
    ConfigErrorText string
)

func init() {
    uuid.EnableRandPool()

    var writer bytes.Buffer
    flagSet.SetOutput(&writer)
	ConfigError = flagSet.Parse(os.Args[1:])
    ConfigErrorText = writer.String()
    if *AdminID == "" {
        *AdminID = uuid.New().String()
    }

    var err error
    if MaxWaitDuration, err = time.ParseDuration(*MaxWaitDurationString); err != nil {
        log.Fatal(err)
    }

    if PurgeInterval, err = time.ParseDuration(*PurgeIntervalString); err != nil {
        log.Fatal(err)
    }
}
