package auth

import "testing"

func TestBaseHandler_Construct(t *testing.T) {
	if NewBaseHandler() == nil {
		t.Fatal("expected non-nil base handler")
	}
}

