package audit

import (
	"maps"
	"slices"
	"strconv"

	"github.com/henok321/knobel-manager-service/pkg/entity"
)

// The persisted shape of a change. Deliberately separate from api.AuditChange:
// coupling stored history to a regenerated type would let a spec edit rewrite
// the meaning of rows already on disk.
type fieldChange struct {
	Field string  `json:"field"`
	From  *string `json:"from"`
	To    *string `json:"to"`
}

type entityChange struct {
	Entity   entity.AuditEntity
	EntityID string
	Action   entity.AuditAction
	Changes  []fieldChange
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

func diff(before, after map[recordKey]fields) []entityChange {
	changes := make([]entityChange, 0)

	for key, afterFields := range after {
		beforeFields, existed := before[key]

		if !existed {
			changes = append(changes, entityChange{key.Entity, key.ID, entity.AuditActionCreate, fieldDiff(nil, afterFields)})
			continue
		}

		if changed := fieldDiff(beforeFields, afterFields); len(changed) > 0 {
			changes = append(changes, entityChange{key.Entity, key.ID, entity.AuditActionUpdate, changed})
		}
	}

	for key, beforeFields := range before {
		if _, survived := after[key]; !survived {
			changes = append(changes, entityChange{key.Entity, key.ID, entity.AuditActionDelete, fieldDiff(beforeFields, nil)})
		}
	}

	return changes
}

func fieldDiff(before, after fields) []fieldChange {
	union := make(fields, len(before)+len(after))
	maps.Copy(union, before)
	maps.Copy(union, after)

	changes := make([]fieldChange, 0, len(union))

	for _, name := range slices.Sorted(maps.Keys(union)) {
		fromValue, hadBefore := before[name]
		toValue, hasAfter := after[name]

		if hadBefore && hasAfter && fromValue == toValue {
			continue
		}

		change := fieldChange{Field: name}

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
