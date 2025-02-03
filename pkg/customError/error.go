package customError

import "errors"

var TeamNotFound = errors.New("team not found")
var PlayerNotFound = errors.New("player not found")
var NotOwner = errors.New("user is not the owner of the requested resource")
var TableAssignment = errors.New("cannot not assign players to tables")
var InvalidScore = errors.New("invalid score")
var RoundOrTableNotFound = errors.New("round or table not found")
var TeamSizeNotAllowed = errors.New("team size not allowed")
