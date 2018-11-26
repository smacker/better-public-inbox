package main

import (
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/smacker/better-public-inbox"
	"github.com/smacker/better-public-inbox/server"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	repo := "/Users/smacker/tmp/meta"
	loader := bpi.NewDirLoader(repo)
	store, err := bpi.NewMemStore(loader)
	if err != nil {
		logrus.Fatal(err)
	}

	server := server.NewHTTPServer(store)

	logrus.Info("starting server")
	http.ListenAndServe("0.0.0.0:8000", server)
}
