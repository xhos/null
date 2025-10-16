package backup

import (
	arian "ariand/internal/gen/arian/v1"
	"fmt"
)

func parseAccountType(s string) (arian.AccountType, error) {
	switch s {
	case "CHECKING", "checking", "CHEQUING", "chequing":
		return arian.AccountType_ACCOUNT_CHEQUING, nil
	case "SAVINGS", "savings":
		return arian.AccountType_ACCOUNT_SAVINGS, nil
	case "CREDIT_CARD", "credit_card":
		return arian.AccountType_ACCOUNT_CREDIT_CARD, nil
	case "INVESTMENT", "investment":
		return arian.AccountType_ACCOUNT_INVESTMENT, nil
	case "OTHER", "other":
		return arian.AccountType_ACCOUNT_OTHER, nil
	default:
		return 0, fmt.Errorf("unknown account type: %s", s)
	}
}

func formatAccountType(at arian.AccountType) string {
	switch at {
	case arian.AccountType_ACCOUNT_CHEQUING:
		return "CHECKING"
	case arian.AccountType_ACCOUNT_SAVINGS:
		return "SAVINGS"
	case arian.AccountType_ACCOUNT_CREDIT_CARD:
		return "CREDIT_CARD"
	case arian.AccountType_ACCOUNT_INVESTMENT:
		return "INVESTMENT"
	case arian.AccountType_ACCOUNT_OTHER:
		return "OTHER"
	default:
		return "OTHER"
	}
}

func parseTransactionDirection(s string) (arian.TransactionDirection, error) {
	switch s {
	case "INBOUND", "inbound", "IN", "INCOMING", "incoming":
		return arian.TransactionDirection_DIRECTION_INCOMING, nil
	case "OUTBOUND", "outbound", "OUT", "OUTGOING", "outgoing":
		return arian.TransactionDirection_DIRECTION_OUTGOING, nil
	default:
		return 0, fmt.Errorf("unknown transaction direction: %s", s)
	}
}

func formatTransactionDirection(td arian.TransactionDirection) string {
	switch td {
	case arian.TransactionDirection_DIRECTION_INCOMING:
		return "INBOUND"
	case arian.TransactionDirection_DIRECTION_OUTGOING:
		return "OUTBOUND"
	default:
		return "OUTBOUND"
	}
}
