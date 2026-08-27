package audit

import (
	"testing"

	"github.com/henok321/knobel-manager-service/pkg/entity"
)

func gameFixture() entity.Game {
	return entity.Game{
		ID:             1,
		Name:           "Turnier",
		TeamSize:       4,
		TableSize:      4,
		NumberOfRounds: 2,
		Status:         entity.StatusSetup,
		Owners:         []*entity.GameOwner{{GameID: 1, OwnerSub: "owner-a"}},
		Teams: []*entity.Team{
			{
				ID: 10, Name: "Team A", GameID: 1,
				Players: []*entity.Player{
					{ID: 100, Name: "Anna", TeamID: 10},
				},
			},
		},
	}
}

func changeByKey(t *testing.T, changes []EntityChange, kind entity.AuditEntity, id string) EntityChange {
	t.Helper()

	for _, change := range changes {
		if change.Entity == kind && change.EntityID == id {
			return change
		}
	}

	t.Fatalf("no change for %s %s in %+v", kind, id, changes)

	return EntityChange{}
}

func value(t *testing.T, pointer *string) string {
	t.Helper()

	if pointer == nil {
		t.Fatal("expected a value, got nil")
	}

	return *pointer
}

func TestDiffNoChanges(t *testing.T) {
	before := flatten(gameFixture())
	after := flatten(gameFixture())

	if changes := diff(before, after); len(changes) != 0 {
		t.Fatalf("expected no changes, got %+v", changes)
	}
}

func TestDiffEmptyGameProducesNothing(t *testing.T) {
	if records := flatten(entity.Game{}); len(records) != 0 {
		t.Fatalf("expected empty snapshot for zero game, got %+v", records)
	}
}

func TestDiffUpdates(t *testing.T) {
	tests := map[string]struct {
		mutate   func(*entity.Game)
		kind     entity.AuditEntity
		id       string
		field    string
		wantFrom string
		wantTo   string
	}{
		"team renamed": {
			mutate:   func(g *entity.Game) { g.Teams[0].Name = "Die Knobelkoenige" },
			kind:     entity.AuditEntityTeam,
			id:       "10",
			field:    "name",
			wantFrom: "Team A",
			wantTo:   "Die Knobelkoenige",
		},
		"player renamed": {
			mutate:   func(g *entity.Game) { g.Teams[0].Players[0].Name = "Annika" },
			kind:     entity.AuditEntityPlayer,
			id:       "100",
			field:    "name",
			wantFrom: "Anna",
			wantTo:   "Annika",
		},
		"game status advanced": {
			mutate:   func(g *entity.Game) { g.Status = entity.StatusInProgress },
			kind:     entity.AuditEntityGame,
			id:       "1",
			field:    "status",
			wantFrom: "setup",
			wantTo:   "in_progress",
		},
		"game renamed": {
			mutate:   func(g *entity.Game) { g.Name = "Finale" },
			kind:     entity.AuditEntityGame,
			id:       "1",
			field:    "name",
			wantFrom: "Turnier",
			wantTo:   "Finale",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			before := flatten(gameFixture())

			mutated := gameFixture()
			test.mutate(&mutated)

			changes := diff(before, flatten(mutated))

			if len(changes) != 1 {
				t.Fatalf("expected exactly one change, got %+v", changes)
			}

			change := changeByKey(t, changes, test.kind, test.id)

			if change.Action != entity.AuditActionUpdate {
				t.Errorf("action = %q, want update", change.Action)
			}

			if len(change.Changes) != 1 {
				t.Fatalf("expected one field change, got %+v", change.Changes)
			}

			field := change.Changes[0]

			if field.Field != test.field {
				t.Errorf("field = %q, want %q", field.Field, test.field)
			}

			if got := value(t, field.From); got != test.wantFrom {
				t.Errorf("from = %q, want %q", got, test.wantFrom)
			}

			if got := value(t, field.To); got != test.wantTo {
				t.Errorf("to = %q, want %q", got, test.wantTo)
			}
		})
	}
}

func TestDiffCreate(t *testing.T) {
	before := flatten(gameFixture())

	after := gameFixture()
	after.Teams = append(after.Teams, &entity.Team{ID: 11, Name: "Team B", GameID: 1})

	changes := diff(before, flatten(after))

	if len(changes) != 1 {
		t.Fatalf("expected exactly one change, got %+v", changes)
	}

	change := changeByKey(t, changes, entity.AuditEntityTeam, "11")

	if change.Action != entity.AuditActionCreate {
		t.Errorf("action = %q, want create", change.Action)
	}

	if len(change.Changes) != 1 {
		t.Fatalf("expected one field change, got %+v", change.Changes)
	}

	if change.Changes[0].From != nil {
		t.Errorf("from = %v, want nil on create", *change.Changes[0].From)
	}

	if got := value(t, change.Changes[0].To); got != "Team B" {
		t.Errorf("to = %q, want %q", got, "Team B")
	}
}

