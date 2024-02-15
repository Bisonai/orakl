package data

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

const (
	GetData = `SELECT * FROM data;`

	GetDataById = `SELECT * FROM data WHERE data_id = @id;`

	DeleteDataById = `DELETE FROM data WHERE data_id = @id RETURNING *;`

	GetDataByFeedId = `SELECT * FROM data WHERE feed_id = @id;`
)

func GenerateBulkInsertQuery(bulkInsertData []DataInsertModel) (string, error) {
	baseQuery := `INSERT INTO data (timestamp, value, aggregator_id, feed_id) VALUES`
	var insertQueries []string
	validate := validator.New()

	for _, insertData := range bulkInsertData {
		if err := validate.Struct(insertData); err != nil {
			return "", err
		}

		insertValueString := fmt.Sprintf("('%v'::timestamptz,%v,%v,%v)", insertData.Timestamp.String(), insertData.Value, insertData.AggregatorId, insertData.FeedId)
		insertQueries = append(insertQueries, insertValueString)
	}
	joinedString := strings.Join(insertQueries, ", ")
	result := baseQuery + " " + joinedString + " RETURNING *;"
	return result, nil
}
