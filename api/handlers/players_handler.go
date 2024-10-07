package handlers

import (
	"github.com/henok321/knobel-manager-service/pkg/player"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PlayersHandler interface {
	GetPlayers(c *gin.Context)
}

type playersHandler struct {
	playersService player.PlayersService
}

func NewPlayersHandler(playersService player.PlayersService) PlayersHandler {
	return &playersHandler{playersService}
}

func (h *playersHandler) GetPlayers(c *gin.Context) {
	players, err := h.playersService.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"players": players})
}
