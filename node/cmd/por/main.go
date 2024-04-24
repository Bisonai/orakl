package main

import (
	"context"

	"bisonai.com/orakl/node/pkg/por"
)

func main() {
	ctx := context.Background()
	app, err := por.New(ctx)
	if err != nil {
		panic(err)
	}
	err = app.Run(ctx)
	if err != nil {
		panic(err)
	}
}
