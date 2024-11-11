package team

import (
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"gorm.io/gorm"
)

type TeamsRepository interface {
	FindById(id uint) (entity.Team, error)
	CreateOrUpdateTeam(team *entity.Team) (entity.Team, error)
	DeleteTeam(id uint) error
}

type teamsRepository struct {
	db *gorm.DB
}

func NewTeamsRepository(db *gorm.DB) TeamsRepository {
	return &teamsRepository{db}
}

func (r *teamsRepository) FindById(id uint) (entity.Team, error) {
	team := entity.Team{}
	err := r.db.Where("id = ?", id).Preload("Game").Preload("Game.Owners").First(&team).Error
	if err != nil {
		return entity.Team{}, err
	}
	return team, nil
}

func (r *teamsRepository) CreateOrUpdateTeam(team *entity.Team) (entity.Team, error) {
	err := r.db.Save(team).Error
	if err != nil {
		return entity.Team{}, err
	}
	return *team, nil
}

func (r *teamsRepository) DeleteTeam(id uint) error {
	err := r.db.Delete(&entity.Team{}, id).Error
	if err != nil {
		return err
	}
	return nil
}
