package player

type PlayersRequest struct {
	Name string `json:"name" binding:"required not_blank"`
}
