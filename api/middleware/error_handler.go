// api/middleware/error_handler.go
package middleware

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/gin-gonic/gin"
	"github.com/henok321/knobel-manager-service/pkg/entity"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if c.Writer.Written() {
			// Response has already been written by the handler
			return
		}
		if len(c.Errors) > 0 {
			err := c.Errors[0].Err
			switch {
			case errors.Is(err, entity.ErrorGameNotFound):
				c.JSON(http.StatusNotFound, gin.H{"error": "game not found"})
			case errors.Is(err, entity.ErrorTeamNotFound):
				c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
			case errors.Is(err, entity.ErrorNotOwner):
				c.JSON(http.StatusForbidden, gin.H{"error": "user is not the owner of the game"})
			default:
				// Handle validation errors and other errors
				var validationErr validator.ValidationErrors
				if errors.As(err, &validationErr) {
					c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				}
			}
		}
	}
}
