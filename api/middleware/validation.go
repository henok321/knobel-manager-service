package middleware

import (
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
)

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
			slog.DebugContext(ctx, "Request validation failed", "error", err)
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}
