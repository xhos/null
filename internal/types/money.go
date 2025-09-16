package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"google.golang.org/genproto/googleapis/type/money"
)

// MoneyWrapper wraps money.Money to implement sql.Scanner and driver.Valuer
type MoneyWrapper struct {
	*money.Money
}

// NewMoneyWrapper creates a new MoneyWrapper
func NewMoneyWrapper(m *money.Money) *MoneyWrapper {
	return &MoneyWrapper{Money: m}
}

// Scan implements sql.Scanner for MoneyWrapper
func (m *MoneyWrapper) Scan(src interface{}) error {
	if src == nil {
		m.Money = nil
		return nil
	}
	bytes, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into Money", src)
	}
	m.Money = &money.Money{}
	return json.Unmarshal(bytes, m.Money)
}

// Value implements driver.Valuer for MoneyWrapper
func (m *MoneyWrapper) Value() (driver.Value, error) {
	if m.Money == nil {
		return nil, nil
	}
	return json.Marshal(m.Money)
}

// UnwrapMoney extracts the money.Money from MoneyWrapper
func (m *MoneyWrapper) UnwrapMoney() *money.Money {
	if m == nil {
		return nil
	}
	return m.Money
}

// WrapMoney creates a MoneyWrapper from money.Money
func WrapMoney(m *money.Money) *MoneyWrapper {
	if m == nil {
		return nil
	}
	return &MoneyWrapper{Money: m}
}
