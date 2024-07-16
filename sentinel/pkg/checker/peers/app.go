package peers

import (
	"errors"
	"fmt"
	"os"
	"time"

	"bisonai.com/orakl/sentinel/pkg/alert"
	"bisonai.com/orakl/sentinel/pkg/request"
	"github.com/rs/zerolog/log"
)

var peerCount int
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
		log.Error().Err(err).Msgf("Using default peer check interval of %d seconds", DEFAULT_PEER_CHECK_INTERVAL)
	}

	initialCount, err := checkPeerCounts()
	if err != nil {
		return err
	}
	peerCount = initialCount

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
		if err != nil {
			log.Error().Err(err).Msg("Failed to check peer count")
			continue
		}

		if newPeerCount != peerCount {
			alarm(newPeerCount, peerCount)
			peerCount = newPeerCount
		}
	}
	return nil
}

func checkPeerCounts() (int, error) {
	oraklApiUrl := os.Getenv("ORAKL_API_URL")
	if oraklApiUrl == "" {
		return 0, errors.New("ORAKL_API_URL not found")
	}

	resp, err := request.Request[peerCountResponse](request.WithEndpoint(oraklApiUrl+peerCountEndpoint), request.WithTimeout(10*time.Second))
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
