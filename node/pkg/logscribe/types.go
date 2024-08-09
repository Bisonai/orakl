package logscribe

import (
	"bisonai.com/orakl/node/pkg/logscribe/api"
	"github.com/google/go-github/github"
)

type LogInsertModelWithID struct {
	api.LogInsertModel
	ID int `db:"id" json:"id"`
}

type App struct {
	githubOwner  string
	githubRepo   string
	githubClient *github.Client
}
