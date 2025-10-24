package team

import (
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"gorm.io/gorm"
)

type TeamsRepository interface {
	FindByID(id int) (entity.Team, error)
	CreateTeam(team *entity.Team) (entity.Team, error)
	UpdateTeam(team *entity.Team) (entity.Team, error)
	DeleteTeam(id int) error
}

type teamsRepository struct {
	db *gorm.DB
}

func NewTeamsRepository(db *gorm.DB) TeamsRepository {
	return &teamsRepository{db}
}

func (r *teamsRepository) FindByID(id int) (entity.Team, error) {
	team := entity.Team{}

	err := r.db.Where("id = ?", id).Preload("Game").Preload("Game.Owners").First(&team).Error
	if err != nil {
		return entity.Team{}, err
	}

	return team, nil
}

func (r *teamsRepository) CreateTeam(team *entity.Team) (entity.Team, error) {
	err := r.db.Create(team).Error
	if err != nil {
		return entity.Team{}, err
	}

	return *team, nil
}

func (r *teamsRepository) UpdateTeam(team *entity.Team) (entity.Team, error) {
	err := r.db.Model(team).Updates(team).Error
	if err != nil {
		return entity.Team{}, err
	}

	return *team, nil
}

func (r *teamsRepository) DeleteTeam(id int) error {
	err := r.db.Delete(&entity.Team{}, id).Error
	if err != nil {
		return err
	}

	return nil
}
