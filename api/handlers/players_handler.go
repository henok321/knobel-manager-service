package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/henok321/knobel-manager-service/pkg/entity"

	"github.com/gin-gonic/gin"
	"github.com/henok321/knobel-manager-service/pkg/player"
)

type PlayersHandler interface {
	CreatePlayer(c *gin.Context)
	UpdatePlayer(c *gin.Context)
	DeletePlayer(c *gin.Context)
}

type playersHandler struct {
	playersService player.PlayersService
}

func NewPlayersHandler(service player.PlayersService) PlayersHandler {
	return playersHandler{
		playersService: service,
	}
}

func (h playersHandler) CreatePlayer(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)

	gameID, err := strconv.ParseUint(c.Param("gameID"), 10, 64)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid gameID"})
		return
	}

	teamID, err := strconv.ParseUint(c.Param("teamID"), 10, 64)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid teamID"})
		return
	}

	request := player.PlayersRequest{}

	err = c.ShouldBindJSON(&request)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createPlayer, err := h.playersService.CreatePlayer(request, uint(teamID), sub)

	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"player": createPlayer})
	c.Header("Location", fmt.Sprintf("/games/%d/teams/%d/players/%d", gameID, teamID, createPlayer.ID))
}

func (h playersHandler) UpdatePlayer(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)

	playerID, err := strconv.ParseUint(c.Param("playerID"), 10, 64)

	if err != nil {

		c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	request := player.PlayersRequest{}

	err = c.ShouldBindJSON(&request)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedPlayer, err := h.playersService.UpdatePlayer(uint(playerID), request, sub)

	if err != nil {
		if errors.Is(err, entity.ErrorTeamNotFound) || errors.Is(err, entity.ErrorPlayerNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	responseBody := player.PlayersResponse{ID: updatedPlayer.ID, Name: updatedPlayer.Name}

	c.JSON(http.StatusOK, gin.H{"player": responseBody})
}

func (h playersHandler) DeletePlayer(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)

	playerID, err := strconv.ParseUint(c.Param("playerID"), 10, 64)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid playerID"})
		return
	}

	err = h.playersService.DeletePlayer(uint(playerID), sub)

	if err != nil {
		if errors.Is(err, entity.ErrorPlayerNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.AbortWithStatus(http.StatusNoContent)
}
