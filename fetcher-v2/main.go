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
	discoveredPeers := make(map[peer.ID]peer.AddrInfo)
	flag.Parse()
	if *port == 0 {
		log.Fatal("app port is required")
	}

	h, err := utils.MakeHost(*port + 6999)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("trying to establish initial connection")
	go utils.DiscoverPeers(context.Background(), h, discoverString, discoveredPeers)
	// healthCheckNode, err := utils.NewHealthCheckerNode(context.Background(), h, "orakl-nodes-syncer-2024")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// healthCheckNode.Start(context.Background(), 10*time.Second)

	nodes := make(map[string]*utils.FetcherNode)
	// will be tracked to track connected peers

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
