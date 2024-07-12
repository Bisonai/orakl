package bus

const (
	// Modular Monolith pkg players
	// 1. admin: works as user interface to control whole system, sends message to other packages to control
	// 2. fetcher: fetches data from different sources and stores in db, sends message to aggregator in case of deviation
	// 3. aggregator: aggregates data through data sent from other nodes and stores global aggregates into db
	// 4. reporter: submits price into blockchain in regular basis
	ADMIN      = "admin"
	FETCHER    = "fetcher"
	AGGREGATOR = "aggregator"
	REPORTER   = "reporter"
	LIBP2P     = "libp2p"

	// Modular Monolith pkg commands, please follow {verb}_{noun} pattern for both variable name and value
	START_FETCHER_APP   = "start_fetcher_app"
	STOP_FETCHER_APP    = "stop_fetcher_app"
	REFRESH_FETCHER_APP = "refresh_fetcher_app"

	ACTIVATE_FETCHER   = "activate_fetcher"
	DEACTIVATE_FETCHER = "deactivate_fetcher"

	START_AGGREGATOR_APP   = "start_aggregator_app"
	STOP_AGGREGATOR_APP    = "stop_aggregator_app"
	REFRESH_AGGREGATOR_APP = "refresh_aggregator_app"

	ACTIVATE_AGGREGATOR   = "activate_aggregator"
	DEACTIVATE_AGGREGATOR = "deactivate_aggregator"

	RENEW_SIGNER = "renew_signer"

	ACTIVATE_REPORTER   = "activate_reporter"
	DEACTIVATE_REPORTER = "deactivate_reporter"
	REFRESH_REPORTER    = "refresh_reporter"

	GET_PEER_COUNT = "get_peer_count"
	SYNC           = "sync"
)
