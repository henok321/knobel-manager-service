package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/henok321/knobel-manager-service/pkg/game"

	"github.com/gin-gonic/gin"
)

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

	games, err := h.service.FindAllByOwner(sub)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"games": games})
}

func (h *gamesHandler) GetGameByID(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)
	id := c.Param("id")

	idUint, err := strconv.ParseUint(id, 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	gameById, err := h.service.FindByID(uint(idUint), sub)
	if err != nil {

		if errors.Is(err, game.ErrorGameNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "game not found"})
			return
		} else if errors.Is(err, game.ErrorNotOwner) {
			c.JSON(http.StatusForbidden, gin.H{"error": "user is not the owner of the game"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"game": gameById})
}

func (h *gamesHandler) CreateGame(c *gin.Context) {

	contentType := c.GetHeader("Content-Type")

	if contentType != "application/json" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content type"})
		return
	}

	sub := c.GetStringMap("user")["sub"].(string)

	gameCreateRequest := game.GameRequest{}

	if err := c.ShouldBindJSON(&gameCreateRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createdGame, err := h.service.CreateGame(sub, &gameCreateRequest)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"game": createdGame})
	c.Header("Location", "/games/"+strconv.Itoa(int(createdGame.ID)))
}

func (h *gamesHandler) UpdateGame(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)
	id := c.Param("id")

	gameID, err := strconv.ParseUint(id, 10, 64)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	gameUpdateRequest := game.GameRequest{}

	if err := c.ShouldBindJSON(&gameUpdateRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedGame, err := h.service.UpdateGame(uint(gameID), sub, gameUpdateRequest)

	if err != nil {

		if errors.Is(err, game.ErrorGameNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "game not found"})
			return
		}
		if errors.Is(err, game.ErrorNotOwner) {
			c.JSON(http.StatusForbidden, gin.H{"error": "user is not the owner of the game"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"game": updatedGame})
}

func (h *gamesHandler) DeleteGame(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)
	id := c.Param("id")

	gameID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	err = h.service.DeleteGame(uint(gameID), sub)

	if err != nil {
		if errors.Is(err, game.ErrorGameNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "game not found"})
			return
		}
		if errors.Is(err, game.ErrorNotOwner) {
			c.JSON(http.StatusForbidden, gin.H{"error": "user is not the owner of the game"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
