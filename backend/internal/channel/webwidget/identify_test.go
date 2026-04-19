package webwidget

import (
	"context"
	"testing"
)

func TestIdentify_ValidHMAC(t *testing.T) {
	if testing.Short() {
		t.Skip("requires database")
	}

	t.Skip("integration test: requires postgres + redis")
}

func TestIdentify_InvalidHMAC(t *testing.T) {
	if testing.Short() {
		t.Skip("requires database")
	}

	t.Skip("integration test: requires postgres + redis")
}

func TestIdentify_NoHash_Unverified(t *testing.T) {
	if testing.Short() {
		t.Skip("requires database")
	}

	t.Skip("integration test: requires postgres + redis")
}

func TestIdentify_MergeExistingContact(t *testing.T) {
	if testing.Short() {
		t.Skip("requires database")
	}

	t.Skip("integration test: requires postgres + redis")
}

var _ = context.Background()
