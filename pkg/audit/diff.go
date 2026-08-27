package audit

import (
	"cmp"
	"slices"
	"strconv"

	"github.com/henok321/knobel-manager-service/pkg/entity"
)

type FieldChange struct {
	Field string  `json:"field"`
	From  *string `json:"from"`
	To    *string `json:"to"`
}

type EntityChange struct {
	Entity   entity.AuditEntity
	EntityID string
	Action   entity.AuditAction
	Changes  []FieldChange
}

type recordKey struct {
	Entity entity.AuditEntity
	ID     string
}

type fields map[string]string

// Rounds, tables and table players are left out on purpose: they are output of the
// table assignment algorithm rather than human edits, and one setup call would emit
// well over a hundred events. Setup is recorded as a single event instead.
func flatten(game entity.Game) map[recordKey]fields {
	records := map[recordKey]fields{}

	if game.ID == 0 {
		return records
	}

	records[recordKey{entity.AuditEntityGame, strconv.Itoa(game.ID)}] = fields{
		"name":             game.Name,
		"team_size":        strconv.Itoa(game.TeamSize),
		"table_size":       strconv.Itoa(game.TableSize),
		"number_of_rounds": strconv.Itoa(game.NumberOfRounds),
		"status":           string(game.Status),
	}

	for _, owner := range game.Owners {
		records[recordKey{entity.AuditEntityOwner, owner.OwnerSub}] = fields{}
	}

	for _, team := range game.Teams {
		records[recordKey{entity.AuditEntityTeam, strconv.Itoa(team.ID)}] = fields{
			"name": team.Name,
		}

		for _, player := range team.Players {
			records[recordKey{entity.AuditEntityPlayer, strconv.Itoa(player.ID)}] = fields{
				"name":    player.Name,
				"team_id": strconv.Itoa(player.TeamID),
			}

			for _, score := range player.Scores {
				records[recordKey{entity.AuditEntityScore, strconv.Itoa(score.ID)}] = fields{
					"score":     strconv.Itoa(score.Score),
					"player_id": strconv.Itoa(score.PlayerID),
					"table_id":  strconv.Itoa(score.TableID),
				}
			}
		}
	}

	return records
}

func diff(before, after map[recordKey]fields) []EntityChange {
	changes := make([]EntityChange, 0)

	for key, afterFields := range after {
		beforeFields, existed := before[key]

		if !existed {
			changes = append(changes, EntityChange{key.Entity, key.ID, entity.AuditActionCreate, fieldDiff(nil, afterFields)})
			continue
		}

		if fieldChanges := fieldDiff(beforeFields, afterFields); len(fieldChanges) > 0 {
			changes = append(changes, EntityChange{key.Entity, key.ID, entity.AuditActionUpdate, fieldChanges})
		}
	}

	for key, beforeFields := range before {
		if _, survived := after[key]; !survived {
			changes = append(changes, EntityChange{key.Entity, key.ID, entity.AuditActionDelete, fieldDiff(beforeFields, nil)})
		}
	}

	// ponytail: map iteration is random, so sort to keep output stable. Lexical id
	// order puts "10" before "9"; harmless, since events from one request are
	// explicitly unordered. Sort numerically if that ever needs to read well.
	slices.SortFunc(changes, func(a, b EntityChange) int {
		return cmp.Or(
			cmp.Compare(a.Entity, b.Entity),
			cmp.Compare(a.EntityID, b.EntityID),
		)
	})

	return changes
}

func fieldDiff(before, after fields) []FieldChange {
	names := make([]string, 0, len(before)+len(after))

	for name := range before {
		names = append(names, name)
	}

	for name := range after {
		if _, seen := before[name]; !seen {
			names = append(names, name)
		}
	}

	slices.Sort(names)

	changes := make([]FieldChange, 0, len(names))

	for _, name := range names {
		fromValue, hadBefore := before[name]
		toValue, hasAfter := after[name]

		if hadBefore && hasAfter && fromValue == toValue {
			continue
		}

		change := FieldChange{Field: name}

		if hadBefore {
			change.From = &fromValue
		}

		if hasAfter {
			change.To = &toValue
		}

		changes = append(changes, change)
	}

	return changes
}
