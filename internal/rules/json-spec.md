# transaction rule json spec

## overview

transaction rules use json to define conditions for auto-categorizing transactions. rules are logic + conditions.

## schema

```json
{
  "logic": "AND|OR",
  "conditions": [
    {
      "field": "string",
      "operator": "string",
      "value": "string|number",
      "values": ["string", ...],
      "min_value": number,
      "max_value": number,
      "case_sensitive": boolean
    }
  ]
}
```

## logic

- "AND" = all conditions true
- "OR" = any condition true

## fields

- merchant
- tx_desc
- tx_direction (0=unknown, 1=credit, 2=debit)
- account_type
- account_name
- bank
- currency
- amount (number only)

## operators

string:

- equals
- not_equals
- contains
- not_contains
- starts_with
- ends_with
- contains_any (needs values[])
- regex
number:
- equals
- not_equals
- greater_than
- less_than
- between (needs min_value, max_value)

## values

- value: for most operators
- values: for contains_any
- min_value, max_value: for between
- case_sensitive: optional, string only, default false

## constraints

- amount: non-negative
- tx_direction: 0, 1, 2
- case_sensitive: string fields only
- min_value < max_value

## examples

merchant match:

```json
{
  "logic": "AND",
  "conditions": [
    {"field": "merchant", "operator": "contains", "value": "starbucks"}
  ]
}
```

amount + currency:

```json
{
  "logic": "AND",
  "conditions": [
    {"field": "amount", "operator": "between", "min_value": 50, "max_value": 200},
    {"field": "currency", "operator": "equals", "value": "USD"}
  ]
}
```

multi merchant:

```json
{
  "logic": "OR",
  "conditions": [
    {"field": "merchant", "operator": "equals", "value": "Amazon"},
    {"field": "merchant", "operator": "regex", "value": "^AMZN.*"}
  ]
}
```

direction + amount:

```json
{
  "logic": "AND",
  "conditions": [
    {"field": "tx_direction", "operator": "equals", "value": 2},
    {"field": "amount", "operator": "greater_than", "value": 100}
  ]
}
```

multi currency:

```json
{
  "logic": "AND",
  "conditions": [
    {"field": "amount", "operator": "greater_than", "value": 1000},
    {"field": "currency", "operator": "contains_any", "values": ["USD", "EUR", "GBP"]},
    {"field": "tx_desc", "operator": "contains", "value": "transfer"}
  ]
}
```

## errors

- INVALID_JSON
- REQUIRED_FIELD
- INVALID_VALUE
- INVALID_FIELD
- INVALID_OPERATOR_FOR_FIELD
- CONFLICTING_FIELDS
- INVALID_RANGE
- INVALID_FIELD_FOR_TYPE
- INVALID_REGEX

## field rename

- merchant_name → merchant
- added: tx_desc, tx_direction
