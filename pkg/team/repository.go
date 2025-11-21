package team

import (
	"context"

	"github.com/henok321/knobel-manager-service/pkg/entity"
	"gorm.io/gorm"
)

type TeamsRepository interface {
	FindByID(ctx context.Context, id int) (entity.Team, error)
	CreateOrUpdateTeam(ctx context.Context, team *entity.Team) (entity.Team, error)
	DeleteTeam(ctx context.Context, id int) error
}

type teamsRepository struct {
	db *gorm.DB
}

func NewTeamsRepository(db *gorm.DB) TeamsRepository {
	return &teamsRepository{db}
}

func (r *teamsRepository) FindByID(ctx context.Context, id int) (entity.Team, error) {
	team := entity.Team{}

	err := r.db.WithContext(ctx).Where("id = ?", id).Preload("Game").Preload("Game.Owners").First(&team).Error
	if err != nil {
		return entity.Team{}, err
	}

	return team, nil
}

func (r *teamsRepository) CreateOrUpdateTeam(ctx context.Context, team *entity.Team) (entity.Team, error) {
	err := r.db.WithContext(ctx).Save(team).Error
	if err != nil {
		return entity.Team{}, err
	}

	return *team, nil
}

func (r *teamsRepository) DeleteTeam(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&entity.Team{}, id).Error
}
