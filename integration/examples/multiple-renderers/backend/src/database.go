package main

import (
	"context"
)

type database interface {
	entries(context.Context) ([]guestbookEntry, error)
	addEntry(context.Context, guestbookEntry) error
}
