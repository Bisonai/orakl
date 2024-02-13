package main

import (
	"context"
	"flag"
	"log"
	"strconv"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"

	"bisonai.com/orakl/node/admin"
	"bisonai.com/orakl/node/node"
	"bisonai.com/orakl/node/utils"
)

func main() {
	discoverString := "orakl-test-discover-2024"
	port := flag.Int("p", 0, "app port")
	bootstrap := flag.String("bootstrap", "", "bootstrap node multiaddress")

	discoveredPeers := make(map[peer.ID]peer.AddrInfo)
	flag.Parse()
	if *port == 0 {
		log.Fatal("app port is required")
	}

	h, err := utils.MakeHost(*port + 6999)
	if err != nil {
		log.Fatal(err)
	}

	ps, err := utils.MakePubsub(context.Background(), h)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("trying to establish initial connection")
	if *bootstrap != "" {
		log.Println("bootstrap:" + *bootstrap)
	}

	go utils.DiscoverPeers(context.Background(), h, discoverString, *bootstrap, discoveredPeers)

	// electorNode, err := utils.NewElectorNode(h, ps, "orakl-nodes-elector-2024")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// electorNode.Start()

	submitter, err := node.NewSubmitter(h, ps, "orakl-node-submitter-2024-gazua!")
	if err != nil {
		log.Fatal(err)
	}
	submitter.Run()

	nodes := make(map[string]*utils.FetcherNode)

	appContext := admin.AppContext{
		DiscoverString: discoverString,
		Host:           &h,
		Nodes:          nodes,
		Peers:          discoveredPeers,
		Pubsub:         ps,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		admin.Run(strconv.Itoa(*port), appContext)
	}()
	wg.Wait()

}
