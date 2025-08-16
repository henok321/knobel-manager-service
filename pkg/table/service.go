package table

import (
	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
)

type TablesService interface {
	UpdateScore(gameID, roundNumber, tableNumber int, sub string, scoresRequest ScoresRequest) (entity.GameTable, error)
}

type tablesService struct {
	repo TablesRepository
}

func NewTablesService(repo TablesRepository) TablesService {
	return tablesService{repo}
}

func (t tablesService) UpdateScore(gameID, roundNumber, tableNumber int, sub string, scoresRequest ScoresRequest) (entity.GameTable, error) {
	table, err := t.repo.FindTable(sub, gameID, roundNumber, tableNumber)
	if err != nil {
		return entity.GameTable{}, apperror.ErrRoundOrTableNotFound
	}

	if len(scoresRequest.Scores) != len(table.Players) {
		return entity.GameTable{}, apperror.ErrInvalidScore
	}

	scores := make([]*entity.Score, 0, len(table.Players))
	for _, s := range scoresRequest.Scores {
		scores = append(scores, &entity.Score{
			PlayerID: s.PlayerID,
			TableID:  table.ID,
			Score:    s.Score,
		})
	}

	table.Scores = scores

	table, err = t.repo.UpdateTable(&table)
	if err != nil {
		return entity.GameTable{}, err
	}

	return table, nil
}
