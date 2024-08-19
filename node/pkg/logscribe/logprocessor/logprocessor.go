package logprocessor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"bisonai.com/orakl/node/pkg/db"
	errorsentinel "bisonai.com/orakl/node/pkg/error"
	"bisonai.com/orakl/node/pkg/secrets"
	"bisonai.com/orakl/node/pkg/utils/retrier"
	"github.com/google/go-github/github"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

const (
	maxGithubIssuesPerService   = 5
	readDeleteLogsQuery         = `DELETE FROM logs WHERE service = @service RETURNING service, timestamp, level, message, fields;`
	logAlreadyProcessedQuery    = `SELECT COUNT(*) FROM processed_logs WHERE log_hash = @hash`
	GetServicesQuery            = `SELECT DISTINCT service FROM logs;`
	DefaultBulkLogsCopyInterval = 10 * time.Minute
)

func New(ctx context.Context, options ...LogProcessingOption) (*LogProcessor, error) {
	githubToken := secrets.GetSecret("GITHUB_TOKEN")
	githubOwner := secrets.GetSecret("GITHUB_OWNER")
	githubRepo := secrets.GetSecret("GITHUB_REPO")

	if githubToken == "" || githubOwner == "" || githubRepo == "" {
		return nil, errorsentinel.ErrLogscribeGithubCredentialsNotFound
	}

	c := &LogProcessorConfig{
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

	chain := os.Getenv("CHAIN")
	if chain == "" {
		chain = "baobab"
	}

	return &LogProcessor{
		githubOwner:          githubOwner,
		githubRepo:           githubRepo,
		githubClient:         client,
		chain:                chain,
		cron:                 c.cron,
		bulkLogsCopyInterval: c.bulkLogsCopyInterval,
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

func (p *LogProcessor) fetchCurrentIssues(ctx context.Context) ([]string, error) {
	opts := &github.IssueListByRepoOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 30, // max limit
		},
	}
	user, _, err := p.githubClient.Users.Get(ctx, "")
	if err == nil && user != nil {
		creator := user.GetLogin()
		opts.Creator = creator
	} else {
		log.Info().Msg("Failed to get github user from token")
	}

	re := regexp.MustCompile(`\[[^\]]*\]`)
	issues := make([]string, 0)
	for {
		batchIssues, resp, err := p.githubClient.Issues.ListByRepo(ctx, p.githubOwner, p.githubRepo, opts)
		if err != nil {
			log.Error().Err(err).Msg("Failed to list issues")
			return nil, err
		}
		for _, issue := range batchIssues {
			if issue.PullRequestLinks == nil {
				title := issue.GetTitle()
				processedTitle := strings.TrimSpace(re.ReplaceAllString(title, ""))
				issues = append(issues, processedTitle)
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
		time.Sleep(500 * time.Millisecond) // to avoid being blocked by github
	}

	log.Info().Msgf("Fetched %d issues", len(issues))
	return issues, nil
}

func (p *LogProcessor) CreateGithubIssue(ctx context.Context, processedLogs []LogInsertModelWithCount, service string) {
	if p.githubOwner == "test" {
		return
	}

	currentIssues, err := p.fetchCurrentIssues(ctx)
	if err != nil {
		log.Warn().Msg("Failed to fetch current issues")
	}
	if len(currentIssues) == 0 {
		log.Debug().Msg("No existing issues")
	}

	issueCount := 0
	processedLogHashes := [][]interface{}{}
	for _, entry := range processedLogs {
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
		if slices.Contains(currentIssues, entry.Message) {
			log.Debug().Msgf("Issue already exists, skipping: %s", entry.Message)
			continue
		}

		entryJson, err := json.MarshalIndent(entry, "", "  ")
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal log")
			return
		}
		formattedBody := "```go\n" + string(entryJson) + "\n```"
		issueRequest := &github.IssueRequest{
			Title:  github.String(fmt.Sprintf("[%s][%s] %s", service, p.chain, entry.Message)),
			Body:   github.String(formattedBody),
			Labels: &[]string{"bug"},
		}

		err = retrier.Retry(func() error {
			_, resp, issueCreateErr := p.githubClient.Issues.Create(ctx, p.githubOwner, p.githubRepo, issueRequest)
			if issueCreateErr != nil {
				log.Error().Err(issueCreateErr).Msg("Failed to create github issue")
				return issueCreateErr
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

	log.Info().Msgf("Created %d github issues", issueCount)
}

func (p *LogProcessor) BulkCopyLogs(ctx context.Context, logsChannel <-chan *[]LogInsertModel) {
	ticker := time.NewTicker(p.bulkLogsCopyInterval)
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

func (p *LogProcessor) StartProcessingCronJob(ctx context.Context) error {
	if p.cron == nil {
		cron := cron.New(cron.WithSeconds())
		_, err := cron.AddFunc("0 0 0 * * 5", func() { // Run once a week, midnight between Thu/Fri
			services, err := db.QueryRows[Service](ctx, GetServicesQuery, nil)
			if err != nil {
				log.Error().Err(err).Msg("Failed to get services")
				return
			}
			for _, service := range services {
				processedLogs := ProcessLogs(ctx, service.Service)
				if len(processedLogs) > 0 {
					p.CreateGithubIssue(ctx, processedLogs, service.Service)
				}
			}
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to start processing cron job")
			return err
		}
		p.cron = cron
	}
	p.cron.Start()
	return nil
}
