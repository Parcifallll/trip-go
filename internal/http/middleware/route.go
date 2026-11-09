package middleware

import (
	"context"
	"net/http"
)

const unmatchedRoute = "unmatched"

type RouteResolver func(r *http.Request) string

type routeKey struct{}

func withRoute(ctx context.Context, route string) context.Context {
	return context.WithValue(ctx, routeKey{}, route)
}

func RouteFromContext(ctx context.Context) string {
	if route, ok := ctx.Value(routeKey{}).(string); ok && route != "" {
		return route
	}
	return unmatchedRoute
}

func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
