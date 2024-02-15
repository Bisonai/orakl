package main

import (
	"context"
	"flag"
	"log"

	"bisonai.com/orakl/node/pkg/aggregator"
	"bisonai.com/orakl/node/pkg/libp2p"
)

func main() {
	discoverString := "orakl-test-discover-2024"
	port := flag.Int("p", 0, "libp2p port")

	flag.Parse()
	if *port == 0 {
		log.Fatal("Please provide a port to bind on with -p")
	}

	h, err := libp2p.MakeHost(*port)
	if err != nil {
		log.Fatal(err)
	}

	ps, err := libp2p.MakePubsub(context.Background(), h)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("establishing connection")
	go libp2p.DiscoverPeers(context.Background(), h, discoverString, "")

	aggregator, err := aggregator.NewAggregator(h, ps, "orakl-aggregator-2024-gazuaa")
	if err != nil {
		log.Fatal(err)
	}
	aggregator.Run()
}
