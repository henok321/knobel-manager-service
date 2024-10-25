package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/henok321/knobel-manager-service/pkg/model"

	"github.com/henok321/knobel-manager-service/pkg/game"

	"github.com/gin-gonic/gin"
)

type GameRequest struct {
	Name           string `json:"name" binding:"required"`
	TeamSize       uint   `json:"teamSize" binding:"required,min=4"`
	TableSize      uint   `json:"tableSize" binding:"required,min=4"`
	NumberOfRounds uint   `json:"numberOfRounds" binding:"required,min=1"`
}

type GamesHandler interface {
	GetGames(c *gin.Context)
	GetGameByID(c *gin.Context)
	CreateGame(c *gin.Context)
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

	requestBody := GameRequest{}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newGame := model.Game{
		Name:           requestBody.Name,
		TeamSize:       requestBody.TeamSize,
		TableSize:      requestBody.TableSize,
		NumberOfRounds: requestBody.NumberOfRounds,
	}

	createdGame, err := h.service.CreateGame(&newGame, sub)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"game": createdGame})
	c.Header("Location", "/games/"+strconv.Itoa(int(createdGame.ID)))

}
