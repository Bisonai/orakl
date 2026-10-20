//nolint:all
package aggregator

import (
	"bytes"
	"context"
	"testing"
	"time"

	"bisonai.com/miko/node/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestNewGlobalAggregateBulkWriter(t *testing.T) {
	bulkWriter := NewGlobalAggregateBulkWriter()
	assert.Equal(t, DefaultBufferSize, cap(bulkWriter.Buffer))
	assert.Equal(t, DefaultPgsqlBulkInsertInterval, bulkWriter.PgsqlBulkInsertInterval)
}

func TestGlobalAggregateBulkWriterStart(t *testing.T) {
	ctx := context.Background()
	cleanup, _, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	bulkWriter := NewGlobalAggregateBulkWriter()

	bulkWriter.Start(ctx)

	assert.NotEqual(t, nil, bulkWriter.ctx)
}

func TestGlobalAggregateBulkWriterStop(t *testing.T) {
	ctx := context.Background()
	cleanup, _, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	bulkWriter := NewGlobalAggregateBulkWriter()

	bulkWriter.Start(ctx)

	bulkWriter.Stop()

	assert.Equal(t, nil, bulkWriter.ctx)
}

func TestGlobalAggregateBulkWriterDataStore(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	bulkWriter := NewGlobalAggregateBulkWriter()

	bulkWriter.Start(ctx)
	defer bulkWriter.Stop()
	assert.NotEqual(t, nil, bulkWriter.ctx)

	node, err := NewAggregator(testItems.app.Host, testItems.app.Pubsub, testItems.topicString, testItems.tmpData.config, testItems.signer, testItems.latestLocalAggMap, nil)
	if err != nil {
		t.Fatal("error creating new node")
	}

	p, err := node.Signer.MakeGlobalAggregateProof(
		testItems.tmpData.globalAggregate.Value,
		testItems.tmpData.globalAggregate.Timestamp,
		"test_pair",
	)
	if err != nil {
		t.Fatal("error making global aggregate proof")
	}

	concatProof := bytes.Join([][]byte{p, p}, nil)

	proof := Proof{
		ConfigID: testItems.tmpData.globalAggregate.ConfigID,
		Round:    testItems.tmpData.globalAggregate.Round,
		Proof:    concatProof,
	}

	data, ok := newSubmissionData("test_pair", testItems.tmpData.globalAggregate, proof)
	assert.True(t, ok)
	assert.True(t, enqueueSubmissionData(bulkWriter.Buffer, data))
	time.Sleep(time.Second * 3)

	pgsLoadedProof, err := getProofFromPgs(ctx, testItems.tmpData.globalAggregate.ConfigID, testItems.tmpData.globalAggregate.Round)
	if err != nil {
		t.Fatal("error getting proof from pgs:" + err.Error())
	}
	assert.Equal(t, proof, pgsLoadedProof)

	pgsLoadedGlobalAggregate, err := getLatestGlobalAggregateFromPgs(ctx, testItems.tmpData.globalAggregate.ConfigID)
	if err != nil {
		t.Fatal("error getting global aggregate from pgs" + err.Error())
	}
	assert.Equal(t, testItems.tmpData.globalAggregate, pgsLoadedGlobalAggregate)
}

