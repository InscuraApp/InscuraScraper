package main

import (
	"fmt"
	"inscurascraper/cmd"
	"inscurascraper/engine"
	"log"
	"net"
	"net/http"
	"os"

	V "inscurascraper/internal/version"
)

func showVersionAndExit() {
	fmt.Println(V.BuildString())
	os.Exit(0)
}

func main() {
	if _, isSet := os.LookupEnv("VERSION"); cmd.Config.VersionFlag &&
		!isSet /* NOTE: ignore this flag if ENV contains VERSION variable. */ {
		showVersionAndExit()
	}

	var (
		addr = net.JoinHostPort(
			cmd.Config.Bind,
			cmd.Config.Port)
		router = cmd.Router(engine.DefaultEngineName)
	)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
