package db

import (
	"time"
)

type ListOpts struct {
	// cursor for pagination
	CursorID   *int64
	CursorDate *time.Time

	// filtering
	Start             *time.Time
	End               *time.Time
	AccountIDs        []int64
	Categories        []string
	Direction         string
	MerchantSearch    string
	DescriptionSearch string
	AmountMin         *float64
	AmountMax         *float64
	Currency          string
	TimeOfDayStart    *string
	TimeOfDayEnd      *string

	// pagination limit
	Limit int
}