func TestDiffDeleteCascadesToPlayers(t *testing.T) {
	before := flatten(gameFixture())

	after := gameFixture()
	after.Teams = nil

	changes := diff(before, flatten(after))

	if len(changes) != 2 {
		t.Fatalf("expected team and player deletions, got %+v", changes)
	}

	for _, expected := range []struct {
		kind entity.AuditEntity
		id   string
	}{
		{entity.AuditEntityTeam, "10"},
		{entity.AuditEntityPlayer, "100"},
	} {
		change := changeByKey(t, changes, expected.kind, expected.id)

		if change.Action != entity.AuditActionDelete {
			t.Errorf("%s %s action = %q, want delete", expected.kind, expected.id, change.Action)
		}

		for _, field := range change.Changes {
			if field.To != nil {
				t.Errorf("%s %s field %q has to = %q, want nil on delete", expected.kind, expected.id, field.Field, *field.To)
			}

			if field.From == nil {
				t.Errorf("%s %s field %q has no from value", expected.kind, expected.id, field.Field)
			}
		}
	}
}

func TestDiffOwnerAddedAndRemoved(t *testing.T) {
	base := gameFixture()

	withExtra := gameFixture()
	withExtra.Owners = append(withExtra.Owners, &entity.GameOwner{GameID: 1, OwnerSub: "owner-b"})

	added := diff(flatten(base), flatten(withExtra))
	if len(added) != 1 {
		t.Fatalf("expected one change on add, got %+v", added)
	}

	addedOwner := changeByKey(t, added, entity.AuditEntityOwner, "owner-b")

	if addedOwner.Action != entity.AuditActionCreate {
		t.Errorf("action = %q, want create", addedOwner.Action)
	}

	if len(addedOwner.Changes) != 0 {
		t.Errorf("owner events are presence only, got %+v", addedOwner.Changes)
	}

	removed := diff(flatten(withExtra), flatten(base))
	if len(removed) != 1 {
		t.Fatalf("expected one change on remove, got %+v", removed)
	}

	if action := changeByKey(t, removed, entity.AuditEntityOwner, "owner-b").Action; action != entity.AuditActionDelete {
		t.Errorf("action = %q, want delete", action)
	}
}

// Re-running setup wipes every score. Scores are in the diffed set precisely so that
// destruction leaves a trail, even though rounds and tables are excluded.
func TestDiffSetupWipesScores(t *testing.T) {
	withScores := gameFixture()
	withScores.Teams[0].Players[0].Scores = []*entity.Score{
		{ID: 500, PlayerID: 100, TableID: 7, Score: 42},
		{ID: 501, PlayerID: 100, TableID: 8, Score: 13},
	}

	changes := diff(flatten(withScores), flatten(gameFixture()))

	if len(changes) != 2 {
		t.Fatalf("expected two score deletions, got %+v", changes)
	}

	for _, id := range []string{"500", "501"} {
		change := changeByKey(t, changes, entity.AuditEntityScore, id)

		if change.Action != entity.AuditActionDelete {
			t.Errorf("score %s action = %q, want delete", id, change.Action)
		}
	}
}

func TestDiffScoreUpdated(t *testing.T) {
	before := gameFixture()
	before.Teams[0].Players[0].Scores = []*entity.Score{{ID: 500, PlayerID: 100, TableID: 7, Score: 0}}

	after := gameFixture()
	after.Teams[0].Players[0].Scores = []*entity.Score{{ID: 500, PlayerID: 100, TableID: 7, Score: 42}}

	changes := diff(flatten(before), flatten(after))

	if len(changes) != 1 {
		t.Fatalf("expected one change, got %+v", changes)
	}

	change := changeByKey(t, changes, entity.AuditEntityScore, "500")

	if len(change.Changes) != 1 || change.Changes[0].Field != "score" {
		t.Fatalf("expected only the score field to change, got %+v", change.Changes)
	}

	if got := value(t, change.Changes[0].From); got != "0" {
		t.Errorf("from = %q, want %q", got, "0")
	}

	if got := value(t, change.Changes[0].To); got != "42" {
		t.Errorf("to = %q, want %q", got, "42")
	}
}

// A fresh game has no before-snapshot at all, which is how POST /games is audited.
func TestDiffGameCreatedFromNothing(t *testing.T) {
	changes := diff(nil, flatten(gameFixture()))

	if len(changes) != 4 {
		t.Fatalf("expected game, owner, team and player creations, got %+v", changes)
	}

	for _, change := range changes {
		if change.Action != entity.AuditActionCreate {
			t.Errorf("%s %s action = %q, want create", change.Entity, change.EntityID, change.Action)
		}
	}
}

func TestDiffIsDeterministic(t *testing.T) {
	before := flatten(gameFixture())

	after := gameFixture()
	after.Name = "Finale"
	after.Teams[0].Name = "Team Z"
	after.Teams[0].Players[0].Name = "Annika"

	first := diff(before, flatten(after))

	for range 20 {
		next := diff(before, flatten(after))

		if len(next) != len(first) {
			t.Fatalf("length differs between runs: %d vs %d", len(next), len(first))
		}

		for i := range first {
			if next[i].Entity != first[i].Entity || next[i].EntityID != first[i].EntityID {
				t.Fatalf("order differs between runs at %d: %v/%v vs %v/%v",
					i, next[i].Entity, next[i].EntityID, first[i].Entity, first[i].EntityID)
			}
		}
	}
}
