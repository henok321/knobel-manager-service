package game

type GameRequest struct {
	Name           string `json:"name" binding:"required"`
	TeamSize       uint   `json:"teamSize" binding:"required,min=4"`
	TableSize      uint   `json:"tableSize" binding:"required,min=4"`
	NumberOfRounds uint   `json:"numberOfRounds" binding:"required,min=1"`
}
