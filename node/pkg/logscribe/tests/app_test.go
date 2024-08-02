package test

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"

	"bisonai.com/orakl/node/pkg/logscribe"
	"bisonai.com/orakl/node/pkg/utils/request"
	"github.com/stretchr/testify/assert"
)

func TestLogscribeRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cleanup(ctx)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := logscribe.Run(ctx)
		if err != nil {
			t.Logf("error running logscribe: %v", err)
		}
	}()

	response, err := request.RequestRaw(request.WithEndpoint("http://localhost:3000/api/v1/"))
	if response.StatusCode != http.StatusOK {
		t.Fatalf("error requesting logscribe: %v", err)
	}

	resultBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)

	}

	assert.Equal(t, string(resultBody), "Logscribe service")

	cancel()
	wg.Wait()
}
