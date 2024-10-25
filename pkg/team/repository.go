package team

import (
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"gorm.io/gorm"
)

type TeamsRepository interface {
	CreateTeam(team *entity.Team) (entity.Team, error)
	UpdateTeam(team *entity.Team) (entity.Team, error)
	DeleteTeam(id uint) error
}

type teamsRepository struct {
	db *gorm.DB
}

func NewTeamsRepository(db *gorm.DB) TeamsRepository {
	return &teamsRepository{db}
}

func (r *teamsRepository) CreateTeam(team *entity.Team) (entity.Team, error) {
	return entity.Team{}, nil
}

func (r *teamsRepository) UpdateTeam(team *entity.Team) (entity.Team, error) {
	return entity.Team{}, nil
}

func (r *teamsRepository) DeleteTeam(id uint) error {
	return nil
}
