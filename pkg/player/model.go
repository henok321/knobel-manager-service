package player

type PlayersRequest struct {
	Name string `json:"name" validate:"required,min=1"`
}

type Player struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	TeamID uint   `json:"teamID"`
}

type PlayersResponse struct {
	Player Player `json:"player"`
}
