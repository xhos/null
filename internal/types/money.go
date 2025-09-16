package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"google.golang.org/genproto/googleapis/type/money"
)

// Money embeds money.Money and implements database interfaces
type Money struct {
	money.Money
}

// Scan implements sql.Scanner for database reads
func (m *Money) Scan(src any) error {
	if src == nil {
		return nil
	}

	isBytes := func(v any) ([]byte, bool) {
		switch data := v.(type) {
		case []byte:
			return data, true
		case string:
			return []byte(data), true
		default:
			return nil, false
		}
	}

	jsonData, ok := isBytes(src)
	if !ok {
		return fmt.Errorf("cannot scan %T into Money", src)
	}

	return json.Unmarshal(jsonData, &m.Money)
}

// Value implements driver.Valuer for database writes
func (m *Money) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}

	// WHY: protobuf omits zero values, but we need all fields for consistency
	data := struct {
		CurrencyCode string `json:"currency_code"`
		Units        int64  `json:"units"`
		Nanos        int32  `json:"nanos"`
	}{
		CurrencyCode: m.CurrencyCode,
		Units:        m.Units,
		Nanos:        m.Nanos,
	}

	return json.Marshal(data)
}
