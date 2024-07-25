package common

type OutgoingSubmissionData struct {
	Symbol        string `json:"symbol"`
	Value         string `json:"value"`
	AggregateTime string `json:"aggregateTime"`
	Proof         string `json:"proof"`
	FeedHash      string `json:"feedHash"`
	Decimals      string `json:"decimals"`
}
