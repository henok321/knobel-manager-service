package team

import "gorm.io/gorm"

func InitalizeTeamsModule(db *gorm.DB) TeamsService {
	repository := NewTeamsRepository(db)
	service := NewTeamsService(repository)

	return service
}
