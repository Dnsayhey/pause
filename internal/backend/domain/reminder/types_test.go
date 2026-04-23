package reminder

import (
	"errors"
	"testing"
)

func TestCreateInputNormalize(t *testing.T) {
	rest := " REST "
	input, err := CreateInput{
		Name:         "Eye",
		IntervalSec:  1200,
		BreakSec:     20,
		ReminderType: &rest,
	}.Normalize()
	if err != nil {
		t.Fatalf("Normalize() err=%v", err)
	}
	if input.Enabled == nil || !*input.Enabled {
		t.Fatalf("expected enabled default true")
	}
	if input.ReminderType == nil || *input.ReminderType != ReminderTypeRest {
		t.Fatalf("unexpected reminder type: %+v", input.ReminderType)
	}
}

func TestCreateInputNormalizeRejectsInvalidValues(t *testing.T) {
	notify := "notify"
	_, err := CreateInput{Name: " ", IntervalSec: 10, BreakSec: 1, ReminderType: &notify}.Normalize()
	if !errors.Is(err, ErrNameRequired) {
		t.Fatalf("name err=%v want=%v", err, ErrNameRequired)
	}

	badType := "stretch"
	_, err = CreateInput{Name: "Hydrate", IntervalSec: 10, BreakSec: 1, ReminderType: &badType}.Normalize()
	if !errors.Is(err, ErrTypeInvalid) {
		t.Fatalf("type err=%v want=%v", err, ErrTypeInvalid)
	}
}

func TestPatchNormalize(t *testing.T) {
	name := "Focus"
	notify := " notify "
	patch, err := Patch{ID: 12, Name: &name, ReminderType: &notify}.Normalize()
	if err != nil {
		t.Fatalf("Normalize() err=%v", err)
	}
	if patch.ReminderType == nil || *patch.ReminderType != ReminderTypeNotify {
		t.Fatalf("unexpected reminder type: %+v", patch.ReminderType)
	}
}

func TestIsRestReminderType(t *testing.T) {
	if !IsRestReminderType("rest") {
		t.Fatalf("expected rest reminder to be treated as rest")
	}
	if IsRestReminderType("notify") {
		t.Fatalf("expected notify reminder to be treated as notify")
	}
	if !IsRestReminderType("unknown") {
		t.Fatalf("unknown types should continue to default to rest semantics")
	}
}
