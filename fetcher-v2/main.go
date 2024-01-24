package main

import (
	"context"
	"flag"
	"log"
	"strconv"
	"sync"

	"bisonai.com/fetcher/v2/admin"
	"bisonai.com/fetcher/v2/utils"
)

func setup(appPort string, wg *sync.WaitGroup) {
	h, err := utils.MakeHost()
	if err != nil {
		log.Fatal(err)
	}

	//discover 1 peer for gossip protocol networking
	wg.Add(2)
	go func() {
		defer wg.Done()
		discoverString := "orakl-rendezvous"
		utils.DiscoverPeers(context.Background(), h, discoverString)
	}()
	go func() {
		defer wg.Done()
		admin.Run(appPort, &h)
	}()
}

func main() {
	port := flag.Int("p", 0, "app port")
	flag.Parse()
	if *port == 0 {
		log.Fatal("app port is required")
	}
	var wg sync.WaitGroup

	setup(strconv.Itoa(*port), &wg)
	wg.Wait()
}
