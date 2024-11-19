package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/henok321/knobel-manager-service/pkg/entity"

	"github.com/gin-gonic/gin"
	"github.com/henok321/knobel-manager-service/pkg/game"
)

type TablesHandler interface {
	GetTables(c *gin.Context)
	GetTable(c *gin.Context)
}

type tablesHandler struct {
	gamesService game.GamesService
}

func NewTablesHandler(gamesService game.GamesService) TablesHandler {
	return tablesHandler{gamesService: gamesService}
}

func (t tablesHandler) GetTables(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)

	gameID, err := strconv.ParseInt(c.Param("gameID"), 10, 64)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid gameID"})
		return
	}

	roundNumber, err := strconv.ParseInt(c.Param("roundNumber"), 10, 64)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid roundNumber"})
		return
	}

	gameById, err := t.gamesService.FindByID(uint(gameID), sub)

	if err != nil {
		switch {
		case errors.Is(err, entity.ErrorNotOwner):
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, entity.ErrorGameNotFound):
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "game not found"})
		default:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	for _, round := range gameById.Rounds {
		if round.RoundNumber == uint(roundNumber) {
			tables := round.Tables
			c.JSON(http.StatusOK, tables)
			return
		}
	}

	c.AbortWithStatusJSON(404, gin.H{"error": "round not found"})
}

func (t tablesHandler) GetTable(c *gin.Context) {
	sub := c.GetStringMap("user")["sub"].(string)

	gameID, err := strconv.ParseInt(c.Param("gameID"), 10, 64)

	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid gameID"})
		return
	}

	roundNumber, err := strconv.ParseInt(c.Param("roundNumber"), 10, 64)

	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid round number"})
		return
	}

	tableNumber, err := strconv.ParseInt(c.Param("tableNumber"), 10, 64)

	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"error": "invalid table number"})
	}

	gameById, err := t.gamesService.FindByID(uint(gameID), sub)

	if err != nil {
		switch {
		case errors.Is(err, entity.ErrorNotOwner):
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		case errors.Is(err, entity.ErrorGameNotFound):
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "game not found"})
			return
		default:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	for _, round := range gameById.Rounds {
		if round.RoundNumber == uint(roundNumber) {
			for _, table := range round.Tables {
				if table.TableNumber == uint(tableNumber) {
					c.JSON(http.StatusOK, table)
					return
				}
			}
		}
	}

	c.AbortWithStatusJSON(404, gin.H{"error": "round or table not found"})
}
