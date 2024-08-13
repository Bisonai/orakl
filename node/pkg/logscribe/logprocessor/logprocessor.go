package logprocessor

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
	"bisonai.com/orakl/node/pkg/utils/retrier"
	"github.com/google/go-github/github"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

const (
	maxGithubIssuesPerService = 5
	readDeleteLogsQuery       = `DELETE FROM logs WHERE service = @service RETURNING service, timestamp, level, message, fields;`
	logAlreadyProcessedQuery  = `SELECT COUNT(*) FROM processed_logs WHERE log_hash = @hash`
	GetServicesQuery          = `SELECT DISTINCT service FROM logs;`
)

func New(ctx context.Context) (*LogProcessor, error) {
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

	return &LogProcessor{
		githubOwner:  githubOwner,
		githubRepo:   githubRepo,
		githubClient: client,
	}, nil
}

func ProcessLogs(ctx context.Context, service string) []LogInsertModelWithCount {
	processedLogs := make([]LogInsertModelWithCount, 0)
	logs, err := fetchLogs(ctx, service)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch logs")
		return nil
	}
	if len(logs) == 0 {
		log.Debug().Msgf("No logs to process")
		return nil
	}

	logMap := buildLogMap(logs)

	topOccurrences := sortLogsWithCount(logMap)
	topOccurrences = topOccurrences[:min(len(topOccurrences), maxGithubIssuesPerService)]

	for _, pair := range topOccurrences {
		processedLogs = append(
			processedLogs,
			LogInsertModelWithCount{OccurrenceCount: pair.count, LogInsertModel: pair.log},
		)
	}

	return processedLogs
}

func buildLogMap(logs []LogInsertModel) map[string]LogsWithCount {
	logMap := make(map[string]LogsWithCount) // {"hashedLog: {count, log}}
	for _, log := range logs {
		hash := hashLog(log)
		logsWithCount, exists := logMap[hash]
		if !exists {
			logsWithCount = LogsWithCount{
				count: 0,
				log:   log,
			}
		}
		logsWithCount.count++
		logMap[hash] = logsWithCount
	}
	return logMap
}

func sortLogsWithCount(hashLogPairs map[string]LogsWithCount) []LogsWithCount {
	pairs := make([]LogsWithCount, 0, len(hashLogPairs))
	for _, pair := range hashLogPairs {
		pairs = append(pairs, pair)
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].count > pairs[j].count
	})
	return pairs
}

func fetchLogs(ctx context.Context, service string) ([]LogInsertModel, error) {
	logs, err := db.QueryRows[LogInsertModel](ctx, readDeleteLogsQuery, map[string]any{
		"service": service,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to read logs")
		return nil, err
	}
	return logs, nil
}

func hashLog(log LogInsertModel) string {
	hash := sha256.New()
	hash.Write([]byte(log.Service))
	hash.Write([]byte(fmt.Sprintf("%d", log.Level)))
	hash.Write([]byte(log.Message))
	hash.Write(log.Fields)
	return hex.EncodeToString(hash.Sum(nil))
}

func (p *LogProcessor) CreateGithubIssue(ctx context.Context, processedLogs []LogInsertModelWithCount, service string) {
	if p.githubOwner == "test" {
		return
	}

	issueCount := 0
	processedLogHashes := [][]interface{}{}
	for _, entry := range processedLogs {
		logsJson := ""
		hash := hashLog(entry.LogInsertModel)
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

		err = retrier.Retry(func() error {
			_, resp, err := p.githubClient.Issues.Create(ctx, p.githubOwner, p.githubRepo, issueRequest)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create github issue")
				return err
			}
			if resp.StatusCode != http.StatusCreated {
				log.Error().Msgf("Failed to create github issue, status code: %d", resp.StatusCode)
				return errorsentinel.ErrLogscribeFailedToCreateGithubIssue
			}
			return nil
		}, 3, 500*time.Millisecond, 2*time.Second)

		if err == nil {
			processedLogHashes = append(processedLogHashes, []interface{}{hash})
			issueCount++
		}
	}

	if len(processedLogHashes) > 0 {
		err := db.BulkInsert(ctx, "processed_logs", []string{"log_hash"}, processedLogHashes)
		if err != nil {
			log.Error().Err(err).Msg("Failed to insert into processed_log")
		}
	}

	log.Debug().Msgf("Created %d github issues", issueCount)
}
