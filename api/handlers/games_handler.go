package handlers

import (
	"fmt"
	"github.com/henok321/knobel-manager-service/pkg/game"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func isOwner(sub string, owners []game.Owner) bool {
	for _, owner := range owners {
		if owner.Sub == sub {
			return true
		}
	}
	return false
}

type GamesHandler interface {
	GetGames(c *gin.Context)
	GetGameByID(c *gin.Context)
	CreateGame(c *gin.Context)
	UpdateGame(c *gin.Context)
	DeleteGame(c *gin.Context)
}

type gamesHandler struct {
	service game.GamesService
}

func NewGamesHandler(service game.GamesService) GamesHandler {
	return &gamesHandler{service}
}

func (h *gamesHandler) GetGames(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)

	if sub == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}

	games, err := h.service.FindAllByOwner(sub)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"games": games})
}

func (h *gamesHandler) GetGameByID(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)

	if sub == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gameById, err := h.service.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	if !isOwner(sub, gameById.Owners) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not the owner of this gameById"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"gameById": gameById})
}

func (h *gamesHandler) CreateGame(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)

	if sub == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}

	var createdGame game.Game
	if err := c.BindJSON(&createdGame); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createdGame.Owners = []game.Owner{{Sub: sub}}

	if err := h.service.Create(&createdGame); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Location", fmt.Sprintf("/games/%d", createdGame.ID))
	c.JSON(http.StatusCreated, gin.H{"createdGame": createdGame})
}

func (h *gamesHandler) UpdateGame(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)

	if sub == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedGame, err := h.service.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	if !isOwner(sub, updatedGame.Owners) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not the owner of this updatedGame"})
		return
	}

	if err := c.BindJSON(&updatedGame); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedGame.ID = uint(id)

	if err := h.service.Update(updatedGame); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"updatedGame": updatedGame})
}

func (h *gamesHandler) DeleteGame(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)

	if sub == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gameById, err := h.service.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	if !isOwner(sub, gameById.Owners) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not the owner of this gameById"})
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
