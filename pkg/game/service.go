package game

import (
	"context"
	"errors"
	"fmt"
	"time"

	"firebase.google.com/go/v4/auth"
	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/gen/api"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/setup"
)

// UserLookup resolves an email to a Firebase user. It is the subset of the
// Firebase auth client the service needs; defined here so pkg/game does not
// depend on api/middleware.
type UserLookup interface {
	GetUserByEmail(ctx context.Context, email string) (*auth.UserRecord, error)
}

type GamesService struct {
	repo  *GamesRepository
	users UserLookup
}

func NewGamesService(repo *GamesRepository, users UserLookup) *GamesService {
	return &GamesService{repo, users}
}

func (s *GamesService) FindAllByOwner(ctx context.Context, sub string) ([]entity.Game, error) {
	return s.repo.FindAllByOwner(ctx, sub)
}

func (s *GamesService) FindByID(ctx context.Context, id int, sub string) (entity.Game, error) {
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

func (s *GamesService) CreateGame(ctx context.Context, sub string, game *api.GameCreateRequest) (entity.Game, error) {
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

func (s *GamesService) UpdateGame(ctx context.Context, id int, sub string, game api.GameUpdateRequest) (entity.Game, error) {
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

		if len(gameByID.Rounds) != gameByID.NumberOfRounds {
			return entity.Game{}, apperror.ErrInvalidGameSetup
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

func (s *GamesService) DeleteGame(ctx context.Context, id int, sub string) error {
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

func (s *GamesService) AddOwner(ctx context.Context, gameID int, callerSub, email string) (entity.Game, error) {
	game, err := s.FindByID(ctx, gameID, callerSub) // enforces game exists + caller is an owner
	if err != nil {
		return entity.Game{}, err
	}

	record, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		return entity.Game{}, apperror.ErrUserNotFound
	}

	if entity.IsOwner(game, record.UID) {
		return entity.Game{}, apperror.ErrAlreadyOwner
	}

	if err := s.repo.AddOwner(ctx, gameID, record.UID); err != nil {
		return entity.Game{}, err
	}

	return s.repo.FindByID(ctx, gameID)
}

func (s *GamesService) RemoveOwner(ctx context.Context, gameID int, callerSub, targetSub string) (entity.Game, error) {
	game, err := s.FindByID(ctx, gameID, callerSub) // enforces game exists + caller is an owner
	if err != nil {
		return entity.Game{}, err
	}

	if !entity.IsOwner(game, targetSub) {
		return entity.Game{}, entity.ErrGameNotFound
	}

	if len(game.Owners) <= 1 {
		return entity.Game{}, apperror.ErrLastOwner
	}

	if err := s.repo.RemoveOwner(ctx, gameID, targetSub); err != nil {
		return entity.Game{}, err
	}

	return s.repo.FindByID(ctx, gameID)
}

func (s *GamesService) AssignTables(ctx context.Context, game entity.Game) error {
	return s.repo.WithinTransaction(ctx, func(ctx context.Context, txRepo *GamesRepository) error {
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
