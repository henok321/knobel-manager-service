package game

import (
	"errors"
	"fmt"
	"time"

	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/setup"
	"gorm.io/gorm"
)

type GamesService interface {
	FindAllByOwner(sub string) ([]entity.Game, error)
	FindByID(id int, sub string) (entity.Game, error)
	GetActiveGame(sub string) (entity.Game, error)
	SetActiveGame(id int, sub string) error
	CreateGame(sub string, game *GameRequest) (entity.Game, error)
	UpdateGame(id int, sub string, game GameRequest) (entity.Game, error)
	DeleteGame(id int, sub string) error
	AssignTables(game entity.Game) error
	UpdateScore(gameID int, roundNumber int, tableNumber int, sub string, scoresRequest ScoresRequest) (entity.Game, error)
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
	gameById, err := s.repo.FindByID(id)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Game{}, entity.ErrorGameNotFound
		}
		return entity.Game{}, err
	}

	if !entity.IsOwner(gameById, sub) {
		return entity.Game{}, entity.ErrorNotOwner
	}

	return gameById, nil
}

func (s *gamesService) GetActiveGame(sub string) (entity.Game, error) {
	activeGame, err := s.repo.FindActiveGame(sub)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Game{}, entity.ErrorGameNotFound
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

func (s *gamesService) CreateGame(sub string, game *GameRequest) (entity.Game, error) {
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

func (s *gamesService) UpdateGame(id int, sub string, game GameRequest) (entity.Game, error) {
	gameByID, err := s.repo.FindByID(id)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Game{}, entity.ErrorGameNotFound
		}
		return entity.Game{}, err
	}

	if !entity.IsOwner(gameByID, sub) {
		return entity.Game{}, entity.ErrorNotOwner
	}

	gameByID.Name = game.Name
	gameByID.TeamSize = game.TeamSize
	gameByID.TableSize = game.TableSize
	gameByID.NumberOfRounds = game.NumberOfRounds

	return s.repo.CreateOrUpdateGame(&gameByID)
}

func (s *gamesService) DeleteGame(id int, sub string) error {
	gameByID, err := s.repo.FindByID(id)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrorGameNotFound
		}
		return err
	}
	if !entity.IsOwner(gameByID, sub) {
		return entity.ErrorNotOwner
	}
	return s.repo.DeleteGame(id)
}

func (s *gamesService) AssignTables(game entity.Game) error {
	teams := map[int][]int{}

	for _, team := range game.Teams {
		for _, player := range team.Players {
			teams[int(team.ID)] = append(teams[int(team.ID)], int(player.ID))
		}
	}

	for i := 0; i < int(game.NumberOfRounds); i++ {
		tables, err := setup.AssignTables(setup.TeamSetup{Teams: teams, TeamSize: int(game.TeamSize), TableSize: int(game.TableSize)}, time.Now().Unix())

		if err != nil {
			return entity.ErrorTableAssignment
		}

		round := entity.Round{
			RoundNumber: int(i + 1),
			GameID:      game.ID,
		}

		round, err = s.repo.CreateRound(&round)

		if err != nil {
			return fmt.Errorf("cannot create round %w", err)
		}

		gameTables := make([]entity.GameTable, 0, len(tables))

		for tableNumber, players := range tables {
			gameTable := entity.GameTable{TableNumber: int(tableNumber), RoundID: round.ID}
			for _, playerID := range players {
				gameTable.Players = append(gameTable.Players, &entity.Player{ID: int(playerID.ID)})
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

func (s *gamesService) UpdateScore(gameID int, roundNumber int, tableNumber int, sub string, scoresRequest ScoresRequest) (entity.Game, error) {
	gameById, err := s.FindByID(gameID, sub)

	if err != nil {
		return entity.Game{}, err
	}

	if int(len(scoresRequest.Scores)) != gameById.TableSize {
		return entity.Game{}, entity.ErrorInvalidScore
	}

	for _, round := range gameById.Rounds {
		if round.RoundNumber == int(roundNumber) {
			for _, table := range round.Tables {
				if table.TableNumber == int(tableNumber) {
					scores := make([]*entity.Score, 0, gameById.TableSize)
					for _, s := range scoresRequest.Scores {
						scores = append(scores, &entity.Score{
							PlayerID: s.PlayerID,
							TableID:  table.ID,
							Score:    int(s.Score),
						})
					}

					table.Scores = scores
					return s.repo.CreateOrUpdateGame(&gameById)
				}
			}
		}
	}
	return entity.Game{}, entity.ErrorRoundOrTableNotFound
}
