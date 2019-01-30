package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
	bpi "github.com/smacker/better-public-inbox"
	"github.com/smacker/better-public-inbox/server"
)

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Printf("Usage: %s [OPTIONS] path-to-repository\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	logrus.SetLevel(logrus.DebugLevel)

	loader := bpi.NewDirLoader(flag.Arg(0))
	store, err := bpi.NewMemStore(loader)
	if err != nil {
		logrus.Fatal(err)
	}

	server := server.NewHTTPServer(store)

	logrus.Info("starting server")
	http.ListenAndServe("0.0.0.0:8000", server)
}
