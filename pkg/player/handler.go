package player

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type PlayersHandler struct {
	playersService PlayersService
}

func NewPlayersHandler(playersService PlayersService) *PlayersHandler {
	return &PlayersHandler{playersService}
}

func (h *PlayersHandler) GetPlayers(c *gin.Context) {
	players, err := h.playersService.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"players": players})
}
