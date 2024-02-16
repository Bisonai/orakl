package tests

import (
	"bisonai.com/orakl/api/feed"
)

// FIXME: redeclares structs that aren't accessable from test
// had to define again since _AdapterInsertModel isn't exported struct

type AdapterInsertModel struct {
	_AdapterInsertModel
	Feeds []feed.FeedInsertModel `json:"feeds"`
}

type _AdapterInsertModel struct {
	AdapterHash string `db:"adapter_hash" json:"adapterHash" validate:"required"`
	Name        string `db:"name" json:"name" validate:"required"`
	Decimals    int    `db:"decimals" json:"decimals" validate:"required"`
}
