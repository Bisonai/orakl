package reducer

import (
	"math"
	"strconv"

	errorSentinel "bisonai.com/orakl/node/pkg/error"
)

type Reducer struct {
	Function string      `json:"function"`
	Args     interface{} `json:"args"`
}

func Reduce(raw interface{}, reducers []Reducer) (float64, error) {
	var result float64
	var err error
	for _, reducer := range reducers {
		raw, err = reduce(raw, reducer)
		if err != nil {
			return 0, err
		}
	}
	result, ok := raw.(float64)
	if !ok {
		return 0, errorSentinel.ErrReducerCastToFloatFail
	}
	return result, nil
}

func reduce(raw interface{}, reducer Reducer) (interface{}, error) {
	switch reducer.Function {
	case "INDEX":
		castedRaw, ok := raw.([]interface{})
		if !ok {
			return nil, errorSentinel.ErrReducerIndexCastToInterfaceFail
		}
		index, err := tryParseFloat(reducer.Args)
		if err != nil {
			return nil, err
		}

		return castedRaw[int(index)], nil
	case "PARSE":
		args, ok := reducer.Args.([]interface{})
		if !ok {
			return nil, errorSentinel.ErrReducerParseCastToInterfaceFail
		}
		argStrs := make([]string, len(args))
		for i, arg := range args {
			argStr, ok := arg.(string)
			if !ok {
				return nil, errorSentinel.ErrReducerParseCastToStringFail
			}
			argStrs[i] = argStr
		}
		for _, arg := range argStrs {
			castedRaw, ok := raw.(map[string]interface{})
			if !ok {
				return nil, errorSentinel.ErrReducerParseCastToMapFail
			}
			raw = castedRaw[arg]
		}
		return raw, nil
	case "MUL":
		castedRaw, err := tryParseFloat(raw)
		if err != nil {
			return nil, err
		}
		arg, ok := reducer.Args.(float64)
		if !ok {
			return nil, errorSentinel.ErrReducerMulCastToFloatFail
		}

		return castedRaw * arg, nil
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
		arg, ok := reducer.Args.(float64)
		if !ok {
			return nil, errorSentinel.ErrReducerDivCastToFloatFail
		}
		if arg == 0 {
			return nil, errorSentinel.ErrReducerDivDivsionByZero
		}
		return castedRaw / arg, nil
	case "DIVFROM":
		castedRaw, err := tryParseFloat(raw)
		if err != nil {
			return nil, err
		}
		arg, ok := reducer.Args.(float64)
		if !ok {
			return nil, errorSentinel.ErrReducerDivFromCastToFloatFail
		}
		return arg / castedRaw, nil
	default:
		return nil, errorSentinel.ErrReducerUnknownReducerFunc
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
	return 0, errorSentinel.ErrReducerCastToFloatFail
}
