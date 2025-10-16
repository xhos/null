package backup

import (
	"time"

	"google.golang.org/genproto/googleapis/type/money"
)

type Backup struct {
	Version      string            `json:"version"`
	ExportedAt   time.Time         `json:"exported_at"`
	Categories   []CategoryData    `json:"categories"`
	Accounts     []AccountData     `json:"accounts"`
	Transactions []TransactionData `json:"transactions"`
	Rules        []RuleData        `json:"rules"`
}

type CategoryData struct {
	Slug  string `json:"slug"`
	Color string `json:"color"`
}

type AccountData struct {
	Name          string       `json:"name"`
	Bank          string       `json:"bank"`
	AccountType   string       `json:"account_type"`
	Alias         *string      `json:"alias,omitempty"`
	AnchorDate    *time.Time   `json:"anchor_date,omitempty"`
	AnchorBalance *money.Money `json:"anchor_balance"`
	MainCurrency  string       `json:"main_currency"`
	Colors        []string     `json:"colors,omitempty"`
}

type TransactionData struct {
	AccountName   string       `json:"account_name"`
	TxDate        time.Time    `json:"tx_date"`
	TxAmount      *money.Money `json:"tx_amount"`
	TxDirection   string       `json:"tx_direction"`
	TxDesc        *string      `json:"tx_desc,omitempty"`
	BalanceAfter  *money.Money `json:"balance_after,omitempty"`
	Merchant      *string      `json:"merchant,omitempty"`
	CategorySlug  *string      `json:"category_slug,omitempty"`
	UserNotes     *string      `json:"user_notes,omitempty"`
	ForeignAmount *money.Money `json:"foreign_amount,omitempty"`
	ExchangeRate  *float64     `json:"exchange_rate,omitempty"`
}

type RuleData struct {
	RuleName      string                 `json:"rule_name"`
	CategorySlug  *string                `json:"category_slug,omitempty"`
	Merchant      *string                `json:"merchant,omitempty"`
	Conditions    map[string]interface{} `json:"conditions"`
	IsActive      *bool                  `json:"is_active,omitempty"`
	PriorityOrder *int32                 `json:"priority_order,omitempty"`
	RuleSource    *string                `json:"rule_source,omitempty"`
}
