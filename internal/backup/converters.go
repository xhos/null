package backup

import (
	"fmt"

	nullpb "null/internal/gen/null/v1"
)

func parseAccountType(s string) (nullpb.AccountType, error) {
	switch s {
	case "CHECKING", "checking", "CHEQUING", "chequing":
		return nullpb.AccountType_ACCOUNT_CHEQUING, nil
	case "SAVINGS", "savings":
		return nullpb.AccountType_ACCOUNT_SAVINGS, nil
	case "CREDIT_CARD", "credit_card":
		return nullpb.AccountType_ACCOUNT_CREDIT_CARD, nil
	case "INVESTMENT", "investment":
		return nullpb.AccountType_ACCOUNT_INVESTMENT, nil
	case "OTHER", "other":
		return nullpb.AccountType_ACCOUNT_OTHER, nil
	default:
		return 0, fmt.Errorf("unknown account type: %s", s)
	}
}

func formatAccountType(at nullpb.AccountType) string {
	switch at {
	case nullpb.AccountType_ACCOUNT_CHEQUING:
		return "CHECKING"
	case nullpb.AccountType_ACCOUNT_SAVINGS:
		return "SAVINGS"
	case nullpb.AccountType_ACCOUNT_CREDIT_CARD:
		return "CREDIT_CARD"
	case nullpb.AccountType_ACCOUNT_INVESTMENT:
		return "INVESTMENT"
	case nullpb.AccountType_ACCOUNT_OTHER:
		return "OTHER"
	default:
		return "OTHER"
	}
}

func parseTransactionDirection(s string) (nullpb.TransactionDirection, error) {
	switch s {
	case "INBOUND", "inbound", "IN", "INCOMING", "incoming":
		return nullpb.TransactionDirection_DIRECTION_INCOMING, nil
	case "OUTBOUND", "outbound", "OUT", "OUTGOING", "outgoing":
		return nullpb.TransactionDirection_DIRECTION_OUTGOING, nil
	default:
		return 0, fmt.Errorf("unknown transaction direction: %s", s)
	}
}

func formatTransactionDirection(td nullpb.TransactionDirection) string {
	switch td {
	case nullpb.TransactionDirection_DIRECTION_INCOMING:
		return "INBOUND"
	case nullpb.TransactionDirection_DIRECTION_OUTGOING:
		return "OUTBOUND"
	default:
		return "OUTBOUND"
	}
}
