package types

import (
	"errors"

	"google.golang.org/genproto/googleapis/type/money"
)

// constants for validation and calculations
const (
	nanosMin     int32 = -999_999_999
	nanosMax     int32 = 999_999_999
	nanosPerUnit int64 = 1_000_000_000
)

// errors for validation and currency checks
var (
	ErrInvalidValue        = errors.New("money amount is invalid (nanos out of range or signs mismatch)")
	ErrMismatchingCurrency = errors.New("mismatching currency codes for operation")
)

// IsValid checks if a money amount conforms to the required rules.
func IsValid(m *money.Money) bool {
	nanosAreInRange := m.Nanos >= nanosMin && m.Nanos <= nanosMax

	// the sign of nanos must match the sign of units, unless one is zero.
	// this is the core rule of the money type.
	signsAreConsistent := m.Units == 0 || m.Nanos == 0 || (m.Units > 0) == (m.Nanos > 0)

	return nanosAreInRange && signsAreConsistent
}

// AddMoney safely adds two money amounts, returning an error for invalid inputs.
func AddMoney(a, b *money.Money) (money.Money, error) {
	// check for invalid inputs first to clear our minds for the main logic.
	if !IsValid(a) || !IsValid(b) {
		return money.Money{}, ErrInvalidValue
	}

	// ensure we aren't mixing currencies, which is usually a bug.
	bothCurrenciesAreSet := a.CurrencyCode != "" && b.CurrencyCode != ""
	currenciesAreDifferent := a.CurrencyCode != b.CurrencyCode
	if bothCurrenciesAreSet && currenciesAreDifferent {
		return money.Money{}, ErrMismatchingCurrency
	}

	sumUnits := a.Units + b.Units
	sumNanos := int64(a.Nanos) + int64(b.Nanos) // use int64 to prevent overflow

	// carry over any nanos that rolled into a full unit.
	unitOverflow := sumNanos / nanosPerUnit
	finalNanos := sumNanos % nanosPerUnit
	finalUnits := sumUnits + unitOverflow

	// if the final units and nanos have conflicting signs, we need to
	// "borrow" from the units to make them consistent.
	resultIsPositive := finalUnits > 0 && finalNanos < 0
	resultIsNegative := finalUnits < 0 && finalNanos > 0

	if resultIsPositive {
		finalUnits--
		finalNanos += nanosPerUnit
	} else if resultIsNegative {
		finalUnits++
		finalNanos -= nanosPerUnit
	}

	finalCurrency := a.CurrencyCode
	if finalCurrency == "" {
		finalCurrency = b.CurrencyCode
	}

	return money.Money{
		CurrencyCode: finalCurrency,
		Units:        finalUnits,
		Nanos:        int32(finalNanos),
	}, nil
}

// IsZero returns true if the money value is exactly zero.
func IsZero(m *money.Money) bool {
	return m.Units == 0 && m.Nanos == 0
}

// IsPositive returns true if the money amount is greater than zero.
func IsPositive(m *money.Money) bool {
	hasPositiveUnits := m.Units > 0
	hasPositiveNanosAtZeroUnits := m.Units == 0 && m.Nanos > 0
	return hasPositiveUnits || hasPositiveNanosAtZeroUnits
}

// IsNegative returns true if the money amount is less than zero.
func IsNegative(m *money.Money) bool {
	hasNegativeUnits := m.Units < 0
	hasNegativeNanosAtZeroUnits := m.Units == 0 && m.Nanos < 0
	return hasNegativeUnits || hasNegativeNanosAtZeroUnits
}

// AreEquals returns true if two money values are identical.
func AreEquals(a, b *money.Money) bool {
	return a.CurrencyCode == b.CurrencyCode && a.Units == b.Units && a.Nanos == b.Nanos
}

// Negate returns a money amount with the opposite sign.
func Negate(m *money.Money) money.Money {
	return money.Money{
		CurrencyCode: m.CurrencyCode,
		Units:        -m.Units,
		Nanos:        -m.Nanos,
	}
}
