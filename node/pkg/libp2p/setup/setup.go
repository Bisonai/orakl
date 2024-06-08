package setup

import (
	"context"
	"os"

	"bisonai.com/orakl/node/pkg/libp2p/utils"
	"bisonai.com/orakl/node/pkg/utils/request"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

type BootPeerModel struct {
	ID  int64  `db:"id" json:"id"`
	Url string `db:"url" json:"url"`
}

func ConnectThroughBootApi(ctx context.Context, h host.Host) error {
	url, err := utils.ExtractConnectionUrl(h)
	if err != nil {
		return err
	}

	apiEndpoint := os.Getenv("BOOT_API_URL")
	if apiEndpoint == "" {
		log.Info().Msg("boot api endpoint not set, using default url: http://localhost:8089")
		apiEndpoint = "http://localhost:8089"
	}

	log.Info().Str("url", url).Str("api_endpoint", apiEndpoint).Msg("connecting to boot API")

	dbPeers, err := request.UrlRequest[[]BootPeerModel](apiEndpoint+"/api/v1/peer/sync", "POST", map[string]any{
		"url": url,
	}, nil, "")
	if err != nil {
		log.Error().Err(err).Msg("Error connecting to boot API")
		return err
	}

	for _, dbPeer := range dbPeers {
		info, err := utils.ConnectionUrl2AddrInfo(dbPeer.Url)
		if err != nil {
			log.Error().Err(err).Msg("error parsing peer url: " + dbPeer.Url)
			continue
		}

		err = h.Connect(ctx, *info)
		if err != nil {
			log.Error().Err(err).Msg("error connecting to peer: " + dbPeer.Url)
			continue
		}
	}

	return nil
}
