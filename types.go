package app

import "time"

type Date struct {
	Year  uint
	Month time.Month
	Day   uint
}

type ApiErrors = map[string][]string

type Adder interface {
	GetID() uint64
	Add() error
	Validate() ApiErrors
}

type Option func() error

type Command struct {
	Name     string
	Handler  func()
	Commands []Command
}
