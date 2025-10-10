package game

import (
	"errors"
	"fmt"
	"time"

	"github.com/henok321/knobel-manager-service/gen/types"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/setup"
	"gorm.io/gorm"
)

type GamesService interface {
	FindAllByOwner(sub string) ([]entity.Game, error)
	FindByID(id int, sub string) (entity.Game, error)
	GetActiveGame(sub string) (entity.Game, error)
	SetActiveGame(id int, sub string) error
	CreateGame(sub string, game *types.GameCreateRequest) (entity.Game, error)
	UpdateGame(id int, sub string, game types.GameUpdateRequest) (entity.Game, error)
	DeleteGame(id int, sub string) error
	AssignTables(game entity.Game) error
}

type gamesService struct {
	repo GamesRepository
}

func NewGamesService(repo GamesRepository) GamesService {
	return &gamesService{repo}
}

func (s *gamesService) FindAllByOwner(sub string) ([]entity.Game, error) {
	return s.repo.FindAllByOwner(sub)
}

func (s *gamesService) FindByID(id int, sub string) (entity.Game, error) {
	gameByID, err := s.repo.FindByID(id)
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

func (s *gamesService) GetActiveGame(sub string) (entity.Game, error) {
	activeGame, err := s.repo.FindActiveGame(sub)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Game{}, entity.ErrGameNotFound
		}

		return entity.Game{}, err
	}

	return activeGame, nil
}

func (s *gamesService) SetActiveGame(id int, sub string) error {
	err := s.repo.UpdateActiveGame(entity.ActiveGame{
		GameID:   id,
		OwnerSub: sub,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *gamesService) CreateGame(sub string, game *types.GameCreateRequest) (entity.Game, error) {
	gameModel := entity.Game{
		Name:           game.Name,
		TeamSize:       game.TeamSize,
		TableSize:      game.TableSize,
		NumberOfRounds: game.NumberOfRounds,
		Owners:         []*entity.GameOwner{{OwnerSub: sub}},
		Status:         entity.StatusSetup,
	}

	return s.repo.CreateOrUpdateGame(&gameModel)
}

func (s *gamesService) UpdateGame(id int, sub string, game types.GameUpdateRequest) (entity.Game, error) {
	gameByID, err := s.repo.FindByID(id)
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

	// Update status if provided
	if game.Status != nil {
		gameByID.Status = entity.GameStatus(*game.Status)
	}

	return s.repo.CreateOrUpdateGame(&gameByID)
}

func (s *gamesService) DeleteGame(id int, sub string) error {
	gameByID, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrGameNotFound
		}

		return err
	}

	if !entity.IsOwner(gameByID, sub) {
		return apperror.ErrNotOwner
	}

	return s.repo.DeleteGame(id)
}

func (s *gamesService) AssignTables(game entity.Game) error {
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

		round, err = s.repo.CreateRound(&round)
		if err != nil {
			return fmt.Errorf("cannot create round %w", err)
		}

		gameTables := make([]entity.GameTable, 0, len(tables))

		for tableNumber, players := range tables {
			gameTable := entity.GameTable{TableNumber: tableNumber, RoundID: round.ID}
			for _, playerID := range players {
				gameTable.Players = append(gameTable.Players, &entity.Player{ID: playerID.ID})
			}

			gameTables = append(gameTables, gameTable)
		}

		err = s.repo.CreateGameTables(gameTables)
		if err != nil {
			return err
		}
	}

	return nil
}
