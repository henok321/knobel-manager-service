package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func GetPlayers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"players": []string{"Alice", "Bob", "Charlie"},
	})
}
