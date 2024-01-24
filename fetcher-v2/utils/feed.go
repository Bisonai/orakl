package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type Definition struct {
	URL      string            `json:"url"`
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers"`
	Reducers []Reducer         `json:"reducers"`
}

type Data struct {
	Definition Definition `json:"definition"`
	AdapterID  string     `json:"adapterId"`
}

// test code, not for actual use
func GetTestFeed() (Data, error) {
	jsonStr := `{
        "definition": {
            "url": "https://api.huobi.pro/market/history/kline?period=1min&size=1&symbol=joyusdt",
            "method": "GET",
            "headers": {
                "Content-Type": "application/json"
            },
            "reducers": [
                {
                    "args": [
                        "data"
                    ],
                    "function": "PARSE"
                },
                {
                    "args": 0,
                    "function": "INDEX"
                },
                {
                    "args": [
                        "close"
                    ],
                    "function": "PARSE"
                },
                {
                    "args": 8,
                    "function": "POW10"
                },
                {
                    "function": "ROUND"
                }
            ]
        },
        "adapterId": "53"
    }`

	var data Data
	err := json.Unmarshal([]byte(jsonStr), &data)
	return data, err
}

// test code, not for actual use
func Test() {
	var requestResult interface{}

	feed, err := GetTestFeed()
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest(
		feed.Definition.Method,
		feed.Definition.URL,
		nil,
	)

	for key, value := range feed.Definition.Headers {
		req.Header.Set(key, value)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	resultBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(resultBody, &requestResult)

	builtReducer, err := BuildReducer(feed.Definition.Reducers)
	if err != nil {
		log.Fatal(err)
	}

	result, err := Pipe(builtReducer...)(requestResult)
	if err != nil {
		log.Fatal(err)
	}

	convertedResult, err := convertToBigInt(result)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(convertedResult)
}
