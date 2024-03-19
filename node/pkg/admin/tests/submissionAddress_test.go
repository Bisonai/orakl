//nolint:all

package tests

import (
	"context"
	"strconv"
	"testing"

	"bisonai.com/orakl/node/pkg/admin/submissionAddress"
	"bisonai.com/orakl/node/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestSubmissionAddressSync(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResultBefore, err := GetRequest[[]submissionAddress.SubmissionAddressModel](testItems.app, "/api/v1/submission-address", nil)
	if err != nil {
		t.Fatalf("error getting submission addresses before: %v", err)
	}

	_, err = RawPostRequest(testItems.app, "/api/v1/submission-address/sync", nil)
	if err != nil {
		t.Fatalf("error syncing submission addresses: %v", err)
	}

	readResultAfter, err := GetRequest[[]submissionAddress.SubmissionAddressModel](testItems.app, "/api/v1/submission-address", nil)
	if err != nil {
		t.Fatalf("error getting submission addresses after: %v", err)
	}

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more submission addresses after syncing")

	//cleanup
	err = db.QueryWithoutResult(ctx, "DELETE FROM submission_addresses;", nil)
	if err != nil {
		t.Fatalf("error cleaning up test: %v", err)
	}
}

func TestSubmissionAddressInsert(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	mockSubmissionAddress1 := submissionAddress.SubmissionAddressInsertModel{
		Name:    "test_submission_address_2",
		Address: "test_submission_address_2",
	}

	readResultBefore, err := GetRequest[[]submissionAddress.SubmissionAddressModel](testItems.app, "/api/v1/submission-address", nil)
	if err != nil {
		t.Fatalf("error getting submission addresses before: %v", err)
	}

	insertResult, err := PostRequest[submissionAddress.SubmissionAddressModel](testItems.app, "/api/v1/submission-address", mockSubmissionAddress1)
	if err != nil {
		t.Fatalf("error inserting submission address: %v", err)
	}
	assert.Equal(t, insertResult.Name, mockSubmissionAddress1.Name)

	readResultAfter, err := GetRequest[[]submissionAddress.SubmissionAddressModel](testItems.app, "/api/v1/submission-address", nil)
	if err != nil {
		t.Fatalf("error getting submission addresses after: %v", err)
	}

	assert.Greaterf(t, len(readResultAfter), len(readResultBefore), "expected to have more submission addresses after insertion")

	//cleanup
	err = db.QueryWithoutResult(ctx, "DELETE FROM submission_addresses WHERE id = @id;", map[string]interface{}{"id": insertResult.Id})
}

func TestSubmissionAddressGet(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[[]submissionAddress.SubmissionAddressModel](testItems.app, "/api/v1/submission-address", nil)
	if err != nil {
		t.Fatalf("error getting submission addresses: %v", err)
	}
	assert.Greater(t, len(readResult), 0)
}

func TestSubmissionAddressGetById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResult, err := GetRequest[submissionAddress.SubmissionAddressModel](testItems.app, "/api/v1/submission-address/"+strconv.FormatInt(*testItems.tmpData.submissionAddress.Id, 10), nil)
	if err != nil {
		t.Fatalf("error getting submission address by id: %v", err)
	}
	assert.Equal(t, readResult.Id, testItems.tmpData.submissionAddress.Id)
}

func TestSubmissionAddressDeleteById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	readResultBefore, err := GetRequest[[]submissionAddress.SubmissionAddressModel](testItems.app, "/api/v1/submission-address", nil)
	if err != nil {
		t.Fatalf("error getting submission addresses before: %v", err)
	}

	result, err := DeleteRequest[submissionAddress.SubmissionAddressModel](testItems.app, "/api/v1/submission-address/"+strconv.FormatInt(*testItems.tmpData.submissionAddress.Id, 10), nil)
	if err != nil {
		t.Fatalf("error deleting submission address by id: %v", err)
	}

	assert.Equal(t, result.Id, testItems.tmpData.submissionAddress.Id)

	readResultAfter, err := GetRequest[[]submissionAddress.SubmissionAddressModel](testItems.app, "/api/v1/submission-address", nil)
	if err != nil {
		t.Fatalf("error getting submission addresses after: %v", err)
	}

	assert.Lessf(t, len(readResultAfter), len(readResultBefore), "expected to have less submission addresses after deletion")
}

func TestSubmissionAddressUpdateById(t *testing.T) {
	ctx := context.Background()
	cleanup, testItems, err := setup(ctx)
	if err != nil {
		t.Fatalf("error setting up test: %v", err)
	}
	defer cleanup()

	mockSubmissionAddress1 := submissionAddress.SubmissionAddressInsertModel{
		Name:    "test_submission_address_2",
		Address: "test_submission_address_2",
	}

	updateResult, err := PatchRequest[submissionAddress.SubmissionAddressModel](testItems.app, "/api/v1/submission-address/"+strconv.FormatInt(*testItems.tmpData.submissionAddress.Id, 10), mockSubmissionAddress1)
	if err != nil {
		t.Fatalf("error updating submission address by id: %v", err)
	}

	assert.Equal(t, updateResult.Name, mockSubmissionAddress1.Name)
	assert.Equal(t, updateResult.Address, mockSubmissionAddress1.Address)

	readResult, err := GetRequest[submissionAddress.SubmissionAddressModel](testItems.app, "/api/v1/submission-address/"+strconv.FormatInt(*testItems.tmpData.submissionAddress.Id, 10), nil)
	if err != nil {
		t.Fatalf("error getting submission address by id: %v", err)
	}

	assert.Equal(t, readResult.Name, mockSubmissionAddress1.Name)
	assert.Equal(t, readResult.Address, mockSubmissionAddress1.Address)
}
