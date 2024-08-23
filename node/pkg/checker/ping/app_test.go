package ping

import (
	"context"
	"errors"
	"testing"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPinger is a mock implementation of probing.Pinger
type MockPinger struct {
	mock.Mock
	*probing.Pinger
}

func (m *MockPinger) Run() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPinger) Statistics() *probing.Statistics {
	args := m.Called()
	return args.Get(0).(*probing.Statistics)
}

func TestApp_Start_SuccessfulPing(t *testing.T) {
	mockPinger := new(MockPinger)
	mockPinger.On("Run").Return(nil)
	mockPinger.On("Statistics").Return(&probing.Statistics{
		PacketsRecv: 1,
		AvgRtt:      100 * time.Millisecond,
	})

	app, err := New(WithEndpoints([]string{"8.8.8.8"}))
	assert.NoError(t, err)

	// Ensure the mockPinger is used
	app.Endpoints[0].Pinger = mockPinger
	app.FailCount = make(map[string]int)

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	app.Start(ctx)

	assert.Equal(t, 0, app.FailCount["8.8.8.8"])
}

func TestApp_Start_FailedPing(t *testing.T) {
	mockPinger := new(MockPinger)
	mockPinger.On("Run").Return(errors.New("ping failed")) // Simulate ping failure

	app, err := New(WithEndpoints([]string{"8.8.8.8"}))
	assert.NoError(t, err)

	// Ensure the mockPinger is used
	app.Endpoints[0].Pinger = mockPinger
	app.FailCount = make(map[string]int)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	app.Start(ctx)

	// Ensure FailCount incremented due to failure
	assert.Equal(t, 1, app.FailCount["8.8.8.8"])
}

func TestApp_Start_ShutdownOnAllFailures(t *testing.T) {
	mockPinger := new(MockPinger)
	mockPinger.On("Run").Return(errors.New("ping failed")) // Simulate ping failure

	app, err := New(WithEndpoints([]string{"8.8.8.8", "1.1.1.1", "208.67.222.222"}))
	assert.NoError(t, err)

	// Ensure the mockPinger is used
	app.Endpoints[0].Pinger = mockPinger
	app.Endpoints[1].Pinger = mockPinger
	app.Endpoints[2].Pinger = mockPinger
	app.FailCount = make(map[string]int)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	app.Start(ctx)

	// Check that FailCount for all endpoints has reached DefaultMaxFails
	for _, endpoint := range app.Endpoints {
		assert.Equal(t, DefaultMaxFails, app.FailCount[endpoint.Address])
	}
}
