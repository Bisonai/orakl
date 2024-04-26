package config

import "strings"

func generateGetConfigIdsQuery(configs []ConfigInsertModel) (string, []interface{}) {
	baseQuery := "SELECT id, name FROM configs WHERE name IN ("
	queryArgs := make([]interface{}, 0, len(configs))
	for _, config := range configs {
		queryArgs = append(queryArgs, config.Name)
	}

	return baseQuery + strings.Repeat("?,", len(configs)-1) + "?" + ")", queryArgs
}
