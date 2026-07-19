package nativemsg

import (
	"context"
	"testing"

	"github.com/masahide/tabcli/internal/tools"
)

func TestValidateContentRevisionsSkipsEmptyReferences(t *testing.T) {
	bridge := ChromeBridge{}
	for _, references := range [][]tools.ContentReference{nil, {}} {
		if err := bridge.ValidateContentRevisions(context.Background(), references); err != nil {
			t.Fatalf("ValidateContentRevisions(%v) = %v, want nil", references, err)
		}
	}
}
