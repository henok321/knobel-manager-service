package game

type GameRequest struct {
	Name           string `json:"name" validate:"required"`
	TeamSize       uint   `json:"teamSize" validate:"required,min=4"`
	TableSize      uint   `json:"tableSize" validate:"required,min=4"`
	NumberOfRounds uint   `json:"numberOfRounds" validate:"required,min=1"`
}

type Score struct {
	PlayerID uint `json:"playerID" validate:"required,numeric"`
	Score    uint `json:"score" validate:"required,numeric"`
}
type ScoresRequest struct {
	Scores []Score `json:"scores" validate:"required"`
}
