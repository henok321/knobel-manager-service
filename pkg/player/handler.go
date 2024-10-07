package player

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type PlayersHandler interface {
	GetPlayers(c *gin.Context)
}

type playersHandler struct {
	playersService PlayersService
}

func NewPlayersHandler(playersService PlayersService) PlayersHandler {
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
