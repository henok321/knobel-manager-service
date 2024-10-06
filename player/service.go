package player

type PlayersService interface {
	FindAll() ([]Player, error)
}

type playersService struct {
	playerRepository PlayerRepository
}

func NewPlayersService(playerRepository PlayerRepository) PlayersService {
	return &playersService{playerRepository}
}

func (s *playersService) FindAll() ([]Player, error) {
	players, err := s.playerRepository.FindAll()
	if err != nil {
		return nil, err
	}
	return players, nil
}
