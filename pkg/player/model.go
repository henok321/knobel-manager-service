package player

type PlayersRequest struct {
	Name string `json:"name" binding:"required,min=1"`
}

type PlayersResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}
