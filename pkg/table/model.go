package table

type Score struct {
	PlayerID int `json:"playerID" validate:"required,numeric"`
	Score    int `json:"score" validate:"required,numeric"`
}
type ScoresRequest struct {
	Scores []Score `json:"scores" validate:"required"`
}
