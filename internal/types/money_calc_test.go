package types

import (
	"testing"

	"google.golang.org/genproto/googleapis/type/money"
)

// helper to create a money object with currency.
func m(units int64, nanos int32, currency string) *money.Money {
	return &money.Money{Units: units, Nanos: nanos, CurrencyCode: currency}
}

// helper to create a usd money object.
func usd(units int64, nanos int32) *money.Money {
	return m(units, nanos, "USD")
}

// helper to create a cad money object.
func cad(units int64, nanos int32) *money.Money {
	return m(units, nanos, "CAD")
}

// helper to create a money object with no currency.
func noCurrency(units int64, nanos int32) *money.Money {
	return m(units, nanos, "")
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name string
		in   *money.Money
		want bool
	}{
		{"valid negative", noCurrency(-1, -500_000_000), true},
		{"valid positive", noCurrency(1, 500_000_000), true},
		{"valid zero units", noCurrency(0, -100), true},
		{"valid zero nanos", noCurrency(1, 0), true},
		{"invalid sign mismatch (-/+)", noCurrency(-1, 1), false},
		{"invalid sign mismatch (+/-)", noCurrency(1, -1), false},
		{"invalid nanos overflow", noCurrency(1, 1_000_000_000), false},
		{"invalid nanos underflow", noCurrency(1, -1_000_000_000), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValid(tt.in); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPositive(t *testing.T) {
	tests := []struct {
		name string
		in   *money.Money
		want bool
	}{
		{"positive units", noCurrency(1, 0), true},
		{"positive nanos", noCurrency(0, 1), true},
		{"zero", noCurrency(0, 0), false},
		{"negative", noCurrency(-1, -1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPositive(tt.in); got != tt.want {
				t.Errorf("IsPositive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNegative(t *testing.T) {
	tests := []struct {
		name string
		in   *money.Money
		want bool
	}{
		{"negative units", noCurrency(-1, 0), true},
		{"negative nanos", noCurrency(0, -1), true},
		{"zero", noCurrency(0, 0), false},
		{"positive", noCurrency(1, 1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNegative(tt.in); got != tt.want {
				t.Errorf("IsNegative() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAreEquals(t *testing.T) {
	tests := []struct {
		name string
		a, b *money.Money
		want bool
	}{
		{"equal values", usd(1, 50), usd(1, 50), true},
		{"different currency", usd(1, 50), cad(1, 50), false},
		{"different units", usd(1, 50), usd(2, 50), false},
		{"different nanos", usd(1, 50), usd(1, 51), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AreEquals(tt.a, tt.b); got != tt.want {
				t.Errorf("AreEquals() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNegate(t *testing.T) {
	tests := []struct {
		name string
		in   *money.Money
		want *money.Money
	}{
		{"positive", usd(1, 200), usd(-1, -200)},
		{"negative", usd(-5, -10), usd(5, 10)},
		{"zero", usd(0, 0), usd(0, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Negate(tt.in)
			if !AreEquals(&got, tt.want) {
				// pass pointers to Errorf to avoid copying the lock
				t.Errorf("Negate() = %v, want %v", &got, tt.want)
			}
		})
	}
}

func TestAddMoney(t *testing.T) {
	type args struct {
		a, b *money.Money
	}
	tests := []struct {
		name    string
		args    args
		want    *money.Money
		wantErr error
	}{
		// Basic Addition
		{"positive + positive", args{usd(1, 100), usd(2, 200)}, usd(3, 300), nil},
		{"negative + negative", args{usd(-1, -100), usd(-2, -200)}, usd(-3, -300), nil},

		// Nanos Carry-Over
		{"positive carry", args{usd(2, 800_000_000), usd(3, 700_000_000)}, usd(6, 500_000_000), nil},
		{"negative carry", args{usd(-2, -800_000_000), usd(-3, -700_000_000)}, usd(-6, -500_000_000), nil},

		// Normalization (Sign Correction)
		{"positive result with borrow", args{usd(5, 500_000_000), usd(-2, -800_000_000)}, usd(2, 700_000_000), nil},
		{"negative result with borrow", args{usd(-5, -500_000_000), usd(2, 800_000_000)}, usd(-2, -700_000_000), nil},
		{"zero result with borrow", args{usd(1, 500_000_000), usd(-1, -500_000_000)}, usd(0, 0), nil},

		// Currency Logic
		{"no currency + currency", args{noCurrency(1, 0), usd(2, 0)}, usd(3, 0), nil},
		{"currency + no currency", args{usd(1, 0), noCurrency(2, 0)}, usd(3, 0), nil},
		{"no currency + no currency", args{noCurrency(1, 0), noCurrency(2, 0)}, noCurrency(3, 0), nil},

		// Error Cases
		{"error on currency mismatch", args{usd(1, 0), cad(1, 0)}, &money.Money{}, ErrMismatchingCurrency},
		{"error on invalid input a", args{m(1, -100, "USD"), usd(1, 0)}, &money.Money{}, ErrInvalidValue},
		{"error on invalid input b", args{usd(1, 0), m(1, -100, "USD")}, &money.Money{}, ErrInvalidValue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AddMoney(tt.args.a, tt.args.b)

			if err != tt.wantErr {
				t.Errorf("AddMoney() error = %v, wantErr %v", err, tt.wantErr)
				return // if error is unexpected, no need to check value
			}

			if !AreEquals(&got, tt.want) {
				// pass pointers to Errorf to avoid copying the lock
				t.Errorf("AddMoney() got = %v, want %v", &got, tt.want)
			}
		})
	}
}

func TestIsZero(t *testing.T) {
	tests := []struct {
		name string
		in   *money.Money
		want bool
	}{
		{"is zero", noCurrency(0, 0), true},
		{"non-zero units", noCurrency(1, 0), false},
		{"non-zero nanos", noCurrency(0, 1), false},
		{"non-zero negative", noCurrency(-1, -1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsZero(tt.in); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}
