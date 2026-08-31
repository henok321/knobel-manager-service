package team

import (
	"context"

	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/pkg/entity"
)

type TeamsRepository struct {
	db *gorm.DB
}

func NewTeamsRepository(db *gorm.DB) *TeamsRepository {
	return &TeamsRepository{db}
}

func (r *TeamsRepository) FindByID(ctx context.Context, id int) (entity.Team, error) {
	team := entity.Team{}

	err := r.db.WithContext(ctx).Where("id = ?", id).Preload("Game").Preload("Game.Owners").First(&team).Error
	if err != nil {
		return entity.Team{}, err
	}

	return team, nil
}

func (r *TeamsRepository) CreateTeam(ctx context.Context, team *entity.Team) (entity.Team, error) {
	err := r.db.WithContext(ctx).Save(team).Error
	if err != nil {
		return entity.Team{}, err
	}

	return *team, nil
}

func (r *TeamsRepository) UpdateTeamName(ctx context.Context, team *entity.Team, name string) (entity.Team, error) {
	if err := r.db.WithContext(ctx).Model(&entity.Team{ID: team.ID}).Update("team_name", name).Error; err != nil {
		return entity.Team{}, err
	}

	team.Name = name

	return *team, nil
}

func (r *TeamsRepository) DeleteTeam(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&entity.Team{}, id).Error
}
