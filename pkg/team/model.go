package team

type TeamsRequest struct {
	Name string `json:"name" binding:"required"`
}
