package logscribe

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	errorsentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/logscribe/api"
	"bisonai.com/orakl/node/pkg/logscribe/utils"
	"bisonai.com/orakl/node/pkg/utils/retrier"
	"github.com/gofiber/fiber/v2"
	"github.com/google/go-github/github"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

const (
	logsChannelSize             = 10_000
	dbReadBatchSize             = 1000
	DefaultBulkLogsCopyInterval = 3 * time.Second
	DefaultProcessLogsInterval  = 2 * time.Second //24 * time.Hour
)

func New(ctx context.Context) (*App, error) {
	githubToken := os.Getenv("GITHUB_TOKEN")
	githubOwner := os.Getenv("GITHUB_OWNER")
	githubRepo := os.Getenv("GITHUB_REPO")

	if githubToken == "" || githubOwner == "" || githubRepo == "" {
		return nil, errorsentinel.ErrLogscribeGithubCredentialsNotFound
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &App{
		githubOwner:  githubOwner,
		githubRepo:   githubRepo,
		githubClient: client,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	log.Debug().Msg("Starting logscribe server")

	logsChannel := make(chan *[]api.LogInsertModel, logsChannelSize)

	app, err := utils.Setup("0.1.0", logsChannel)
	if err != nil {
		log.Error().Err(err).Msg("Failed to setup logscribe server")
		return err
	}

	v1 := app.Group("/api/v1")
	v1.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Logscribe service")
	})

	port := os.Getenv("LOGSCRIBE_PORT")
	if port == "" {
		port = "3000"
	}
	api.Routes(v1)

	go func() {
		if err := app.Listen(":" + port); err != nil {
			log.Fatal().Err(err).Msg("Failed to start logscribe server")
		}
	}()

	go bulkCopyLogs(ctx, logsChannel)
	go a.processLogs(ctx)

	<-ctx.Done()

	if err := app.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown logscribe server")
		return err
	}

	return nil
}

func bulkCopyLogs(ctx context.Context, logsChannel <-chan *[]api.LogInsertModel) {
	ticker := time.NewTicker(DefaultBulkLogsCopyInterval)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			bulkCopyEntries := [][]interface{}{}
		loop:
			for {
				select {
				case logs := <-logsChannel:
					for _, log := range *logs {
						bulkCopyEntries = append(bulkCopyEntries, []interface{}{log.Service, log.Timestamp, log.Level, log.Message, log.Fields})
					}
				default:
					break loop
				}
			}

			if len(bulkCopyEntries) > 0 {
				_, err := db.BulkCopy(ctx, "logs", []string{"service", "timestamp", "level", "message", "fields"}, bulkCopyEntries)
				if err != nil {
					log.Error().Err(err).Msg("Failed to bulk copy logs")
				}
			}
		}
	}
}

func (a *App) processLogs(ctx context.Context) {
	ticker := time.NewTicker(DefaultProcessLogsInterval)

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			processedLogs := make([]LogInsertModelWithID, 0)
			logMap := make(map[string]bool, 0)
			totalLogs := 0
			for {
				logs, err := fetchLogs(ctx)
				if err != nil || len(logs) == 0 {
					break
				}
				totalLogs += len(logs)

				for _, log := range logs {
					hash := hashLog(log)
					if !logMap[hash] {
						processedLogs = append(processedLogs, log)
						logMap[hash] = true
					}
				}

				err = deleteLogs(ctx)
				if err != nil {
					break
				}
			}

			log.Debug().Msgf("Processed %d logs with %d unique logs", totalLogs, len(processedLogs))
			if len(processedLogs) > 0 {
				a.createGithubIssue(ctx, processedLogs)
			}
		}
	}
}

func fetchLogs(ctx context.Context) ([]LogInsertModelWithID, error) {
	logs, err := db.QueryRows[LogInsertModelWithID](ctx, api.ReadLogs, map[string]any{
		"limit": dbReadBatchSize,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to read logs")
		return nil, err
	}
	return logs, nil
}

func deleteLogs(ctx context.Context) error {
	err := retrier.Retry(func() error {
		return db.QueryWithoutResult(ctx, api.DeleteLogs, map[string]any{
			"limit": dbReadBatchSize,
		})
	}, 3, 1*time.Second, 3*time.Second)

	if err != nil {
		log.Error().Err(err).Msg("Failed to delete logs")
		return err
	}
	return nil
}

func hashLog(log LogInsertModelWithID) string {
	hash := sha256.New()
	hash.Write([]byte(log.Service))
	hash.Write([]byte(fmt.Sprintf("%d", log.Level)))
	hash.Write([]byte(log.Message))
	hash.Write(log.Fields)
	return hex.EncodeToString(hash.Sum(nil))
}

func (a *App) createGithubIssue(ctx context.Context, logs []LogInsertModelWithID) {
	var logsJson string
	for _, entry := range logs {
		entryJson, err := json.Marshal(entry)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal log")
			return
		}
		logsJson += string(entryJson) + "\n"
	}

	formattedBody := "```go\n" + logsJson + "\n```"
	issueRequest := &github.IssueRequest{
		Title:  github.String("Issues from Logscribe"),
		Body:   github.String(formattedBody),
		Labels: &[]string{"bug"},
	}

	_, resp, err := a.githubClient.Issues.Create(ctx, a.githubOwner, a.githubRepo, issueRequest)
	if err != nil || resp.StatusCode != http.StatusCreated {
		log.Error().Err(err).Msg("Failed to create github issue")
	}
}
