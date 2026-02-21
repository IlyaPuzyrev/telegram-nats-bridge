package main

import (
	"fmt"
	"log/slog"
	"runtime"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"golang.org/x/sync/errgroup"
)

type compiledRoute struct {
	condition     *vm.Program
	subjectType   RouteSubjectType
	subjectStatic string
	subjectExpr   *vm.Program
}

type Router struct {
	routes       []compiledRoute
	mode         string
	routeWorkers int
	logger       *slog.Logger
}

func NewRouter(routes []Route, mode string, routeWorkers int, logger *slog.Logger) (*Router, error) {
	compiledRoutes := make([]compiledRoute, len(routes))

	numWorkers := min(runtime.GOMAXPROCS(0), len(routes))

	eg := errgroup.Group{}
	eg.SetLimit(numWorkers)

	for i := range routes {
		eg.Go(func() error {
			route := routes[i]

			condition, err := expr.Compile(route.Condition, expr.Env(env), expr.AsBool())
			if err != nil {
				return fmt.Errorf("failed to compile condition for route[%d]: %w", i, err)
			}

			var subjectStatic string
			var subjectExpr *vm.Program

			switch route.Subject.Type {
			case SubjectTypeString:
				subjectStatic = route.Subject.Value
			case SubjectTypeExpr:
				subjectExpr, err = expr.Compile(route.Subject.Value, expr.Env(env))
				if err != nil {
					return fmt.Errorf("failed to compile subject expression for route[%d]: %w", i, err)
				}
			}

			compiledRoutes[i] = compiledRoute{
				condition:     condition,
				subjectType:   route.Subject.Type,
				subjectStatic: subjectStatic,
				subjectExpr:   subjectExpr,
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	logger.Info("router initialized",
		"mode", mode,
		"routes_count", len(compiledRoutes),
		"route_workers", routeWorkers)

	return &Router{
		routes:       compiledRoutes,
		mode:         mode,
		routeWorkers: routeWorkers,
		logger:       logger,
	}, nil
}

func (r *Router) Route(update Update) ([]string, error) {
	type routingResult struct {
		idx  int
		cond bool
		subj string
		err  error
	}

	results := make([]routingResult, len(r.routes))
	resCh := make(chan routingResult, r.routeWorkers)

	var wg sync.WaitGroup

	for i := 0; i < len(r.routes); i += r.routeWorkers {
		batchSize := min(r.routeWorkers, len(r.routes)-i)

		for j := range batchSize {
			idx := i + j
			route := r.routes[idx]

			wg.Go(func() {
				cond, err := runExpr[bool](route.condition, update)
				if err != nil {
					resCh <- routingResult{idx: idx, err: err}
					return
				}

				if !cond {
					resCh <- routingResult{idx: idx, cond: false}
					return
				}

				subj := ""
				switch route.subjectType {
				case SubjectTypeString:
					subj = route.subjectStatic
				case SubjectTypeExpr:
					subj, err = runExpr[string](route.subjectExpr, update)
					if err != nil {
						resCh <- routingResult{idx: idx, err: err}
						return
					}
				}

				resCh <- routingResult{idx: idx, cond: true, subj: subj}
			})
		}

		wg.Wait()

		var match bool
		for range batchSize {
			rr := <-resCh
			if rr.err != nil {
				return nil, rr.err
			}
			results[rr.idx] = rr
			match = match || rr.cond
		}

		if r.mode == "first" && match {
			for _, rr := range results {
				if rr.cond {
					return []string{rr.subj}, nil
				}
			}
		}
	}

	seen := make(map[string]bool)
	var final []string

	for _, rr := range results {
		if rr.cond && !seen[rr.subj] {
			seen[rr.subj] = true
			final = append(final, rr.subj)
		}
	}

	return final, nil
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
