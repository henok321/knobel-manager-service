package game

type CreateOrUpdateRequest struct {
	Name           string `json:"name" validate:"required"`
	TeamSize       int    `json:"teamSize" validate:"required,min=4"`
	TableSize      int    `json:"tableSize" validate:"required,min=4"`
	NumberOfRounds int    `json:"numberOfRounds" validate:"required,min=1"`
}

type Score struct {
	PlayerID int `json:"playerID" validate:"required,numeric"`
	Score    int `json:"score" validate:"required,numeric"`
}
type ScoresRequest struct {
	Scores []Score `json:"scores" validate:"required"`
}
