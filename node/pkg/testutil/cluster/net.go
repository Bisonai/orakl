package cluster

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"

	libp2psetup "bisonai.com/miko/node/pkg/libp2p/setup"
	"github.com/libp2p/go-libp2p/core/control"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// gater applies the cluster's partitions and a node's failing dials to that
// node's host. It outlives the node's hosts, so a control set before a start
// holds from the start's first dial.
type gater struct {
	n *Node
}

// allowDial reports whether n may dial p.
func (g gater) allowDial(p peer.ID) bool {
	return !g.n.failDials.Load() && !g.n.c.cutOff(g.n.index, p)
}

func (g gater) InterceptPeerDial(p peer.ID) bool { return g.allowDial(p) }

func (g gater) InterceptAddrDial(p peer.ID, _ ma.Multiaddr) bool { return g.allowDial(p) }

func (g gater) InterceptAccept(network.ConnMultiaddrs) bool { return true }

// InterceptSecured refuses connections from a partitioned peer, inbound ones
// included.
func (g gater) InterceptSecured(_ network.Direction, p peer.ID, _ network.ConnMultiaddrs) bool {
	return !g.n.c.cutOff(g.n.index, p)
}

func (g gater) InterceptUpgraded(network.Conn) (bool, control.DisconnectReason) { return true, 0 }

// bootAPI is an in-process stand-in for the boot API's peer sync: a sync
// returns the peers registered before the caller, then registers it.
type bootAPI struct {
	server *httptest.Server

	mu    sync.Mutex
	peers []libp2psetup.BootPeerModel
	next  int64
}

func newBootAPI() *bootAPI {
	b := &bootAPI{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/peer/sync", b.sync)
	b.server = httptest.NewServer(mux)
	return b
}

func (b *bootAPI) sync(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Url string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Url == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	b.mu.Lock()
	known := append([]libp2psetup.BootPeerModel(nil), b.peers...)
	registered := false
	for _, p := range b.peers {
		if p.Url == payload.Url {
			registered = true
			break
		}
	}
	if !registered {
		b.next++
		b.peers = append(b.peers, libp2psetup.BootPeerModel{ID: b.next, Url: payload.Url})
	}
	b.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(known)
}

// drop removes every registration of id, as the boot API's refresh job drops
// an unreachable peer.
func (b *bootAPI) drop(id peer.ID) {
	b.mu.Lock()
	defer b.mu.Unlock()
	kept := b.peers[:0]
	for _, p := range b.peers {
		if !urlHasPeer(p.Url, id) {
			kept = append(kept, p)
		}
	}
	b.peers = kept
}

func urlHasPeer(url string, id peer.ID) bool {
	addr, err := ma.NewMultiaddr(url)
	if err != nil {
		return false
	}
	info, err := peer.AddrInfoFromP2pAddr(addr)
	return err == nil && info.ID == id
}

func (b *bootAPI) close() { b.server.Close() }
