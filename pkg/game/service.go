package game

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/henok321/knobel-manager-service/gen/games"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/setup"
	"gorm.io/gorm"
)

type GamesService interface {
	FindAllByOwner(ctx context.Context, sub string) ([]entity.Game, error)
	FindByID(ctx context.Context, id int, sub string) (entity.Game, error)
	CreateGame(ctx context.Context, sub string, game *games.GameCreateRequest) (entity.Game, error)
	UpdateGame(ctx context.Context, id int, sub string, game games.GameUpdateRequest) (entity.Game, error)
	DeleteGame(ctx context.Context, id int, sub string) error
	AssignTables(ctx context.Context, game entity.Game) error
}

type gamesService struct {
	repo GamesRepository
}

func NewGamesService(repo GamesRepository) GamesService {
	return &gamesService{repo}
}

func (s *gamesService) FindAllByOwner(ctx context.Context, sub string) ([]entity.Game, error) {
	return s.repo.FindAllByOwner(ctx, sub)
}

func (s *gamesService) FindByID(ctx context.Context, id int, sub string) (entity.Game, error) {
	gameByID, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Game{}, entity.ErrGameNotFound
		}

		return entity.Game{}, err
	}

	if !entity.IsOwner(gameByID, sub) {
		return entity.Game{}, apperror.ErrNotOwner
	}

	return gameByID, nil
}

func (s *gamesService) CreateGame(ctx context.Context, sub string, game *games.GameCreateRequest) (entity.Game, error) {
	gameModel := entity.Game{
		Name:           game.Name,
		TeamSize:       game.TeamSize,
		TableSize:      game.TableSize,
		NumberOfRounds: game.NumberOfRounds,
		Owners:         []*entity.GameOwner{{OwnerSub: sub}},
		Status:         entity.StatusSetup,
	}

	return s.repo.CreateOrUpdateGame(ctx, &gameModel)
}

func (s *gamesService) UpdateGame(ctx context.Context, id int, sub string, game games.GameUpdateRequest) (entity.Game, error) {
	gameByID, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Game{}, entity.ErrGameNotFound
		}

		return entity.Game{}, err
	}

	if !entity.IsOwner(gameByID, sub) {
		return entity.Game{}, apperror.ErrNotOwner
	}

	gameByID.Name = game.Name
	gameByID.TeamSize = game.TeamSize
	gameByID.TableSize = game.TableSize
	gameByID.NumberOfRounds = game.NumberOfRounds

	if game.Status != "" {
		gameByID.Status = entity.GameStatus(game.Status)
	}

	if game.Status == "in_progress" {
		teams := map[int][]int{}

		for _, team := range gameByID.Teams {
			for _, player := range team.Players {
				teams[team.ID] = append(teams[team.ID], player.ID)
			}
		}

		validSetup := setup.IsAssignable(teams, gameByID.TeamSize, gameByID.TableSize)

		if !validSetup {
			return entity.Game{}, apperror.ErrInvalidGameSetup
		}
	}

	if game.Status == "completed" {
		if gameIncompleteScoresMissing(gameByID) {
			return entity.Game{}, apperror.ErrGameIncomplete
		}
	}
	return s.repo.CreateOrUpdateGame(ctx, &gameByID)
}

func gameIncompleteScoresMissing(game entity.Game) bool {
	for _, team := range game.Teams {
		for _, player := range team.Players {
			scores := len(player.Scores)
			if scores < game.NumberOfRounds {
				return true
			}
		}
	}
	return false
}

func (s *gamesService) DeleteGame(ctx context.Context, id int, sub string) error {
	gameByID, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrGameNotFound
		}

		return err
	}

	if !entity.IsOwner(gameByID, sub) {
		return apperror.ErrNotOwner
	}

	return s.repo.DeleteGame(ctx, id)
}

func (s *gamesService) AssignTables(ctx context.Context, game entity.Game) error {
	return s.repo.WithinTransaction(ctx, func(ctx context.Context, txRepo GamesRepository) error {
		if err := txRepo.ResetGameTables(ctx, game.ID); err != nil {
			return fmt.Errorf("cannot reset game tables: %w", err)
		}

		teams := map[int][]int{}

		for _, team := range game.Teams {
			for _, player := range team.Players {
				teams[team.ID] = append(teams[team.ID], player.ID)
			}
		}

		for i := range game.NumberOfRounds {
			tables, err := setup.AssignTables(setup.TeamSetup{Teams: teams, TeamSize: game.TeamSize, TableSize: game.TableSize}, time.Now().Unix()-(int64(i)*1000))
			if err != nil {
				return apperror.ErrTableAssignment
			}

			round := entity.Round{
				RoundNumber: i + 1,
				GameID:      game.ID,
			}

			round, err = txRepo.CreateRound(ctx, &round)
			if err != nil {
				return fmt.Errorf("cannot create round: %w", err)
			}

			gameTables := make([]entity.GameTable, 0, len(tables))

			for tableNumber, players := range tables {
				gameTable := entity.GameTable{TableNumber: tableNumber + 1, RoundID: round.ID}
				for _, playerID := range players {
					gameTable.Players = append(gameTable.Players, &entity.Player{ID: playerID.ID})
				}

				gameTables = append(gameTables, gameTable)
			}

			err = txRepo.CreateGameTables(ctx, gameTables)
			if err != nil {
				return fmt.Errorf("cannot create game tables: %w", err)
			}
		}

		return nil
	})
}
