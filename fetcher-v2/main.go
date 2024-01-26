package main

import (
	"context"
	"flag"
	"log"
	"strconv"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"

	"bisonai.com/fetcher/v2/admin"
	"bisonai.com/fetcher/v2/utils"
)

func main() {
	discoverString := "orakl-test-discover-2024"
	port := flag.Int("p", 0, "app port")
	flag.Parse()
	if *port == 0 {
		log.Fatal("app port is required")
	}

	h, err := utils.MakeHost(*port + 7000)
	if err != nil {
		log.Fatal(err)
	}

	go utils.DiscoverPeers(context.Background(), h, discoverString)

	nodes := make(map[string]*utils.Node)
	// will be tracked to track connected peers
	discoveredPeers := make(map[peer.ID]peer.AddrInfo)

	appContext := admin.AppContext{
		DiscoverString: discoverString,
		Host:           &h,
		Nodes:          nodes,
		Peers:          discoveredPeers,
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		admin.Run(strconv.Itoa(*port), appContext)
	}()
	wg.Wait()
}
