package table

import (
	"context"

	"github.com/henok321/knobel-manager-service/gen/types"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
)

type TablesService interface {
	UpdateScore(ctx context.Context, gameID, roundNumber, tableNumber int, sub string, scoresRequest types.ScoresRequest) (entity.GameTable, error)
}

type tablesService struct {
	repo TablesRepository
}

func NewTablesService(repo TablesRepository) TablesService {
	return &tablesService{repo}
}

func (t *tablesService) UpdateScore(ctx context.Context, gameID, roundNumber, tableNumber int, sub string, scoresRequest types.ScoresRequest) (entity.GameTable, error) {
	table, err := t.repo.FindTable(ctx, sub, gameID, roundNumber, tableNumber)
	if err != nil {
		return entity.GameTable{}, apperror.ErrRoundOrTableNotFound
	}

	if len(scoresRequest.Scores) != len(table.Players) {
		return entity.GameTable{}, apperror.ErrInvalidScore
	}

	existingScores := make(map[int]*entity.Score)
	for _, score := range table.Scores {
		existingScores[score.PlayerID] = score
	}

	scores := make([]*entity.Score, 0, len(table.Players))
	for _, s := range scoresRequest.Scores {
		if existingScore, exists := existingScores[s.PlayerID]; exists {
			existingScore.Score = s.Score
			scores = append(scores, existingScore)
		} else {
			scores = append(scores, &entity.Score{
				PlayerID: s.PlayerID,
				TableID:  table.ID,
				Score:    s.Score,
			})
		}
	}

	table.Scores = scores

	table, err = t.repo.UpdateTable(ctx, &table)
	if err != nil {
		return entity.GameTable{}, err
	}

	return table, nil
}
