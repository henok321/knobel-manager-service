package entity_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/henok321/knobel-manager-service/pkg/entity"
)

func TestRoundStatus(t *testing.T) {
	seat := func(scored bool) *entity.GameTable {
		table := &entity.GameTable{Players: []*entity.Player{{ID: 1}, {ID: 2}}}
		if scored {
			table.Scores = []*entity.Score{{PlayerID: 1}, {PlayerID: 2}}
		}

		return table
	}

	tests := map[string]struct {
		gameStatus entity.GameStatus
		tables     []*entity.GameTable
		expected   entity.GameStatus
	}{
		"game not started": {
			gameStatus: entity.StatusSetup,
			tables:     []*entity.GameTable{seat(false)},
			expected:   entity.StatusSetup,
		},
		"running, nothing scored": {
			gameStatus: entity.StatusInProgress,
			tables:     []*entity.GameTable{seat(false), seat(false)},
			expected:   entity.StatusInProgress,
		},
		"running, one table still open": {
			gameStatus: entity.StatusInProgress,
			tables:     []*entity.GameTable{seat(true), seat(false)},
			expected:   entity.StatusInProgress,
		},
		"running, every seat scored": {
			gameStatus: entity.StatusInProgress,
			tables:     []*entity.GameTable{seat(true), seat(true)},
			expected:   entity.StatusCompleted,
		},
		"game completed": {
			gameStatus: entity.StatusCompleted,
			tables:     []*entity.GameTable{seat(true)},
			expected:   entity.StatusCompleted,
		},
		"tables not loaded": {
			gameStatus: entity.StatusInProgress,
			tables:     nil,
			expected:   entity.StatusInProgress,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			status := entity.RoundStatus(entity.Game{Status: tc.gameStatus}, entity.Round{Tables: tc.tables})
			assert.Equal(t, tc.expected, status)
		})
	}
}
