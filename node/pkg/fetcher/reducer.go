package fetcher

import (
	"fmt"
	"math"
	"strconv"
)

func ReduceAll(raw interface{}, reducers []Reducer) (float64, error) {
	var result float64
	for _, reducer := range reducers {
		var err error
		tmp, err := reduce(raw, reducer)
		if err != nil {
			return 0, err
		}
		raw = tmp
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
		index, err := tryParseFloat(reducer.Args)
		if err != nil {
			return nil, err
		}

		return castedRaw[int(index)], nil
	case "PARSE", "PATH":

		args, ok := reducer.Args.([]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot cast reducer.Args to []interface{}")
		}
		argStrs := make([]string, len(args))
		for i, arg := range args {
			argStr, ok := arg.(string)
			if !ok {
				return nil, fmt.Errorf("cannot cast arg to string")
			}
			argStrs[i] = argStr
		}
		for _, arg := range argStrs {
			castedRaw, ok := raw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("cannot cast raw data to map[string]interface{}")
			}
			raw = castedRaw[arg]
		}
		return raw, nil
	case "MUL":
		castedRaw, err := tryParseFloat(raw)
		if err != nil {
			return nil, err
		}

		return castedRaw * reducer.Args.(float64), nil
	case "POW10":
		castedRaw, err := tryParseFloat(raw)
		if err != nil {
			return nil, err
		}
		arg, err := tryParseFloat(reducer.Args)
		if err != nil {
			return nil, err
		}

		return float64(math.Pow10(int(arg))) * castedRaw, nil
	case "ROUND":
		castedRaw, err := tryParseFloat(raw)
		if err != nil {
			return nil, err
		}
		return math.Round(castedRaw), nil
	case "DIV":
		castedRaw, err := tryParseFloat(raw)
		if err != nil {
			return nil, err
		}
		return castedRaw / reducer.Args.(float64), nil
	case "DIVFROM":
		castedRaw, err := tryParseFloat(raw)
		if err != nil {
			return nil, err
		}
		return reducer.Args.(float64) / castedRaw, nil
	default:
		return nil, fmt.Errorf("unknown reducer function: %s", reducer.Function)
	}

}

// numbers in raw json string are parsed as float64 from golang
func tryParseFloat(raw interface{}) (float64, error) {
	f, ok := raw.(float64)
	if ok {
		return f, nil
	}
	s, ok := raw.(string)
	if ok {
		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			return f, nil
		}
	}
	return 0, fmt.Errorf("cannot parse raw data to float")
}
