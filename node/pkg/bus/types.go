package bus

const (
	// Modular Monolith pkg players
	// 1. admin: works as user interface to control whole system, sends message to other packages to control
	// 2. fetcher: fetches data from different sources and stores in db, sends message to aggregator in case of deviation
	// 3. aggregator: aggregates data through data sent from other nodes and stores global aggregates into db
	// 4. submitter: submits price into blockchain in regular basis
	ADMIN      = "admin"
	FETCHER    = "fetcher"
	AGGREGATOR = "aggregator"
	SUBMITTER  = "submitter"

	// Modular Monolith pkg commands, please follow {verb}_{noun} pattern for both variable name and value
	START_FETCHER_APP   = "start_fetcher_app"
	STOP_FETCHER_APP    = "stop_fetcher_app"
	REFRESH_FETCHER_APP = "refresh_fetcher_app"

	ACTIVATE_FETCHER   = "activate_fetcher"
	DEACTIVATE_FETCHER = "deactivate_fetcher"
)
