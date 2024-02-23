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

	// Modular Monolith pkg commands
	START_FETCHER   = "start_fetcher"
	STOP_FETCHER    = "stop_fetcher"
	REFRESH_FETCHER = "refresh_fetcher"

	ACTIVATE_ADAPTER   = "activate_adapter"
	DEACTIVATE_ADAPTER = "deactivate_adapter"
)
