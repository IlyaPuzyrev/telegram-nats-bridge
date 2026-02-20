package main

import (
	"fmt"
	"log/slog"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

type compiledRoute struct {
	condition     *vm.Program
	subjectType   RouteSubjectType
	subjectStatic string
	subjectExpr   *vm.Program
}

type Router struct {
	routes []compiledRoute
	mode   string
	logger *slog.Logger
}

func NewRouter(routes []Route, mode string, logger *slog.Logger) (*Router, error) {
	compiledRoutes := make([]compiledRoute, 0, len(routes))

	for i, route := range routes {
		condition, err := expr.Compile(route.Condition, expr.Env(env), expr.AsBool())
		if err != nil {
			return nil, fmt.Errorf("failed to compile condition for route[%d]: %w", i, err)
		}

		var subjectStatic string
		var subjectExpr *vm.Program

		switch route.Subject.Type {
		case SubjectTypeString:
			subjectStatic = route.Subject.Value
		case SubjectTypeExpr:
			subjectExpr, err = expr.Compile(route.Subject.Value, expr.Env(env))
			if err != nil {
				return nil, fmt.Errorf("failed to compile subject expression for route[%d]: %w", i, err)
			}
		}

		compiledRoutes = append(compiledRoutes, compiledRoute{
			condition:     condition,
			subjectType:   route.Subject.Type,
			subjectStatic: subjectStatic,
			subjectExpr:   subjectExpr,
		})
	}

	logger.Info("router initialized",
		"mode", mode,
		"routes_count", len(compiledRoutes))

	return &Router{
		routes: compiledRoutes,
		mode:   mode,
		logger: logger,
	}, nil
}

func (r *Router) Route(update Update) (map[string]bool, error) {
	result := make(map[string]bool)

	for _, route := range r.routes {
		cond, err := runExpr[bool](route.condition, update)
		if err != nil {
			return nil, fmt.Errorf("condition evaluation error: %w", err)
		}

		if !cond {
			continue
		}

		var subject string

		switch route.subjectType {
		case SubjectTypeString:
			subject = route.subjectStatic
		case SubjectTypeExpr:
			subject, err = runExpr[string](route.subjectExpr, update)
			if err != nil {
				return nil, fmt.Errorf("subject expression evaluation error: %w", err)
			}
		}

		result[subject] = true

		if r.mode == "first" {
			break
		}
	}

	return result, nil
}

var env = map[string]interface{}{
	"sprintf": fmt.Sprintf,
	"update":  Update{},
}

func runExpr[T any](program *vm.Program, update Update) (T, error) {
	var zero T

	runEnv := map[string]interface{}{
		"sprintf": fmt.Sprintf,
		"update":  update,
	}

	output, err := expr.Run(program, runEnv)
	if err != nil {
		return zero, err
	}

	val, ok := output.(T)
	if !ok {
		return zero, fmt.Errorf("expected %T, got %T", zero, output)
	}

	return val, nil
}
