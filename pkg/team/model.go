package team

type TeamsRequest struct {
	Name string `json:"name" validate:"required"`
}
