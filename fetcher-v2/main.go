package main

import (
	"context"
	"log"

	"bisonai.com/fetcher/v2/utils"
)

func setup(ctx context.Context) {
	host, err := utils.MakeHost()
	if err != nil {
		log.Fatal(err)
	}

	discoverString := "orakl-rendezvous"
	//discover 1 peer for gossip protocol networking
	utils.DiscoverPeers(ctx, host, discoverString)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setup(ctx)
}
