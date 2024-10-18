package handlers

import (
	"net/http"
	"strconv"

	"github.com/henok321/knobel-manager-service/pkg/player"

	"github.com/gin-gonic/gin"
)

type PlayersHandler interface {
	GetPlayersByGame(c *gin.Context)
	GetPlayersByTeam(c *gin.Context)
}

type playersHandler struct {
	playersService player.PlayersService
}

func NewPlayersHandler(playersService player.PlayersService) PlayersHandler {
	return &playersHandler{playersService}
}

func (h *playersHandler) GetPlayersByGame(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)

	gameID, err := strconv.ParseUint(c.Param("gameID"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid game ID"})
		return
	}

	players, err := h.playersService.FindByGame(uint(gameID), sub)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"players": players})
}

func (h *playersHandler) GetPlayersByTeam(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)

	gameID, err := strconv.ParseUint(c.Param("gameID"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid game ID"})
		return
	}

	teamID, err := strconv.ParseUint(c.Param("teamID"), 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}

	players, err := h.playersService.FindByTeam(uint(gameID), uint(teamID), sub)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"players": players})

}
