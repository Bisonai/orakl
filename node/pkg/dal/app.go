package dal

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"bisonai.com/orakl/node/pkg/common/types"
	"bisonai.com/orakl/node/pkg/dal/api"
	"bisonai.com/orakl/node/pkg/dal/collector"
	"bisonai.com/orakl/node/pkg/dal/hub"
	"bisonai.com/orakl/node/pkg/dal/utils/initializer"
	"bisonai.com/orakl/node/pkg/dal/utils/keycache"
	"bisonai.com/orakl/node/pkg/dal/wsserver"
	"bisonai.com/orakl/node/pkg/utils/request"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type Config = types.Config

func Run(ctx context.Context) error {
	log.Debug().Msg("Starting DAL API server")

	keyCache := keycache.NewAPIKeyCache(1 * time.Hour)
	keyCache.CleanupLoop(10 * time.Minute)

	adminEndpoint := os.Getenv("ORAKL_NODE_ADMIN_URL")
	if adminEndpoint == "" {
		return errors.New("ORAKL_NODE_ADMIN_URL is not set")
	}

	configs, err := fetchConfigs(ctx, adminEndpoint)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch configs")
		return err
	}

	collector, err := collector.NewCollector(ctx, configs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup collector")
		return err
	}

	hub := hub.HubSetup(ctx, configs)

	app, err := initializer.Setup(ctx, collector, hub, keyCache)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup DAL API server")
		return err
	}
	defer func() {
		_ = app.Shutdown()
	}()

	v1 := app.Group("")
	v1.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Orakl Node DAL API")
	})
	api.Routes(v1)

	port := os.Getenv("DAL_API_PORT")
	if port == "" {
		port = "8090"
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func(app *fiber.App) {
		defer wg.Done()
		err = app.Listen(":" + port)
		if err != nil {
			log.Error().Err(err).Msg("Failed to start DAL API server")
			return
		}
	}(app)

	l, err := net.Listen("tcp", ":8091")
	wss := wsserver.NewWSServer(keyCache, hub)
	s := &http.Server{
		Handler: wss,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = s.Serve(l)
		if err != nil {
			log.Error().Err(err).Msg("Failed to start WebSocket server")
			return
		}
	}()

	wg.Wait()
	return nil
}

func fetchConfigs(ctx context.Context, endpoint string) ([]Config, error) {
	return request.Request[[]Config](request.WithEndpoint(endpoint + "/config"))
}
