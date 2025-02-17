package team

import (
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"gorm.io/gorm"
)

type TeamsRepository struct {
	db *gorm.DB
}

func NewTeamsRepository(db *gorm.DB) *TeamsRepository {
	return &TeamsRepository{db}
}

func (r *TeamsRepository) FindById(id int) (entity.Team, error) {
	team := entity.Team{}
	err := r.db.Where("id = ?", id).Preload("Game").Preload("Game.Owners").First(&team).Error
	if err != nil {
		return entity.Team{}, err
	}
	return team, nil
}

func (r *TeamsRepository) CreateOrUpdateTeam(team *entity.Team) (entity.Team, error) {
	err := r.db.Save(team).Error
	if err != nil {
		return entity.Team{}, err
	}
	return *team, nil
}

func (r *TeamsRepository) DeleteTeam(id int) error {
	err := r.db.Delete(&entity.Team{}, id).Error
	if err != nil {
		return err
	}
	return nil
}
