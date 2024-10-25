package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/henok321/knobel-manager-service/pkg/team"
)

type TeamsHandler interface {
	CreateTeam(c *gin.Context)
	UpdateTeam(c *gin.Context)
	DeleteTeam(c *gin.Context)
}
type teamsHandler struct {
	service team.TeamsService
}

func NewTeamsHandler(service team.TeamsService) TeamsHandler {
	return &teamsHandler{service}
}

func (h *teamsHandler) CreateTeam(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)
	gameID, err := strconv.ParseUint(c.Param("gameID"), 10, 64)
	if err != nil {
		_ = c.Error(err)
		return
	}

	request := team.TeamsRequest{}

	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(err)
		return
	}

	createdTeam, err := h.service.CreateTeam(uint(gameID), sub, request)

	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"team": createdTeam})
}

func (h *teamsHandler) UpdateTeam(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)
	gameID, err := strconv.ParseUint(c.Param("gameID"), 10, 64)

	if err != nil {
		_ = c.Error(err)
		return
	}

	teamID, err := strconv.ParseUint(c.Param("teamID"), 10, 64)

	if err != nil {
		_ = c.Error(err)
		return
	}

	request := team.TeamsRequest{}

	err = c.ShouldBindJSON(&request)

	if err != nil {
		_ = c.Error(err)
		return
	}

	updatedTeam, err := h.service.UpdateTeam(uint(gameID), sub, uint(teamID), request)

	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"team": updatedTeam})

}

func (h *teamsHandler) DeleteTeam(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)
	gameID, err := strconv.ParseUint(c.Param("gameID"), 10, 64)
	if err != nil {
		_ = c.Error(err)
		return
	}

	teamID, err := strconv.ParseUint(c.Param("teamID"), 10, 64)
	if err != nil {
		_ = c.Error(err)
		return
	}

	err = h.service.DeleteTeam(uint(gameID), sub, uint(teamID))

	if err != nil {
		_ = c.Error(err)
	}

	c.Status(http.StatusNoContent)
}
