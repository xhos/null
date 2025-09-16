package types

import (
	"encoding/json"
	"testing"

	"google.golang.org/genproto/googleapis/type/money"
)

// assertMoneyJSON is a test helper that checks if a Money object
// serializes to the correct JSON value.
func assertMoneyJSON(t *testing.T, m *Money, expectedCurrency string, expectedUnits int64, expectedNanos int32) {
	// this is what the database driver would do.
	jsonValue, err := m.Value()
	if err != nil {
		t.Fatalf("Value() returned an unexpected error: %v", err)
	}
	jsonBytes, ok := jsonValue.([]byte)
	if !ok {
		t.Fatalf("Value() did not return []byte; got %T", jsonValue)
	}

	// unmarshal to a generic map to inspect the fields.
	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON from Value(): %v", err)
	}

	// check that all expected fields are present.
	_, currencyPresent := result["currency_code"]
	_, unitsPresent := result["units"]
	_, nanosPresent := result["nanos"]
	if !currencyPresent || !unitsPresent || !nanosPresent {
		t.Errorf("JSON is missing required fields: got %s", jsonBytes)
	}

	// numbers are decoded as float64 by default when unmarshaling to map[string]any.
	// we must cast our expectations to match.
	currencyMatches := result["currency_code"] == expectedCurrency
	unitsMatch := result["units"] == float64(expectedUnits)
	nanosMatch := result["nanos"] == float64(expectedNanos)

	if !currencyMatches || !unitsMatch || !nanosMatch {
		t.Errorf("JSON value mismatch: got %v, want %s/%d/%d", result, expectedCurrency, expectedUnits, expectedNanos)
	}
}

func TestMoney_Value_ZeroValues(t *testing.T) {
	zeroCAD := &Money{
		Money: money.Money{
			CurrencyCode: "CAD",
			Units:        0,
			Nanos:        0,
		},
	}
	assertMoneyJSON(t, zeroCAD, "CAD", 0, 0)
}

func TestMoney_Value_NonZeroValues(t *testing.T) {
	hundredFiftyUSD := &Money{
		Money: money.Money{
			CurrencyCode: "USD",
			Units:        100,
			Nanos:        500_000_000,
		},
	}
	assertMoneyJSON(t, hundredFiftyUSD, "USD", 100, 500_000_000)
}

func TestMoney_Value_Nil(t *testing.T) {
	var nilMoney *Money
	value, err := nilMoney.Value()
	if err != nil {
		t.Fatalf("Value() on nil receiver returned an error: %v", err)
	}

	isNotNil := value != nil
	if isNotNil {
		t.Errorf("Expected nil value for nil receiver, got %v", value)
	}
}

func TestMoney_Scan_StringInput(t *testing.T) {
	jsonStr := `{"currency_code":"USD","units":100,"nanos":500000000}`
	m := &Money{}
	if err := m.Scan(jsonStr); err != nil {
		t.Fatalf("Scan() failed: %v", err)
	}

	scannedCorrectly := m.CurrencyCode == "USD" && m.Units == 100 && m.Nanos == 500_000_000
	if !scannedCorrectly {
		t.Errorf("Scan resulted in wrong values: got %+v", m)
	}
}

func TestMoney_Scan_BytesInput(t *testing.T) {
	jsonBytes := []byte(`{"currency_code":"CAD","units":0,"nanos":0}`)
	m := &Money{}
	if err := m.Scan(jsonBytes); err != nil {
		t.Fatalf("Scan() failed: %v", err)
	}

	scannedCorrectly := m.CurrencyCode == "CAD" && m.Units == 0 && m.Nanos == 0
	if !scannedCorrectly {
		t.Errorf("Scan resulted in wrong values: got %+v", m)
	}
}

func TestMoney_Scan_Nil(t *testing.T) {
	m := &Money{}
	if err := m.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) returned an unexpected error: %v", err)
	}
}

func TestMoney_Scan_InvalidType(t *testing.T) {
	m := &Money{}
	// test the defensive check against unsupported database types.
	err := m.Scan(12345)

	isNotError := err == nil
	if isNotError {
		t.Fatal("Scan() with an invalid type did not return an error")
	}
}
