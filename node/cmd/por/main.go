package main

import (
	"context"

	"bisonai.com/miko/node/pkg/por"
	"bisonai.com/miko/node/pkg/utils/loginit"
)

func main() {
	ctx := context.Background()
	loginit.InitZeroLog()

	app, err := por.New(ctx)
	if err != nil {
		panic(err)
	}
	app.Run(ctx)
}
