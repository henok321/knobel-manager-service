package game

import (
	"errors"
	"fmt"
	"time"

	"github.com/henok321/knobel-manager-service/pkg/customError"

	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/setup"
	"gorm.io/gorm"
)

type GamesService struct {
	repo *GamesRepository
}

func NewGamesService(repo *GamesRepository) *GamesService {
	return &GamesService{repo}
}

func (s *GamesService) FindAllByOwner(sub string) ([]entity.Game, error) {
	return s.repo.FindAllByOwner(sub)
}

func (s *GamesService) FindByID(id int, sub string) (entity.Game, error) {
	gameById, err := s.repo.FindByID(id)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Game{}, entity.ErrorGameNotFound
		}
		return entity.Game{}, err
	}

	if !entity.IsOwner(gameById, sub) {
		return entity.Game{}, customError.NotOwner
	}

	return gameById, nil
}

func (s *GamesService) GetActiveGame(sub string) (entity.Game, error) {
	activeGame, err := s.repo.FindActiveGame(sub)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Game{}, entity.ErrorGameNotFound
		}
		return entity.Game{}, err
	}

	return activeGame, nil
}

func (s *GamesService) SetActiveGame(id int, sub string) error {

	err := s.repo.UpdateActiveGame(entity.ActiveGame{
		GameID:   id,
		OwnerSub: sub,
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *GamesService) CreateGame(sub string, game *CreateOrUpdateRequest) (entity.Game, error) {
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

func (s *GamesService) UpdateGame(id int, sub string, game CreateOrUpdateRequest) (entity.Game, error) {
	gameByID, err := s.repo.FindByID(id)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Game{}, entity.ErrorGameNotFound
		}
		return entity.Game{}, err
	}

	if !entity.IsOwner(gameByID, sub) {
		return entity.Game{}, customError.NotOwner
	}

	gameByID.Name = game.Name
	gameByID.TeamSize = game.TeamSize
	gameByID.TableSize = game.TableSize
	gameByID.NumberOfRounds = game.NumberOfRounds

	return s.repo.CreateOrUpdateGame(&gameByID)
}

func (s *GamesService) DeleteGame(id int, sub string) error {
	gameByID, err := s.repo.FindByID(id)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrorGameNotFound
		}
		return err
	}
	if !entity.IsOwner(gameByID, sub) {
		return customError.NotOwner
	}
	return s.repo.DeleteGame(id)
}

func (s *GamesService) AssignTables(game entity.Game) error {
	teams := map[int][]int{}

	for _, team := range game.Teams {
		for _, player := range team.Players {
			teams[team.ID] = append(teams[team.ID], player.ID)
		}
	}

	for i := 0; i < game.NumberOfRounds; i++ {
		tables, err := setup.AssignTables(setup.TeamSetup{Teams: teams, TeamSize: game.TeamSize, TableSize: game.TableSize}, time.Now().Unix())

		if err != nil {
			return customError.TableAssignment
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

func (s *GamesService) UpdateScore(gameID int, roundNumber int, tableNumber int, sub string, scoresRequest ScoresRequest) (entity.Game, error) {
	gameById, err := s.FindByID(gameID, sub)

	if err != nil {
		return entity.Game{}, err
	}

	if len(scoresRequest.Scores) != gameById.TableSize {
		return entity.Game{}, customError.InvalidScore
	}

	for _, round := range gameById.Rounds {
		if round.RoundNumber == roundNumber {
			for _, table := range round.Tables {
				if table.TableNumber == tableNumber {
					scores := make([]*entity.Score, 0, gameById.TableSize)
					for _, s := range scoresRequest.Scores {
						scores = append(scores, &entity.Score{
							PlayerID: s.PlayerID,
							TableID:  table.ID,
							Score:    s.Score,
						})
					}

					table.Scores = scores
					return s.repo.CreateOrUpdateGame(&gameById)
				}
			}
		}
	}
	return entity.Game{}, customError.RoundOrTableNotFound
}
