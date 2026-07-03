package apperror

import "errors"

var (
	ErrTeamNotFound         = errors.New("team not found")
	ErrPlayerNotFound       = errors.New("player not found")
	ErrNotOwner             = errors.New("user is not the owner of the requested resource")
	ErrTableAssignment      = errors.New("cannot assign players to tables")
	ErrInvalidScore         = errors.New("invalid score")
	ErrRoundOrTableNotFound = errors.New("round or table not found")
	ErrTeamSizeNotAllowed   = errors.New("team size not allowed")
	ErrInvalidGameSetup     = errors.New("invalid game setup")
	ErrGameIncomplete       = errors.New("game is complete")
	ErrUserNotFound         = errors.New("no user found for the given email")
	ErrAlreadyOwner         = errors.New("user is already an owner")
	ErrLastOwner            = errors.New("cannot remove the last owner")
)
