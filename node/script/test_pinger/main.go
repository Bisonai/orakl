package main

import (
	"context"
	"fmt"

	"bisonai.com/miko/node/pkg/checker/ping"
)

func main() {
	ctx := context.Background()
	ping.Run(ctx)
	fmt.Println("done")
}
