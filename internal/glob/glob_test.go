package glob

import "testing"

func TestMatchAnyPatternSkipGlobs(t *testing.T) {
	if !MatchAnyPattern("pkg/api.pb.go", []string{"*.pb.go"}) {
		t.Fatal("expected *.pb.go to match basename in a subdirectory")
	}
	if MatchAnyPattern("pkg/api.go", []string{"*.pb.go"}) {
		t.Fatal("did not expect *.pb.go to match api.go")
	}
}
