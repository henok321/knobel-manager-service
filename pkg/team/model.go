package team

type PlayersRequest struct {
	Name string `json:"name" validate:"required,min=1"`
}
type TeamsRequest struct {
	Name    string            `json:"name" validate:"required"`
	Players []*PlayersRequest `json:"players"`
}
