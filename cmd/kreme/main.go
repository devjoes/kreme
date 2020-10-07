package main

import (
	"log"

	"github.com/devjoes/kreme/internal/config"
	"github.com/devjoes/kreme/internal/server"
	"github.com/devjoes/kreme/pkg/proxy"
)

func main() {
	config, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	headerBuilders, err := server.FromConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	proxy, err := proxy.NewProxy(&config.Proxy, server.GenerateHeadersForRequest(headerBuilders, config.Matches))
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(proxy.StartProxy())
}
