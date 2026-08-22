package pack

import "testing"

func TestCanEditSlug(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{StatusDraft, true},
		{StatusPendingReview, true},
		{StatusAwaitingConfirm, true},
		{StatusRejected, true},
		{StatusPublished, false},
		{StatusRevoked, false},
	}

	for _, tt := range tests {
		got := CanEditSlug(tt.status)
		if got != tt.expected {
			t.Errorf("CanEditSlug(%s) = %v; want %v", tt.status, got, tt.expected)
		}
	}
}

func TestCanEditPack(t *testing.T) {
	p := Pack{
		ID:          "pack-1",
		SubmitterID: "user-1",
		Status:      StatusPublished,
	}

	// Owner contributor can edit
	if err := CanEditPack(p, "user-1", false); err != nil {
		t.Errorf("expected owner contributor to be able to edit, got %v", err)
	}

	// Non-owner contributor cannot edit
	if err := CanEditPack(p, "user-2", false); err != ErrForbidden {
		t.Errorf("expected non-owner contributor to be forbidden, got %v", err)
	}

	// Admin can edit even if not owner
	if err := CanEditPack(p, "user-admin", true); err != nil {
		t.Errorf("expected admin to be able to edit, got %v", err)
	}
}
