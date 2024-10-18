package player

type PlayersService interface {
	FindByGame(teamID uint, ownerID string) ([]Player, error)
}

type playersService struct {
	playerRepository PlayersRepository
}

func NewPlayersService(playerRepository PlayersRepository) PlayersService {
	return &playersService{playerRepository}
}

func (s *playersService) FindByGame(teamID uint, ownerID string) ([]Player, error) {
	return s.playerRepository.FindByGame(teamID, ownerID)
}
