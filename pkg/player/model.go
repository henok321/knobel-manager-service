package player

type PlayersRequest struct {
	Name string `json:"name" validate:"required,min=1"`
}

type Player struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	TeamID int    `json:"teamID"`
}

type PlayersResponse struct {
	Player Player `json:"player"`
}
