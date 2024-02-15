# Offchain Aggregator POC

## Further works to be done

- functionality
- [ ] peer decision among pubsub subscribers (elect reporter)

- optimization
- [ ] faster peer search
- [ ] optimize pubsub peerfilter
- [ ] check following options (libp2p host option, pubsub.option, discovery.option, pubsub.TopicOpt, pubsub.SubOpt)
- [ ] is there a better way to generate id based on timerange? 🤔

- db integration
- [ ] store data into db (redis or pgsql)

- migrate codes
- [ ] fetcher codes
- [ ] reporter codes: how to deal with caver.js 🤔

- tests
- [ ] add test codes
- [ ] try to run with multiple nodes
- [ ] try to run with multiple hosts

## How to run

1. Prepare 2 consoles
2. Start application from each node. Fiber endpoint will be opened for each port with -p parameter.

```
go run main.go -p="3001"
go run main.go -p="3002"
```

3. Run testPubsub.sh bash file

```
./testPubsub.sh
```

## Structure

- `main.go`: entrypoint to run application
- `/admin`: gofiber app for user interface
  > - `/admin/admin.go`: functionality for fiber app initialization
  > - `/admin/node.go`: fiber controller for node (add, start, stop node)
- `/utils`: utilities
  > - `/utils/fetcher-node.go`: node simulating fetcher, generates random number, publish, and subscribe random number from other nodes
  > - `/utils/libp2p.go`: libp2p utility to setup libp2p host and nodes
  > - `/utils/local.go`: saves fiber locals which are referenced from fiber app (ex. host, nodes)
  > - `/utils/utils.fo`: contains basic utility functions such as random number generator

## Libp2p

- libp2p is library for decentralized networking. (https://libp2p.io/)

### Host

- Host is an instance containing all the functionality for networking
- Following is how host is declared from here

```golang
func MakeHost(listenPort int) (host.Host, error) {
	r := rand.Reader

	log.Println("generating private key")
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 2048, r)
	if err != nil {
		return nil, err
	}
	log.Println("generating libp2p options")

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
	}

	log.Println("generating libp2p instance")

	return libp2p.New(opts...)
}
```

- `ListenAddrStrings`: port for libp2p listening, normally set starting from 10000 from official example codes
- `Identity`: sets up PID for the host, one of the information needed for peer connection. (ip, port, PID)
- `DisableRelay`: disables relay function since it's not used

### Pubsub

- Pubsub is an instance for multiple pubsub channels
- It doesn't track for peers which are listening or publishing
- Gossip Pubsub doesn't require peers to be directly connected to send or receive message
- Following is how pubsub is declared from here

```golang
func MakePubsub(ctx context.Context, host host.Host) (*pubsub.PubSub, error) {
	log.Println("generating pubsub instance")
	var basePeerFilter pubsub.PeerFilter = func(pid peer.ID, topic string) bool {
		return strings.HasPrefix(pid.String(), "12D")
	}
	var fetcherSubParams = pubsub.DefaultGossipSubParams()

	fetcherSubParams.D = 2
	fetcherSubParams.Dlo = 1
	fetcherSubParams.Dhi = 3

	return pubsub.NewGossipSub(ctx, host, pubsub.WithPeerFilter(basePeerFilter), pubsub.WithGossipSubParams(fetcherSubParams))
}
```

- `PeerFilter`: It helps filter peer which is not of an interest. Once rules for topic string are settled, it could be better optimized.
- `fetcherSubParams.D`: Degree of the network, number of peers a node tries to maintain connection with
- `fetcherSubParams.Dlo`, `fetcherSubParams.Dhi`: Low and highest limit for number of peers which control when a node will prune or graft peers.

### Node

- Node is basic instance for single pubsub channel
- Each node has different `topicString`, which means peers subscribing to same topicString can only receive message published through this node
- Following is how node is declared from here

```golang
type FetcherNode struct {
	Host      host.Host
	Ps        *pubsub.PubSub
	Topic     *pubsub.Topic
	Sub       *pubsub.Subscription
	Data      map[string][]int
	NodeMutex sync.Mutex
	Cancel    context.CancelFunc
}

func NewNode(host host.Host, ps *pubsub.PubSub, topicString string) (*FetcherNode, error) {
	topic, err := ps.Join(topicString)
	if err != nil {
		return nil, err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	return &FetcherNode{
		Host:  host,
		Ps:    ps,
		Topic: topic,
		Sub:   sub,
		Data:  make(map[string][]int),
	}, nil
}
```

### Peer Discovery

- Peer discovery is done through `DiscoverPeers` function inside `utils/libp2p.go`
- Basic codebase is referenced from here: https://github.com/libp2p/go-libp2p/blob/6aa701ac36456e0d8862b631ad19f7f7fbc1f233/examples/chat-with-rendezvous/chat.go#L117-L182
- initializes DHT, connects bootstrap peers from dht, and then initialize `routingDiscovery` instance which contains functionality to advertise and search peers with same topic string. Connects peers which are found from `FindPeers` function

### Order of Execution

1. Host & Pubsub declaration

```golang
h, err := utils.MakeHost(*port + 6999)
if err != nil {
    log.Fatal(err)
}

ps, err := utils.MakePubsub(context.Background(), h)
if err != nil {
    log.Fatal(err)
}
```

2. Peer Discovery

```golang
go utils.DiscoverPeers(context.Background(), h, discoverString, *bootstrap, discoveredPeers)
```

3. Node declaration (called from gofiber controller)

```golang
func addNode(c *fiber.Ctx) error {
    ...
    node, err := utils.NewNode(*h, ps, topicString)
	if err != nil {
		log.Errorf("failed to create node: %s", err)
	}
    ...
}
```

4. Node start (called from go fiber controller)

```golang
func startNode(c *fiber.Ctx) error {
    ...
	node, err := utils.GetNode(c, topicString)
	if err != nil {
		log.Errorf("failed to load node: %s", err)
	}
	node.Start(time.Second * 2)
	...
}
```
