package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// float8 in postgresql
// json return type: float
type CustomFloat float64

// boolean in postgresql
// json return type: boolean
type CustomBool bool

// int4 in postgresql
// json return type: number
type CustomInt32 int32

// int8 and bigint in postgresql
// json return type: string
type CustomInt64 int64

type CustomDateTime struct {
	time.Time
}

const RFC3339Milli = "2006-01-02T15:04:05.000Z07:00"

func (cf *CustomFloat) MarshalJSON() ([]byte, error) {
	return json.Marshal(*cf)
}

func (cf *CustomFloat) UnmarshalJSON(data []byte) error {
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	switch v := value.(type) {
	case float64:
		*cf = CustomFloat(v)
	case float32:
		*cf = CustomFloat(float64(v))
	case int:
		*cf = CustomFloat(float64(v))
	case string:
		converted, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		*cf = CustomFloat(converted)
	default:
		return fmt.Errorf("unexpected type for CustomFloat: %T", value)

	}
	return nil
}

func (cb *CustomBool) MarshalJSON() ([]byte, error) {
	return json.Marshal(*cb)
}

func (cb *CustomBool) UnmarshalJSON(data []byte) error {
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	switch v := value.(type) {
	case bool:
		*cb = CustomBool(v)
	case string:
		converted, err := strconv.ParseBool(v)
		if err != nil {
			return err
		}
		*cb = CustomBool(converted)
	default:
		return fmt.Errorf("unexpected type for CustomBoolean: %T", value)
	}
	return nil
}

func (ci_32 *CustomInt32) MarshalJSON() ([]byte, error) {
	return json.Marshal(*ci_32)
}

func (ci_32 *CustomInt32) UnmarshalJSON(data []byte) error {
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	switch v := value.(type) {
	case int32:
		*ci_32 = CustomInt32(v)
	case int64:
		*ci_32 = CustomInt32(int32(v))
	case int:
		*ci_32 = CustomInt32(int32(v))
	case float64:
		*ci_32 = CustomInt32(int32(v))
	case float32:
		*ci_32 = CustomInt32(int32(v))
	case string:
		if v == "" {
			*ci_32 = CustomInt32(0)
		} else {
			converted, err := strconv.Atoi(v)
			if err != nil {
				return err
			}
			*ci_32 = CustomInt32(int32(converted))
		}

	default:
		return fmt.Errorf("unexpected type for customInt32: %T", value)
	}
	return nil
}

func (ci_64 CustomInt64) String() string {
	return strconv.FormatInt(int64(ci_64), 10)
}

func (ci_64 *CustomInt64) MarshalJSON() ([]byte, error) {
	return json.Marshal(ci_64.String())
}

func (ci_64 *CustomInt64) UnmarshalJSON(data []byte) error {
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	switch v := value.(type) {
	case int64:
		*ci_64 = CustomInt64(v)
	case int32:
		*ci_64 = CustomInt64(int64(v))
	case int:
		*ci_64 = CustomInt64(int64(v))
	case float64:
		*ci_64 = CustomInt64(int64(v))
	case float32:
		*ci_64 = CustomInt64(int64(v))
	case string:
		if v == "" {
			*ci_64 = CustomInt64(0)
		} else {
			converted, err := strconv.Atoi(v)
			if err != nil {
				return err
			}
			*ci_64 = CustomInt64(converted)
		}

	default:
		return fmt.Errorf("unexpected type for CustomInt64: %T", value)
	}
	return nil
}

func (cdt CustomDateTime) String() string {
	utcTime := cdt.Time.UTC()
	return utcTime.Format(RFC3339Milli)
}

func (cdt *CustomDateTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(cdt.String())
}

func (cdt *CustomDateTime) Scan(src interface{}) error {
	switch v := src.(type) {
	case time.Time:
		cdt.Time = v
	case string:
		v = strings.Replace(v, "GMT", "UTC", -1)

		if err := tryParsingRFC3339Milli(v, cdt); err != nil {
			if err := tryParsingRFC3339(v, cdt); err != nil {
				return fmt.Errorf("unexpected dateTime format: %s", v)
			}
		}
	default:
		return fmt.Errorf("unexpected type for CustomDateTime: %T", src)
	}
	return nil
}

func (cdt *CustomDateTime) UnmarshalJSON(data []byte) error {
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	switch v := value.(type) {
	case time.Time:
		cdt.Time = v
	case string:
		v = strings.Replace(v, "GMT", "UTC", -1)

		if err := tryParsingRFC3339Milli(v, cdt); err != nil {
			if err := tryParsingRFC3339(v, cdt); err != nil {
				return fmt.Errorf("unexpected dateTime format: %s", v)
			}
		}
	default:
		return fmt.Errorf("unexpected type for CustomDateTime: %T", value)
	}
	return nil
}

// Recommended dateTime format which matches output format
func tryParsingRFC3339Milli(v string, cdt *CustomDateTime) error {
	converted, err := time.Parse(RFC3339Milli, v)
	if err == nil {
		cdt.Time = converted
	}
	return err
}

func tryParsingRFC3339(v string, cdt *CustomDateTime) error {
	converted, err := time.Parse(time.RFC3339, v)
	if err == nil {
		cdt.Time = converted
	}
	return err
}
