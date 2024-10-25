package team

import "github.com/henok321/knobel-manager-service/pkg/entity"

type TeamsService interface {
	CreateTeam(gameID uint, sub string, request TeamsRequest) (entity.Team, error)
	UpdateTeam(gameID uint, sub string, request TeamsRequest) (entity.Team, error)
}

type teamsService struct {
	repo TeamsRepository
}

func NewTeamsService(repo TeamsRepository) TeamsService {
	return &teamsService{
		repo,
	}
}

func (s *teamsService) CreateTeam(gameID uint, sub string, request TeamsRequest) (entity.Team, error) {
	return entity.Team{}, nil
}

func (s *teamsService) UpdateTeam(gameID uint, sub string, request TeamsRequest) (entity.Team, error) {
	return entity.Team{}, nil
}
