package game

type GamesService interface {
	FindAllByOwner(sub string) ([]Game, error)
	FindByID(id uint) (*Game, error)
	Create(game *Game) error
	Update(game *Game) error
	Delete(id uint) error
}

type gamesService struct {
	repo GamesRepository
}

func NewGamesService(repo GamesRepository) GamesService {
	return &gamesService{repo}
}

func (s *gamesService) FindAllByOwner(sub string) ([]Game, error) {
	return s.repo.FindByOwner(sub)
}

func (s *gamesService) FindByID(id uint) (*Game, error) {
	return s.repo.FindById(id)

}

func (s *gamesService) Create(game *Game) error {
	return s.repo.Create(game)
}

func (s *gamesService) Update(game *Game) error {
	return s.repo.Update(game)
}

func (s *gamesService) Delete(id uint) error {
	return s.repo.Delete(id)
}
