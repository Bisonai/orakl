package peers

import (
	"errors"
	"os"
	"time"

	"bisonai.com/miko/node/pkg/utils/request"
	"github.com/rs/zerolog/log"
)

var peerCheckInterval time.Duration

const (
	DEFAULT_PEER_CHECK_INTERVAL = 10 * time.Second
	peerCountEndpoint           = "/host/peercount"
	SAMPLE_COUNT                = 5
)

type peerCountResponse struct {
	Count int `json:"Count"`
}

func setUp() error {
	var err error
	peerCheckInterval, err = time.ParseDuration(os.Getenv("PEER_CHECK_INTERVAL"))
	if err != nil {
		peerCheckInterval = DEFAULT_PEER_CHECK_INTERVAL
		log.Info().Err(err).Dur("peerCheckInterval", peerCheckInterval).Msg("Using default peer check interval of 10s")
	}

	if os.Getenv("ORAKL_NODE_ADMIN_URL") == "" {
		return errors.New("ORAKL_NODE_ADMIN_URL not found")
	}

	return nil
}

func Start() error {
	err := setUp()
	if err != nil {
		return err
	}

	log.Info().Msg("Starting peer count checker")
	checkTicker := time.NewTicker(peerCheckInterval)
	defer checkTicker.Stop()

	check()
	for range checkTicker.C {
		check()
	}
	return nil
}

// check exports the peer count; alerts are Prometheus rules on orakl_peers_*.
func check() {
	count, err := checkPeerCounts()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to check peer count")
		errorsCounter.Inc()
		return
	}
	log.Debug().Int("peer count", count).Msg("peer count")
	recordPeerCount(count, time.Now())
}

func checkPeerCounts() (int, error) {
	mikoNodeAdminUrl := os.Getenv("ORAKL_NODE_ADMIN_URL")
	if mikoNodeAdminUrl == "" {
		return 0, errors.New("ORAKL_NODE_ADMIN_URL not found")
	}

	maxCount := 0
	succeeded := false
	for i := 0; i < SAMPLE_COUNT; i++ {
		resp, err := request.Request[peerCountResponse](request.WithEndpoint(mikoNodeAdminUrl+peerCountEndpoint), request.WithTimeout(10*time.Second))
		if err != nil {
			log.Warn().Err(err).Int("sample", i).Msg("failed to get peer count sample")
			continue
		}
		succeeded = true
		if resp.Count > maxCount {
			maxCount = resp.Count
		}
	}
	if !succeeded {
		return 0, errors.New("all peer count samples failed")
	}

	return maxCount, nil
}
