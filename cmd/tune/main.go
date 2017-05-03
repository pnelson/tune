package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pnelson/tune"
	"github.com/pnelson/tune/http"
)

var (
	h         = flag.Bool("h", false, "show this usage information")
	addr      = flag.String("addr", os.Getenv("TUNE_ADDR"), "http server address")
	listenKey = flag.String("listen-key", os.Getenv("TUNE_LISTEN_KEY"), "di.fm listen key")
	publicDir = flag.String("public-dir", filepath.Join(os.Getenv("GOPATH"), "src/github.com/pnelson/tune/public"), "public directory")
)

func init() {
	log.SetFlags(0)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION]... [CHANNEL]\n\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	if *h {
		flag.Usage()
		return
	}
	config := tune.Config{
		Addr:      *addr,
		ListenKey: *listenKey,
		PublicDir: *publicDir,
	}
	core, err := tune.NewCore(config)
	if err != nil {
		log.Fatal(err)
	}
	err = http.Serve(core)
	if err != nil {
		log.Fatal(err)
	}
}
