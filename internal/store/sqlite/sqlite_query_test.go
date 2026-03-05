package sqlite

import "testing"

func TestListActiveByCreatedDayQuery_MatchesPlanLiteral(t *testing.T) {
	expected := "SELECT id, title, status, created_day, due_day, done_day, abandoned_day\n" +
		" FROM tasks\n" +
		" WHERE status = 'active' AND created_day = ?\n" +
		" ORDER BY id ASC"

	if listActiveByCreatedDayQuery != expected {
		t.Fatalf("query mismatch:\nexpected:\n%s\n\ngot:\n%s", expected, listActiveByCreatedDayQuery)
	}
}
