package game

type GameRequest struct {
	Name           string `json:"name" validate:"required"`
	TeamSize       uint   `json:"teamSize" validate:"required,min=4"`
	TableSize      uint   `json:"tableSize" validate:"required,min=4"`
	NumberOfRounds uint   `json:"numberOfRounds" validate:"required,min=1"`
}
