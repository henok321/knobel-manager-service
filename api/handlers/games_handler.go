package handlers

import (
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
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"games": games})
}

func (h *gamesHandler) GetGameByID(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)
	gameID, err := strconv.ParseUint(c.Param("gameID"), 10, 64)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid gameID"})
		return
	}

	gameById, err := h.service.FindByID(uint(gameID), sub)

	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"game": gameById})
}

func (h *gamesHandler) CreateGame(c *gin.Context) {

	sub := c.GetStringMap("user")["sub"].(string)

	gameCreateRequest := game.GameRequest{}

	if err := c.ShouldBindJSON(&gameCreateRequest); err != nil {
		_ = c.Error(err)
		return
	}

	createdGame, err := h.service.CreateGame(sub, &gameCreateRequest)

	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"game": createdGame})
	c.Header("Location", "/games/"+strconv.Itoa(int(createdGame.ID)))
}

func (h *gamesHandler) UpdateGame(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)
	gameID, err := strconv.ParseUint(c.Param("gameID"), 10, 64)

	if err != nil {
		_ = c.Error(err)
		return
	}

	gameUpdateRequest := game.GameRequest{}

	if err := c.ShouldBindJSON(&gameUpdateRequest); err != nil {
		_ = c.Error(err)
		return
	}

	updatedGame, err := h.service.UpdateGame(uint(gameID), sub, gameUpdateRequest)

	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"game": updatedGame})
}

func (h *gamesHandler) DeleteGame(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)
	gameID, err := strconv.ParseUint(c.Param("gameID"), 10, 64)
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.service.DeleteGame(uint(gameID), sub); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
