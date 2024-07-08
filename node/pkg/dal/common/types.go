package common

type OutgoingSubmissionData struct {
	Symbol        string   `json:"symbol"`
	Value         string   `json:"value"`
	AggregateTime string   `json:"aggregateTime"`
	Proof         []byte   `json:"proof"`
	FeedHash      [32]byte `json:"feedHash"`
	Decimals      string   `json:"decimals"`
}
