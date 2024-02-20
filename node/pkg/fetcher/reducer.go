package fetcher

import (
	"fmt"
	"math"
)

type Definition struct {
	Url      string            `json:"url"`
	Headers  map[string]string `json:"headers"`
	Method   string            `json:"method"`
	Reducers []Reducer         `json:"reducers"`
}

type Reducer struct {
	Function string      `json:"function"`
	Args     interface{} `json:"args"`
}

func ReduceAll(raw interface{}, reducers []Reducer) (float64, error) {
	var result float64
	for _, reducer := range reducers {
		var err error
		raw, err = reduce(raw, reducer)
		if err != nil {
			return 0, err
		}
	}
	result, ok := raw.(float64)
	if !ok {
		return 0, fmt.Errorf("cannot cast raw data to float")
	}
	return result, nil
}

func reduce(raw interface{}, reducer Reducer) (interface{}, error) {
	switch reducer.Function {
	case "INDEX":
		castedRaw, ok := raw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot cast raw data to []interface{}")
		}
		return castedRaw[reducer.Args.(int)], nil
	case "PARSE", "PATH":
		castedRaw, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot cast raw data to map[string]interface{}")
		}
		args := reducer.Args.([]string)
		for _, arg := range args {
			castedRaw = castedRaw[arg].(map[string]interface{})
		}
		return castedRaw, nil
	case "MUL":
		castedRaw, ok := raw.(float64)
		if !ok {
			return nil, fmt.Errorf("cannot cast raw data to float")
		}
		return castedRaw * reducer.Args.(float64), nil
	case "POW10":
		castedRaw, ok := raw.(float64)
		if !ok {
			return nil, fmt.Errorf("cannot cast raw data to float")
		}
		return float64(math.Pow10(reducer.Args.(int))) * castedRaw, nil
	case "ROUND":
		castedRaw, ok := raw.(float64)
		if !ok {
			return nil, fmt.Errorf("cannot cast raw data to float")
		}
		return math.Round(castedRaw), nil
	case "DIV":
		castedRaw, ok := raw.(float64)
		if !ok {
			return nil, fmt.Errorf("cannot cast raw data to float")
		}
		return castedRaw / reducer.Args.(float64), nil
	case "DIVFROM":
		castedRaw, ok := raw.(float64)
		if !ok {
			return nil, fmt.Errorf("cannot cast raw data to float")
		}
		return reducer.Args.(float64) / castedRaw, nil
	default:
		return nil, fmt.Errorf("unknown reducer function: %s", reducer.Function)
	}

}
