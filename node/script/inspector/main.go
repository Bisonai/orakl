package main

import (
	"context"
	"fmt"

	"bisonai.com/orakl/node/pkg/checker/inspect"
)

func main() {
	ctx := context.Background()
	inspector, err := inspect.Setup(ctx)
	if err != nil {
		panic(err)
	}

	result, err := inspector.Inspect(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println(result)
}