// Items enqueued while the writer is stopped/swapped must be persisted by the
// next writer sharing the App-owned buffer.
func TestGlobalAggregateBulkWriterSurvivesRefresh(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	}()

	app := testItems.app
	app.setGlobalAggregateBulkWriter()
	app.startGlobalAggregateBulkWriter(ctx)
	app.stopGlobalAggregateBulkWriter()

	globalAggregate := testItems.tmpData.globalAggregate
	globalAggregate.Round = globalAggregate.Round + 1
	proof := Proof{ConfigID: globalAggregate.ConfigID, Round: globalAggregate.Round, Proof: []byte("refresh-proof")}
	defer func() {
		_ = db.QueryWithoutResult(ctx, "DELETE FROM proofs WHERE config_id = @config_id", map[string]any{"config_id": globalAggregate.ConfigID})
	}()
	data, ok := newSubmissionData("test_pair", globalAggregate, proof)
	assert.True(t, ok)
	assert.True(t, enqueueSubmissionData(app.submissionBuffer, data))

	app.setGlobalAggregateBulkWriter()
	app.startGlobalAggregateBulkWriter(ctx)
	defer app.stopGlobalAggregateBulkWriter()
	time.Sleep(time.Second * 3)

	pgsLoadedProof, err := getProofFromPgs(ctx, globalAggregate.ConfigID, globalAggregate.Round)
	if err != nil {
		t.Fatal("error getting proof from pgs:" + err.Error())
	}
	assert.Equal(t, proof, pgsLoadedProof)

	pgsLoadedGlobalAggregate, err := getLatestGlobalAggregateFromPgs(ctx, globalAggregate.ConfigID)
	if err != nil {
		t.Fatal("error getting global aggregate from pgs:" + err.Error())
	}
	assert.Equal(t, globalAggregate, pgsLoadedGlobalAggregate)
}

func TestEnqueueSubmissionDataNonBlocking(t *testing.T) {
	buffer := make(chan SubmissionData, 1)
	data := SubmissionData{Symbol: "test_pair"}

	assert.True(t, enqueueSubmissionData(buffer, data))

	done := make(chan bool)
	go func() { done <- enqueueSubmissionData(buffer, data) }()
	select {
	case ok := <-done:
		assert.False(t, ok, "full buffer must drop")
	case <-time.After(time.Second):
		t.Fatal("enqueueSubmissionData blocked on full buffer")
	}

	assert.False(t, enqueueSubmissionData(nil, data), "nil buffer must drop")
	assert.Equal(t, 1, len(buffer))
}

func TestNewSubmissionDataSkipsEmpty(t *testing.T) {
	_, ok := newSubmissionData("test_pair", GlobalAggregate{Value: 0, Timestamp: time.Now()}, Proof{})
	assert.False(t, ok)
	_, ok = newSubmissionData("test_pair", GlobalAggregate{Value: 1}, Proof{})
	assert.False(t, ok)

	globalAggregate := GlobalAggregate{ConfigID: 1, Value: 1, Round: 2, Timestamp: time.Now()}
	proof := Proof{ConfigID: 1, Round: 2, Proof: []byte{1}}
	data, ok := newSubmissionData("test_pair", globalAggregate, proof)
	assert.True(t, ok)
	assert.Equal(t, SubmissionData{Symbol: "test_pair", GlobalAggregate: globalAggregate, Proof: proof}, data)
}

func TestBulkWriterUsesAppBuffer(t *testing.T) {
	app := New(nil, nil, nil)
	assert.NotNil(t, app.submissionBuffer)

	app.setGlobalAggregateBulkWriter()
	first := app.GlobalAggregateBulkWriter
	app.setGlobalAggregateBulkWriter()

	assert.NotSame(t, first, app.GlobalAggregateBulkWriter)
	assert.Equal(t, app.submissionBuffer, first.Buffer)
	assert.Equal(t, app.submissionBuffer, app.GlobalAggregateBulkWriter.Buffer)

	own := NewGlobalAggregateBulkWriter(WithBufferSize(3))
	assert.Equal(t, 3, cap(own.Buffer))
}

// A writer whose ctx is canceled (Stop() during refresh) must leave items in
// the shared buffer for the next writer.
func TestBulkInsertCanceledLeavesBuffer(t *testing.T) {
	buffer := make(chan SubmissionData, 2)
	bulkWriter := NewGlobalAggregateBulkWriter(WithBuffer(buffer))
	buffer <- SubmissionData{Symbol: "test_pair"}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	bulkWriter.bulkInsert(ctx)

	assert.Equal(t, 1, len(buffer))
}

func TestEnqueueSubmissionDataCountsDrops(t *testing.T) {
	before := enqueueDropped.Load()
	for i := 0; i < 5; i++ {
		assert.False(t, enqueueSubmissionData(nil, SubmissionData{Symbol: "test_pair"}))
	}
	assert.Equal(t, before+5, enqueueDropped.Load())
}
