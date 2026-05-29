package main

import "time"

type MembershipFee struct {
	ID       uint
	MemberID uint
	Year     uint
	PaidOn   time.Time
	Quantity uint // quantity in EUR cents
}
