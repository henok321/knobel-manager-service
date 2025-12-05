package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
)

type validationErrorResponse struct {
	Error string `json:"error"`
}

func writeJSONError(w http.ResponseWriter, errorMessage string, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(&validationErrorResponse{Error: errorMessage})
}

func RequestValidation(router routers.Router, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		route, pathParams, err := router.FindRoute(request)
		if err != nil {
			next.ServeHTTP(writer, request.WithContext(ctx))
			return
		}

		reqInput := &openapi3filter.RequestValidationInput{
			Request:    request,
			Route:      route,
			PathParams: pathParams,
			Options: &openapi3filter.Options{
				AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
			},
		}

		err = openapi3filter.ValidateRequest(ctx, reqInput)
		if err != nil {
			writeJSONError(writer, err.Error(), http.StatusBadRequest)
			return
		}

		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}
