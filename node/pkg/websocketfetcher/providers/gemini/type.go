package gemini

const URL = "wss://api.gemini.com/v1/multimarketdata?bids=false&offers=false&top_of_book=true&symbols="

type Response struct {
	Type        string  `json:"type"`
	TimestampMs *int64  `json:"timestampms"`
	Events      []Event `json:"events"`
}

type Event struct {
	Type      string  `json:"type"`
	Symbol    string  `json:"symbol"`
	Price     string  `json:"price"`
	Side      *string `json:"side"`
	Reason    *string `json:"reason"`
	Remaining *string `json:"remaining"`
	Delta     *string `json:"delta"`
	Amount    *string `json:"amount"`
	MakerSide *string `json:"makerSide"`
}
