package game

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/gen/api"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/setup"
)

type GamesService struct {
	repo  *GamesRepository
	users middleware.FirebaseAuth
}

func NewGamesService(repo *GamesRepository, users middleware.FirebaseAuth) *GamesService {
	return &GamesService{repo, users}
}

func (s *GamesService) FindAllByOwner(ctx context.Context, sub string) ([]entity.Game, error) {
	return s.repo.FindAllByOwner(ctx, sub)
}

func (s *GamesService) FindByID(ctx context.Context, id int, sub string) (entity.Game, error) {
	gameByID, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Game{}, apperror.ErrGameNotFound
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

func (s *GamesService) UpdateGame(ctx context.Context, id int, sub string, request api.GameUpdateRequest) (entity.Game, error) {
	var updated entity.Game

	err := s.repo.WithinTransaction(ctx, func(ctx context.Context, txRepo *GamesRepository) error {
		locked, err := txRepo.LockGame(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.ErrGameNotFound
			}

			return err
		}

		if !entity.IsOwner(locked, sub) {
			return apperror.ErrNotOwner
		}

		gameByID, err := txRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}

		counts, err := txRepo.CountRelated(ctx, id)
		if err != nil {
			return err
		}

		configChanged := request.TeamSize != gameByID.TeamSize ||
			request.TableSize != gameByID.TableSize ||
			request.NumberOfRounds != gameByID.NumberOfRounds

		if configChanged {
			if err := entity.EnsureSetupNotAssigned(gameByID.Status, counts.Rounds); err != nil {
				return err
			}
		}

		if request.Status != "" {
			if err := ensureTransitionAllowed(gameByID, counts, entity.GameStatus(request.Status)); err != nil {
				return err
			}

			gameByID.Status = entity.GameStatus(request.Status)
		}

		gameByID.Name = request.Name
		gameByID.TeamSize = request.TeamSize
		gameByID.TableSize = request.TableSize
		gameByID.NumberOfRounds = request.NumberOfRounds

		updated, err = txRepo.CreateOrUpdateGame(ctx, &gameByID)

		return err
	})

	return updated, err
}

func ensureTransitionAllowed(game entity.Game, counts Counts, next entity.GameStatus) error {
	if game.Status == next {
		return nil
	}

	switch {
	case game.Status == entity.StatusSetup && next == entity.StatusInProgress:
		if counts.Rounds != game.NumberOfRounds {
			return apperror.ErrInvalidGameSetup
		}

		if !setup.IsAssignable(teamsMap(game), game.TeamSize, game.TableSize) {
			return apperror.ErrInvalidGameSetup
		}

		return nil
	case game.Status == entity.StatusInProgress && next == entity.StatusCompleted:
		if counts.Scores < counts.Players*game.NumberOfRounds {
			return apperror.ErrGameIncomplete
		}

		return nil
	case game.Status == entity.StatusInProgress && next == entity.StatusSetup:
		if counts.Scores > 0 {
			return apperror.ErrInvalidStatusTransition
		}

		return nil
	default:
		return apperror.ErrInvalidStatusTransition
	}
}

func (s *GamesService) WithinSetup(ctx context.Context, gameID int, sub string, write func(ctx context.Context, tx *gorm.DB, game entity.Game) error) error {
	return s.repo.WithinTransaction(ctx, func(ctx context.Context, txRepo *GamesRepository) error {
		game, err := txRepo.LockGame(ctx, gameID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.ErrGameNotFound
			}

			return err
		}

		if !entity.IsOwner(game, sub) {
			return apperror.ErrNotOwner
		}

		counts, err := txRepo.CountRelated(ctx, gameID)
		if err != nil {
			return err
		}

		if err := entity.EnsureSetupNotAssigned(game.Status, counts.Rounds); err != nil {
			return err
		}

		return write(ctx, txRepo.db, game)
	})
}

func (s *GamesService) ResetSetup(ctx context.Context, gameID int, sub string) error {
	return s.repo.WithinTransaction(ctx, func(ctx context.Context, txRepo *GamesRepository) error {
		game, err := txRepo.LockGame(ctx, gameID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.ErrGameNotFound
			}

			return err
		}

		if !entity.IsOwner(game, sub) {
			return apperror.ErrNotOwner
		}

		if game.Status != entity.StatusSetup {
			return apperror.ErrGameNotEditable
		}

		return txRepo.ResetGameTables(ctx, gameID)
	})
}

func teamsMap(game entity.Game) map[int][]int {
	teams := map[int][]int{}

	for _, team := range game.Teams {
		for _, player := range team.Players {
			teams[team.ID] = append(teams[team.ID], player.ID)
		}
	}

	return teams
}

func (s *GamesService) DeleteGame(ctx context.Context, id int, sub string) error {
	if _, err := s.FindByID(ctx, id, sub); err != nil {
		return err
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
		return entity.Game{}, apperror.ErrGameNotFound
	}

	if len(game.Owners) <= 1 {
		return entity.Game{}, apperror.ErrLastOwner
	}

	if err := s.repo.RemoveOwner(ctx, gameID, targetSub); err != nil {
		return entity.Game{}, err
	}

	return s.repo.FindByID(ctx, gameID)
}

func (s *GamesService) AssignTables(ctx context.Context, gameID int, sub string) error {
	return s.repo.WithinTransaction(ctx, func(ctx context.Context, txRepo *GamesRepository) error {
		if _, err := txRepo.LockGame(ctx, gameID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.ErrGameNotFound
			}

			return err
		}

		game, err := txRepo.FindByID(ctx, gameID)
		if err != nil {
			return err
		}

		if !entity.IsOwner(game, sub) {
			return apperror.ErrNotOwner
		}

		if game.Status != entity.StatusSetup {
			return apperror.ErrGameNotEditable
		}

		if len(game.Teams) < game.TableSize {
			return apperror.ErrNotEnoughTeams
		}

		if err := txRepo.ResetGameTables(ctx, game.ID); err != nil {
			return fmt.Errorf("cannot reset game tables: %w", err)
		}

		teams := teamsMap(game)

		for i := range game.NumberOfRounds {
			tables, err := setup.AssignTables(setup.TeamSetup{Teams: teams, TeamSize: game.TeamSize, TableSize: game.TableSize}, time.Now().Unix()-(int64(i)*1000))
			if err != nil {
				return apperror.ErrTableAssignment
			}

			round := entity.Round{
				RoundNumber: i + 1,
				GameID:      game.ID,
				Status:      string(entity.StatusSetup),
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
