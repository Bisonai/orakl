// Package cluster runs several oracle nodes in one test process, so raft,
// reconnect and leadership behaviour can be reproduced in go test.
//
// Each node is a real libp2p host and gossipsub and a real aggregator.App (one
// raft-driven aggregator per feed) with a static signer generated in the test;
// with Options.PersistRounds also a SQLite node state, so rounds persist across
// restarts as in production. There are no fetchers or exchanges: every node's local
// aggregates are injected (Options.Value, Node.SetValue) and republished every
// 200 ms like the fetcher does, until Node.StopValue. There is no reporter
// either: every submission an aggregator hands to the in-node reporter is
// recorded per node, with whether that node led the round (Node.Submissions,
// Node.Led). Postgres is not used; the aggregators still publish each round
// to Redis (REDIS_HOST), and log the error when there is none.
//
// Typical use:
//
//	c := cluster.Start(t, cluster.Options{Nodes: 3, Feeds: []string{"a", "b"}})
//	leaders := c.WaitSingleLeader(10 * time.Second)
//	leaders = c.WaitStable(raft.DefaultCooldownPeriod, 30*time.Second) // no election cooling down
//	took := c.Failover(leaders["a"].Index(), 10*time.Second)          // stop, wait for a new leader
//	c.Node(1).Restart()                                               // new peer ID, as in production
//	c.WaitMesh(10 * time.Second)
//
// Controls:
//   - Node.Stop runs the production shutdown order (stop aggregators, close
//     host, wait, flush rounds); StopAggregators, CloseHost, WaitAggregators
//     and FlushRounds are the single steps, for a test of the hand-over itself.
//   - Node.Start / Node.Restart start a node on a new host with a new peer ID.
//   - Cluster.Partition / Cluster.Heal cut and restore the link between two
//     nodes (connections closed, dials and inbound connections refused).
//   - Node.FailDials makes the node's own dials fail, also before its start.
//   - Node.SetClockOffset shifts the clock the node's rounds are stamped with.
//   - Node.SetCanLead(false) marks the node unfit to lead (e.g. its reporter
//     can't submit): it resigns its feeds and holds off campaigning.
//   - Options.ConfigureApp / Options.ConfigureAggregator reach the
//     aggregator app and each aggregator (raft timers, new hooks) before they
//     start; Options.OnSubmission sees every submission, e.g. to model a
//     reporter that fails.
//
// Discovery: by default a starting node dials every running node directly.
// With Options.BootAPI the nodes find each other as in production instead:
// an in-process boot API (BOOT_API_URL, so such tests can't run in parallel),
// the startup sync (setup.SyncThroughBootApi) and the reconnect worker
// (libp2p helper). A stopped node is dropped from the boot list at once.
//
// The package is for tests only: it imports testing and must not be imported
// by production code. Tests of the aggregator package itself can't use it
// (import cycle).
package cluster
