package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/henok321/knobel-manager-service/pkg/entity"

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid gameID"})
		return
	}

	request := team.TeamsRequest{}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createdTeam, err := h.service.CreateTeam(uint(gameID), sub, request)

	if err != nil {
		if errors.Is(err, entity.ErrorGameNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "game not found"})
			return
		}
		if errors.Is(err, entity.ErrorTeamNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "team name already exists"})
		} else if errors.Is(err, entity.ErrorNotOwner) {
			c.JSON(http.StatusForbidden, gin.H{"error": "user is not the owner of the game"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"team": createdTeam})
}

func (h *teamsHandler) UpdateTeam(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)
	gameID, err := strconv.ParseUint(c.Param("gameID"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid gameID"})
		return
	}

	teamID, err := strconv.ParseUint(c.Param("teamID"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid teamID"})
		return
	}

	request := team.TeamsRequest{}
	err = c.ShouldBindJSON(&request)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedTeam, err := h.service.UpdateTeam(uint(gameID), sub, uint(teamID), request)

	if err != nil {
		if errors.Is(err, entity.ErrorGameNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "game not found"})
			return
		}
		if errors.Is(err, entity.ErrorTeamNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
			return
		}
		if errors.Is(err, entity.ErrorNotOwner) {
			c.JSON(http.StatusForbidden, gin.H{"error": "user is not the owner of the game"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"team": updatedTeam})

}

func (h *teamsHandler) DeleteTeam(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)
	gameID, err := strconv.ParseUint(c.Param("gameID"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	teamID, err := strconv.ParseUint(c.Param("teamID"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	err = h.service.DeleteTeam(uint(gameID), sub, uint(teamID))

	if err != nil {
		if errors.Is(err, entity.ErrorGameNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "game not found"})
			return
		}
		if errors.Is(err, entity.ErrorTeamNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
			return
		}
		if errors.Is(err, entity.ErrorNotOwner) {
			c.JSON(http.StatusForbidden, gin.H{"error": "user is not the owner of the game"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
