package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation(t *testing.T) {
	unique := &pgconn.PgError{Code: "23505", ConstraintName: "packs_slug_key", Message: "duplicate key value violates unique constraint"}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"unique violation", unique, true},
		{"unique violation wrapped", fmt.Errorf("insert draft: %w", unique), true},
		{"unique violation in chain", fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", unique)), true},
		{"not null violation", &pgconn.PgError{Code: "23502"}, false},
		{"foreign key violation", &pgconn.PgError{Code: "23503"}, false},
		{"plain error", errors.New("connection refused"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUniqueViolation(tt.err); got != tt.want {
				t.Errorf("isUniqueViolation(%v) = %v; want %v", tt.err, got, tt.want)
			}
		})
	}
}
