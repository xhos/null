package rules

import (
	"encoding/json"
	"testing"
)

func TestValidateRuleJSON_ValidRules(t *testing.T) {
	tests := []struct {
		name string
		rule string
	}{
		{
			name: "Coffee shop rule",
			rule: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "merchant",
						"operator": "contains_any",
						"values": ["starbucks", "dunkin", "peet's coffee"],
						"case_sensitive": false
					},
					{
						"field": "amount",
						"operator": "between",
						"min_value": 2.00,
						"max_value": 25.00
					}
				]
			}`,
		},
		{
			name: "Rent payment rule",
			rule: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "amount",
						"operator": "equals",
						"value": 1200.00
					},
					{
						"field": "merchant",
						"operator": "contains",
						"value": "property management",
						"case_sensitive": false
					}
				]
			}`,
		},
		{
			name: "OR logic example",
			rule: `{
				"logic": "OR",
				"conditions": [
					{
						"field": "bank",
						"operator": "equals",
						"value": "Chase"
					},
					{
						"field": "bank",
						"operator": "equals",
						"value": "Wells Fargo"
					}
				]
			}`,
		},
		{
			name: "Regex example",
			rule: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "merchant",
						"operator": "regex",
						"value": "^AMZN.*",
						"case_sensitive": false
					}
				]
			}`,
		},
		{
			name: "Amount greater than",
			rule: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "amount",
						"operator": "greater_than",
						"value": 50.00
					}
				]
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateRuleJSON([]byte(tt.rule))
			if !result.Valid {
				t.Errorf("Expected rule to be valid, got errors: %v", result.Errors)
			}
		})
	}
}

func TestValidateRuleJSON_InvalidRules(t *testing.T) {
	tests := []struct {
		name           string
		rule           string
		expectedErrors []string
	}{
		{
			name: "Missing logic",
			rule: `{
				"conditions": [
					{
						"field": "amount",
						"operator": "equals",
						"value": 100
					}
				]
			}`,
			expectedErrors: []string{"Logic is required"},
		},
		{
			name: "Invalid logic operator",
			rule: `{
				"logic": "XOR",
				"conditions": [
					{
						"field": "amount",
						"operator": "equals",
						"value": 100
					}
				]
			}`,
			expectedErrors: []string{"Logic must be 'AND' or 'OR'"},
		},
		{
			name: "No conditions",
			rule: `{
				"logic": "AND",
				"conditions": []
			}`,
			expectedErrors: []string{"At least one condition is required"},
		},
		{
			name: "Invalid field",
			rule: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "invalid_field",
						"operator": "equals",
						"value": "test"
					}
				]
			}`,
			expectedErrors: []string{"Invalid field: invalid_field"},
		},
		{
			name: "String operator on numeric field",
			rule: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "amount",
						"operator": "contains",
						"value": "test"
					}
				]
			}`,
			expectedErrors: []string{"Operator 'contains' is not valid for numeric field 'amount'"},
		},
		{
			name: "Numeric operator on string field",
			rule: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "merchant",
						"operator": "greater_than",
						"value": "100"
					}
				]
			}`,
			expectedErrors: []string{"Operator 'greater_than' is not valid for string field 'merchant'"},
		},
		{
			name: "Missing value for regular operator",
			rule: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "merchant",
						"operator": "equals"
					}
				]
			}`,
			expectedErrors: []string{"Operator 'equals' requires 'value'"},
		},
		{
			name: "Missing values for contains_any",
			rule: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "merchant",
						"operator": "contains_any"
					}
				]
			}`,
			expectedErrors: []string{"Operator 'contains_any' requires 'values' array"},
		},
		{
			name: "Missing min/max for between",
			rule: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "amount",
						"operator": "between",
						"value": 100
					}
				]
			}`,
			expectedErrors: []string{"Operator 'between' requires 'min_value'", "Operator 'between' requires 'max_value'"},
		},
		{
			name: "Invalid range for between",
			rule: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "amount",
						"operator": "between",
						"min_value": 100,
						"max_value": 50
					}
				]
			}`,
			expectedErrors: []string{"min_value must be less than max_value"},
		},
		{
			name: "Case sensitive on numeric field",
			rule: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "amount",
						"operator": "equals",
						"value": 100,
						"case_sensitive": true
					}
				]
			}`,
			expectedErrors: []string{"case_sensitive only applies to string fields"},
		},
		{
			name: "Currency property not supported",
			rule: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "merchant",
						"operator": "equals",
						"value": "test",
						"currency": "USD"
					}
				]
			}`,
			expectedErrors: []string{"currency property is not supported"},
		},
		{
			name: "Invalid regex pattern",
			rule: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "merchant",
						"operator": "regex",
						"value": "[invalid"
					}
				]
			}`,
			expectedErrors: []string{"Invalid regex pattern"},
		},
		{
			name: "Conflicting value fields for contains_any",
			rule: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "merchant",
						"operator": "contains_any",
						"value": "test",
						"values": ["test1", "test2"]
					}
				]
			}`,
			expectedErrors: []string{"Operator 'contains_any' should use 'values' not 'value'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateRuleJSONDetailed([]byte(tt.rule))
			if result.Valid {
				t.Errorf("Expected rule to be invalid")
			}

			for _, expectedError := range tt.expectedErrors {
				found := false
				for _, actualError := range result.Errors {
					if contains(actualError.Message, expectedError) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing '%s', got errors: %v", expectedError, result.Errors)
				}
			}
		})
	}
}

func TestNormalizeAndValidateRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Normalize case insensitive string",
			input: `{
				"logic": "and",
				"conditions": [
					{
						"field": "merchant",
						"operator": "equals",
						"value": "STARBUCKS"
					}
				]
			}`,
			expected: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "merchant",
						"operator": "equals",
						"value": "starbucks",
						"case_sensitive": false
					}
				]
			}`,
		},
		{
			name: "Preserve case sensitive string",
			input: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "merchant",
						"operator": "equals",
						"value": "STARBUCKS",
						"case_sensitive": true
					}
				]
			}`,
			expected: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "merchant",
						"operator": "equals",
						"value": "STARBUCKS",
						"case_sensitive": true
					}
				]
			}`,
		},
		{
			name: "Normalize values array",
			input: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "merchant",
						"operator": "contains_any",
						"values": ["STARBUCKS", "DUNKIN"]
					}
				]
			}`,
			expected: `{
				"logic": "AND",
				"conditions": [
					{
						"field": "merchant",
						"operator": "contains_any",
						"values": ["starbucks", "dunkin"],
						"case_sensitive": false
					}
				]
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, err := NormalizeAndValidateRule([]byte(tt.input))
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Parse expected result for comparison
			var expected RuleConditions
			if err := json.Unmarshal([]byte(tt.expected), &expected); err != nil {
				t.Errorf("Failed to parse expected result: %v", err)
				return
			}

			// Compare logic
			if normalized.Logic != expected.Logic {
				t.Errorf("Expected logic %s, got %s", expected.Logic, normalized.Logic)
			}

			// Compare conditions
			if len(normalized.Conditions) != len(expected.Conditions) {
				t.Errorf("Expected %d conditions, got %d", len(expected.Conditions), len(normalized.Conditions))
				return
			}

			for i, condition := range normalized.Conditions {
				expectedCondition := expected.Conditions[i]

				if condition.Field != expectedCondition.Field {
					t.Errorf("Condition %d: expected field %s, got %s", i, expectedCondition.Field, condition.Field)
				}

				if condition.Operator != expectedCondition.Operator {
					t.Errorf("Condition %d: expected operator %s, got %s", i, expectedCondition.Operator, condition.Operator)
				}

				// Compare values based on what's expected
				if expectedCondition.Value != nil {
					if condition.Value == nil || condition.Value != expectedCondition.Value {
						t.Errorf("Condition %d: expected value %v, got %v", i, expectedCondition.Value, condition.Value)
					}
				}

				if len(expectedCondition.Values) > 0 {
					if len(condition.Values) != len(expectedCondition.Values) {
						t.Errorf("Condition %d: expected %d values, got %d", i, len(expectedCondition.Values), len(condition.Values))
					} else {
						for j, val := range condition.Values {
							if val != expectedCondition.Values[j] {
								t.Errorf("Condition %d, value %d: expected %s, got %s", i, j, expectedCondition.Values[j], val)
							}
						}
					}
				}

				if expectedCondition.CaseSensitive != nil {
					if condition.CaseSensitive == nil || *condition.CaseSensitive != *expectedCondition.CaseSensitive {
						t.Errorf("Condition %d: expected case_sensitive %v, got %v", i, expectedCondition.CaseSensitive, condition.CaseSensitive)
					}
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
