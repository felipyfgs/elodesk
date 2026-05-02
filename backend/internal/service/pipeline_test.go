package service

import (
	"testing"

	"backend/internal/model"
)

func TestListTemplates_HasFourEntries(t *testing.T) {
	got := ListTemplates()
	if len(got) != 4 {
		t.Fatalf("got %d templates, want 4", len(got))
	}
	wantKeys := map[string]bool{"sales-crm": true, "support": true, "tasks": true, "blank": true}
	for _, tpl := range got {
		if !wantKeys[tpl.Key] {
			t.Errorf("unexpected template key %q", tpl.Key)
		}
		delete(wantKeys, tpl.Key)
	}
	if len(wantKeys) != 0 {
		t.Errorf("missing template keys: %v", wantKeys)
	}
}

func TestGetTemplate_SalesCRMHasSixStagesAndIsContact(t *testing.T) {
	tpl, ok := GetTemplate("sales-crm")
	if !ok {
		t.Fatal("expected sales-crm to exist")
	}
	if tpl.CardKind != "contact" {
		t.Errorf("CardKind: got %q, want %q", tpl.CardKind, "contact")
	}
	if len(tpl.Stages) != 6 {
		t.Errorf("Stages: got %d, want 6", len(tpl.Stages))
	}
	wantNames := []string{"Lead", "Qualificado", "Proposta", "Negociação", "Ganho", "Perdido"}
	for i, want := range wantNames {
		if tpl.Stages[i].Name != want {
			t.Errorf("stage %d: got %q, want %q", i, tpl.Stages[i].Name, want)
		}
	}
	if !tpl.Stages[4].IsTerminal || tpl.Stages[4].TerminalKind == nil || *tpl.Stages[4].TerminalKind != "won" {
		t.Errorf("stage Ganho should be terminal kind=won")
	}
}

func TestGetTemplate_SupportIsConversation(t *testing.T) {
	tpl, ok := GetTemplate("support")
	if !ok {
		t.Fatal("expected support to exist")
	}
	if tpl.CardKind != "conversation" {
		t.Errorf("CardKind: got %q, want %q", tpl.CardKind, "conversation")
	}
}

func TestGetTemplate_UnknownReturnsFalse(t *testing.T) {
	if _, ok := GetTemplate("does-not-exist"); ok {
		t.Error("expected ok=false for unknown template")
	}
}

func TestValidateLink_Free(t *testing.T) {
	if err := validateLink(model.CardKindFree, nil, nil); err != nil {
		t.Errorf("free + no link: unexpected err %v", err)
	}
	t1, id1 := "contact", int64(42)
	if err := validateLink(model.CardKindFree, &t1, &id1); err == nil {
		t.Error("free + link should return ErrLinkedEntityForbidden")
	}
}

func TestValidateLink_Contact(t *testing.T) {
	if err := validateLink(model.CardKindContact, nil, nil); err == nil {
		t.Error("contact without link should error")
	}
	t1, id1 := "conversation", int64(1)
	if err := validateLink(model.CardKindContact, &t1, &id1); err == nil {
		t.Error("contact with conversation link should error")
	}
	t2, id2 := "contact", int64(7)
	if err := validateLink(model.CardKindContact, &t2, &id2); err != nil {
		t.Errorf("contact with contact link: unexpected err %v", err)
	}
}

func TestValidateLink_Conversation(t *testing.T) {
	t1, id1 := "conversation", int64(99)
	if err := validateLink(model.CardKindConversation, &t1, &id1); err != nil {
		t.Errorf("conversation with conversation link: unexpected err %v", err)
	}
	t2, id2 := "contact", int64(99)
	if err := validateLink(model.CardKindConversation, &t2, &id2); err == nil {
		t.Error("conversation with contact link should error")
	}
}

func TestNeedsRebalance(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want bool
	}{
		{"empty", []float64{}, false},
		{"single", []float64{100}, false},
		{"healthy gaps", []float64{100, 200, 300}, false},
		{"borderline gap", []float64{100, 100.001}, false},
		{"too tight", []float64{100, 100.00005}, true},
		{"two equal", []float64{100, 100}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := needsRebalance(c.in); got != c.want {
				t.Errorf("needsRebalance(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
