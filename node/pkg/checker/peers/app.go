package peers

import (
	"errors"
	"fmt"
	"os"
	"time"

	"bisonai.com/miko/node/pkg/alert"
	"bisonai.com/miko/node/pkg/utils/request"
	"github.com/rs/zerolog/log"
)

var peerCount int
var failCount int
var peerCheckInterval time.Duration

const (
	DEFAULT_PEER_CHECK_INTERVAL = 10 * time.Second
	peerCountEndpoint           = "/host/peercount"
)

type peerCountResponse struct {
	Count int `json:"Count"`
}

func setUp() error {
	var err error
	peerCheckInterval, err = time.ParseDuration(os.Getenv("PEER_CHECK_INTERVAL"))
	if err != nil {
		peerCheckInterval = DEFAULT_PEER_CHECK_INTERVAL
		log.Error().Err(err).Dur("peerCheckInterval", peerCheckInterval).Msg("Using default peer check interval of 10s")
	}

	initialCount, err := checkPeerCounts()
	if err != nil {
		return err
	}
	peerCount = initialCount
	failCount = 0

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

	for range checkTicker.C {
		newPeerCount, err := checkPeerCounts()
		log.Debug().Int("peer count", newPeerCount).Msg("peer count")
		if err != nil {
			log.Error().Err(err).Msg("Failed to check peer count")
			failCount++
			if failCount > 10 {
				alert.SlackAlert(fmt.Sprintf("failed to check peer count %d times. Check miko Sentinel logs", failCount))
				failCount = 0
			}
			continue
		}
		failCount = 0

		if newPeerCount != peerCount {
			alarm(newPeerCount, peerCount)
			peerCount = newPeerCount
		}
	}
	return nil
}

func checkPeerCounts() (int, error) {
	mikoNodeAdminUrl := os.Getenv("ORAKL_NODE_ADMIN_URL")
	if mikoNodeAdminUrl == "" {
		return 0, errors.New("ORAKL_NODE_ADMIN_URL not found")
	}

	resp, err := request.Request[peerCountResponse](request.WithEndpoint(mikoNodeAdminUrl+peerCountEndpoint), request.WithTimeout(10*time.Second))
	if err != nil {
		return 0, err
	}
	return resp.Count, nil
}

func alarm(newCount int, oldCount int) {
	if newCount > oldCount {
		alert.SlackAlert(fmt.Sprintf("number of peers in orakl offchain cluster has increased from %d to %d", oldCount, newCount))
	} else {
		alert.SlackAlert(fmt.Sprintf("number of peers in orakl offchain cluster has decreased from %d to %d", oldCount, newCount))
	}
}
