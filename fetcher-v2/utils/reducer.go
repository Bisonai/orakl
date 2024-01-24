package utils

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
)

type Reducer struct {
	Args     interface{} `json:"args"`
	Function string      `json:"function"`
}

type ReducerFunc func(interface{}) (interface{}, error)

func Pipe(fns ...ReducerFunc) ReducerFunc {
	return func(x interface{}) (interface{}, error) {
		var err error
		for _, fn := range fns {
			x, err = fn(x)
			if err != nil {
				return nil, err
			}
		}
		return x, nil
	}
}

func BuildReducer(reducers []Reducer) ([]ReducerFunc, error) {
	reducerFuncs := make([]ReducerFunc, len(reducers))
	for i, reducer := range reducers {
		switch reducer.Function {
		case "PARSE":
			reducerFuncs[i] = parseFn(reducer.Args)
		case "MUL":
			reducerFuncs[i] = mulFn(reducer.Args)
		case "POW10":
			reducerFuncs[i] = pow10Fn(reducer.Args)
		case "ROUND":
			reducerFuncs[i] = roundFn(reducer.Args)
		case "INDEX":
			reducerFuncs[i] = indexFn(reducer.Args)
		case "DIV":
			reducerFuncs[i] = divFn(reducer.Args)
		case "DIV_FROM":
			reducerFuncs[i] = divFromFn(reducer.Args)
		default:
			return nil, fmt.Errorf("unknown reducer function: %s", reducer.Function)
		}
	}

	return reducerFuncs, nil

}

func parseFn(args interface{}) ReducerFunc {
	return func(obj interface{}) (interface{}, error) {
		argsInterface, _ok := args.([]interface{})
		if !_ok {
			return nil, errors.New("PARSE requires a string list of interface")
		}

		path := make([]string, len(argsInterface))
		for i, v := range argsInterface {
			str, ok := v.(string)
			if !ok {
				return nil, errors.New("PARSE requires a string list of string")

			}
			path[i] = str
		}

		var ok bool
		for _, key := range path {
			val := reflect.ValueOf(obj)
			if val.Kind() == reflect.Map {
				for _, k := range val.MapKeys() {
					if k.String() == key {
						obj = val.MapIndex(k).Interface()
						ok = true
						break
					}
				}
			}
			if !ok {
				return nil, errors.New("Missing key in JSON: " + key)
			}
		}
		return obj, nil
	}
}

func mulFn(args interface{}) ReducerFunc {
	return func(value interface{}) (interface{}, error) {
		factor, ok := args.(float64)
		if !ok {
			return nil, errors.New("MUL requires a number argument")
		}

		num, ok := value.(float64)
		if !ok {
			return nil, errors.New("MUL requires a number value")
		}

		return num * factor, nil
	}
}

func pow10Fn(args interface{}) ReducerFunc {
	return func(value interface{}) (interface{}, error) {
		power, ok := args.(float64)
		if !ok {
			return nil, errors.New("POW10 requires a number argument")
		}

		num, ok := value.(float64)
		if !ok {
			return nil, errors.New("POW10 requires a number value")
		}

		return num * math.Pow(10, power), nil
	}
}

func roundFn(args interface{}) ReducerFunc {
	return func(value interface{}) (interface{}, error) {
		num, ok := value.(float64)
		if !ok {
			return nil, errors.New("ROUND requires a number value")
		}

		return math.Round(num), nil
	}
}

func indexFn(args interface{}) ReducerFunc {
	return func(value interface{}) (interface{}, error) {
		indexFloat, ok := args.(float64)
		if !ok {
			return nil, errors.New("INDEX requires an float argument")
		}

		index := int(indexFloat)

		list, ok := value.([]interface{})
		if !ok {
			return nil, errors.New("INDEX requires a list value")
		}

		if index < 0 || index >= len(list) {
			return nil, errors.New("INDEX out of range")
		}

		return list[index], nil
	}
}

func divFn(args interface{}) ReducerFunc {
	return func(value interface{}) (interface{}, error) {
		divisor, ok := args.(float64)
		if !ok {
			return nil, errors.New("DIV requires a number argument")
		}

		num, ok := value.(float64)
		if !ok {
			return nil, errors.New("DIV requires a number value")
		}

		if divisor == 0 {
			return nil, errors.New("DIV division by zero")
		}

		return num / divisor, nil
	}
}

func divFromFn(args interface{}) ReducerFunc {
	return func(value interface{}) (interface{}, error) {
		divisor, ok := args.(float64)
		if !ok {
			return nil, errors.New("DIV requires a number argument")
		}

		num, ok := value.(float64)
		if !ok {
			return nil, errors.New("DIV requires a number value")
		}

		if num == 0 {
			return nil, errors.New("DIV division by zero")
		}

		return divisor / num, nil
	}
}

func convertToBigInt(i interface{}) (*big.Int, error) {
	bigInt := new(big.Int)

	switch v := i.(type) {
	case string:
		_, success := bigInt.SetString(v, 10)
		if !success {
			return nil, fmt.Errorf("failed to convert string to big.Int")
		}
	case int:
		bigInt.SetInt64(int64(v))
	case int64:
		bigInt.SetInt64(v)
	case float64:
		bigInt.SetInt64(int64(v))
	default:
		return nil, fmt.Errorf("unsupported type: %T", v)
	}

	return bigInt, nil
}
