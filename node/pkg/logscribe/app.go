package logscribe

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	errorsentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/logscribe/api"
	"bisonai.com/orakl/node/pkg/logscribe/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/go-github/github"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

const (
	logsChannelSize              = 10_000
	DefaultBulkLogsCopyInterval  = 30 * time.Second
	readDeleteLogsQuery          = `DELETE FROM logs RETURNING *;`
	topOccurrencesForGithubIssue = 5
	logAlreadyProcessedQuery     = `SELECT COUNT(*) FROM processed_logs WHERE log_hash = @hash`
	insertIntoProcessedLogsQuery = "INSERT INTO processed_logs (log_hash) VALUES (@hash)"
)

func New(ctx context.Context, options ...AppOption) (*App, error) {
	githubToken := os.Getenv("GITHUB_TOKEN")
	githubOwner := os.Getenv("GITHUB_OWNER")
	githubRepo := os.Getenv("GITHUB_REPO")

	if githubToken == "" || githubOwner == "" || githubRepo == "" {
		return nil, errorsentinel.ErrLogscribeGithubCredentialsNotFound
	}

	c := &AppConfig{
		bulkLogsCopyInterval: DefaultBulkLogsCopyInterval,
	}
	for _, option := range options {
		option(c)
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &App{
		githubOwner:          githubOwner,
		githubRepo:           githubRepo,
		githubClient:         client,
		bulkLogsCopyInterval: c.bulkLogsCopyInterval,
		cron:                 c.cron,
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

	go a.bulkCopyLogs(ctx, logsChannel)

	if a.cron == nil {
		cron := cron.New()
		_, err = cron.AddFunc("@weekly", func() { // Run once a week, midnight between Sat/Sun
			processedLogs := ProcessLogs(ctx)
			if len(processedLogs) > 0 {
				a.createGithubIssue(ctx, processedLogs)
			}
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to add cron job")
			return err
		}
		a.cron = cron
	}
	a.cron.Start()

	<-ctx.Done()

	if err := app.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown logscribe server")
		return err
	}

	return nil
}

func (a *App) bulkCopyLogs(ctx context.Context, logsChannel <-chan *[]api.LogInsertModel) {
	ticker := time.NewTicker(a.bulkLogsCopyInterval)
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

func ProcessLogs(ctx context.Context) map[string][]LogInsertModelWithIDWithCount {
	processedLogs := make(map[string][]LogInsertModelWithIDWithCount)
	logMap := make(map[string]map[string]LogsWithCount) // {"service": {hashedLog: {count, log}}}
	logs, err := fetchDeleteLogs(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch logs")
		return nil
	}
	if len(logs) == 0 {
		log.Debug().Msgf("No logs to process")
		return nil
	}

	for _, log := range logs {
		hash := hashLog(log)
		if logMap[log.Service] == nil {
			logMap[log.Service] = make(map[string]LogsWithCount)
		}
		logsWithCount, exists := logMap[log.Service][hash]
		if !exists {
			logsWithCount = LogsWithCount{
				count: 0,
				log:   log,
			}
		}
		logsWithCount.count++
		logMap[log.Service][hash] = logsWithCount
	}

	for service, hashLogPairs := range logMap {
		pairs := make([]LogsWithCount, 0, len(hashLogPairs))
		for _, pair := range hashLogPairs {
			pairs = append(pairs, pair)
		}
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].count > pairs[j].count
		})

		topOccurrences := pairs
		if len(pairs) > topOccurrencesForGithubIssue {
			topOccurrences = pairs[:topOccurrencesForGithubIssue]
		}
		processedLogs[service] = make([]LogInsertModelWithIDWithCount, 0, len(topOccurrences))
		for _, pair := range topOccurrences {
			processedLogs[service] = append(
				processedLogs[service],
				LogInsertModelWithIDWithCount{OccurrenceCount: pair.count, LogInsertModelWithID: pair.log},
			)
		}
	}

	return processedLogs
}

func fetchDeleteLogs(ctx context.Context) ([]LogInsertModelWithID, error) {
	logs, err := db.QueryRows[LogInsertModelWithID](ctx, readDeleteLogsQuery, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read logs")
		return nil, err
	}
	return logs, nil
}

func hashLog(log LogInsertModelWithID) string {
	hash := sha256.New()
	hash.Write([]byte(log.Service))
	hash.Write([]byte(fmt.Sprintf("%d", log.Level)))
	hash.Write([]byte(log.Message))
	hash.Write(log.Fields)
	return hex.EncodeToString(hash.Sum(nil))
}

func (a *App) createGithubIssue(ctx context.Context, processedLogs map[string][]LogInsertModelWithIDWithCount) {
	if a.githubOwner == "test" {
		return
	}

	issueCount := 0
	processedLogHashes := [][]interface{}{}
	for service, logs := range processedLogs {
		var logsJson string
		for _, entry := range logs {
			hash := hashLog(entry.LogInsertModelWithID)
			res, err := db.QueryRow[Count](ctx, logAlreadyProcessedQuery, map[string]any{
				"hash": hash,
			})
			if err != nil {
				log.Error().Err(err).Msg("Failed to check if log already processed")
				continue
			}
			if res.Count > 0 {
				log.Debug().Msg("Log already processed, skipping creation of github issue")
				continue
			}

			entryJson, err := json.MarshalIndent(entry, "", "  ")
			if err != nil {
				log.Error().Err(err).Msg("Failed to marshal log")
				return
			}
			logsJson += string(entryJson) + "\n"
			formattedBody := "```go\n" + logsJson + "\n```"
			issueRequest := &github.IssueRequest{
				Title:  github.String(fmt.Sprintf("[%s] %s", service, entry.Message)),
				Body:   github.String(formattedBody),
				Labels: &[]string{"bug"},
			}

			_, resp, err := a.githubClient.Issues.Create(ctx, a.githubOwner, a.githubRepo, issueRequest)
			if err != nil || resp.StatusCode != http.StatusCreated {
				log.Error().Err(err).Msg("Failed to create github issue")
			}

			processedLogHashes = append(processedLogHashes, []interface{}{hash})

			issueCount++
		}
	}

	_, err := db.BulkCopy(ctx, "processed_logs", []string{"log_hash"}, processedLogHashes)
	if err != nil {
		log.Error().Err(err).Msg("Failed to insert into processed_log")
	}

	log.Debug().Msgf("Created %d github issues", issueCount)
}
