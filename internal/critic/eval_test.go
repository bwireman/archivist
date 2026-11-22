package critic

import (
	"context"
	"strings"
	"testing"
)

func TestLastingAccuracyChangesWhenNegativesIncluded(t *testing.T) {
	always := func(context.Context, string, string) (Result, error) {
		return Result{Type: "decision", Lasting: true, OK: true}, nil
	}
	records := []Item{
		{Title: "a", Body: "b", GoldType: "decision", GoldLasting: true, ScoreType: true},
		{Title: "c", Body: "d", GoldType: "decision", GoldLasting: true, ScoreType: true},
	}
	raw := strings.Join([]string{
		`{"class":"code","title":"sym","body":"a doc comment that lives only in code"}`,
		`{"class":"error","title":"once","body":"this session's command failed once"}`,
	}, "\n") + "\n"
	negs, err := LoadNegatives(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	without, err := Measure(context.Background(), records, always)
	if err != nil {
		t.Fatal(err)
	}
	with, err := Measure(context.Background(), append(records, negs...), always)
	if err != nil {
		t.Fatal(err)
	}
	if without.LastingAccuracy == with.LastingAccuracy {
		t.Fatalf("lasting accuracy unchanged: %v", with.LastingAccuracy)
	}
	if with.NLasting != without.NLasting+len(negs) {
		t.Fatalf("n_lasting %d want %d", with.NLasting, without.NLasting+len(negs))
	}
	if with.NType != without.NType {
		t.Fatalf("negatives scored on type: %d %d", with.NType, without.NType)
	}
}
