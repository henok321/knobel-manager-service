package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/henok321/knobel-manager-service/pkg/team"
)

type TeamsHandler interface {
	CreateTeam(c *gin.Context)
	UpdateTeam(c *gin.Context)
	DeleteTeam(c *gin.Context)
}
type teamsHandler struct {
	service team.TeamsService
}

func NewTeamsHandler(service team.TeamsService) TeamsHandler {
	return &teamsHandler{service}
}

func (h *teamsHandler) CreateTeam(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func (h *teamsHandler) UpdateTeam(c *gin.Context) {
	c.Status(http.StatusNotImplemented)

}

func (h *teamsHandler) DeleteTeam(c *gin.Context) {
	c.Status(http.StatusNotImplemented)

}
