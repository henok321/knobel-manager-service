package apperror

import "errors"

var ErrTeamNotFound = errors.New("team not found")
var ErrPlayerNotFound = errors.New("player not found")
var ErrNotOwner = errors.New("user is not the owner of the requested resource")
var ErrTableAssignment = errors.New("cannot assign players to tables")
var ErrInvalidScore = errors.New("invalid score")
var ErrRoundOrTableNotFound = errors.New("round or table not found")
var ErrTeamSizeNotAllowed = errors.New("team size not allowed")
