package core

import "testing"

func TestRebindPostgres(t *testing.T) {
	query := "SELECT `users`.`email` FROM `users` WHERE `email` = ? AND note = '?' AND active = ?"
	want := `SELECT "users"."email" FROM "users" WHERE "email" = $1 AND note = '?' AND active = $2`
	if got := rebindPostgres(query); got != want {
		t.Fatalf("rebindPostgres() = %q, want %q", got, want)
	}
}

func TestNormalizeDatabaseDriver(t *testing.T) {
	for _, alias := range []string{"postgres", "postgresql", "pgx"} {
		if got := normalizeDatabaseDriver(alias); got != "postgres" {
			t.Fatalf("normalizeDatabaseDriver(%q) = %q", alias, got)
		}
	}
}
