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
	topicType     RouteSubjectType
	topicStatic   string
	topicExpr     *vm.Program
	keyType       RouteSubjectType
	keyStatic     string
	keyExpr       *vm.Program
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

			if route.Subject != nil {
				switch route.Subject.Type {
				case SubjectTypeString:
					subjectStatic = route.Subject.Value
				case SubjectTypeExpr:
					subjectExpr, err = expr.Compile(route.Subject.Value, expr.Env(env))
					if err != nil {
						return fmt.Errorf("failed to compile subject expression for route[%d]: %w", i, err)
					}
				}
			}

			var topicStatic string
			var topicExpr *vm.Program

			if route.Topic != nil {
				switch route.Topic.Type {
				case SubjectTypeString:
					topicStatic = route.Topic.Value
				case SubjectTypeExpr:
					topicExpr, err = expr.Compile(route.Topic.Value, expr.Env(env))
					if err != nil {
						return fmt.Errorf("failed to compile topic expression for route[%d]: %w", i, err)
					}
				}
			}

			var keyStatic string
			var keyExpr *vm.Program

			if route.Key != nil {
				switch route.Key.Type {
				case SubjectTypeString:
					keyStatic = route.Key.Value
				case SubjectTypeExpr:
					keyExpr, err = expr.Compile(route.Key.Value, expr.Env(env))
					if err != nil {
						return fmt.Errorf("failed to compile key expression for route[%d]: %w", i, err)
					}
				}
			}

			subjectType := RouteSubjectType("")
			if route.Subject != nil {
				subjectType = route.Subject.Type
			}
			topicType := RouteSubjectType("")
			if route.Topic != nil {
				topicType = route.Topic.Type
			}
			keyType := RouteSubjectType("")
			if route.Key != nil {
				keyType = route.Key.Type
			}

			compiledRoutes[i] = compiledRoute{
				condition:     condition,
				subjectType:   subjectType,
				subjectStatic: subjectStatic,
				subjectExpr:   subjectExpr,
				topicType:     topicType,
				topicStatic:   topicStatic,
				topicExpr:     topicExpr,
				keyType:       keyType,
				keyStatic:     keyStatic,
				keyExpr:       keyExpr,
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

func (r *Router) Route(update Update) ([]Destination, error) {
	type routingResult struct {
		idx  int
		cond bool
		dest Destination
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

				dest := Destination{}

				if route.subjectExpr != nil || route.subjectStatic != "" {
					switch route.subjectType {
					case SubjectTypeString:
						dest.Subject = route.subjectStatic
					case SubjectTypeExpr:
						dest.Subject, err = runExpr[string](route.subjectExpr, update)
						if err != nil {
							resCh <- routingResult{idx: idx, err: err}
							return
						}
					}
				}

				if route.topicExpr != nil || route.topicStatic != "" {
					switch route.topicType {
					case SubjectTypeString:
						dest.Topic = route.topicStatic
					case SubjectTypeExpr:
						dest.Topic, err = runExpr[string](route.topicExpr, update)
						if err != nil {
							resCh <- routingResult{idx: idx, err: err}
							return
						}
					}
				}

				if route.keyExpr != nil || route.keyStatic != "" {
					switch route.keyType {
					case SubjectTypeString:
						dest.Key = route.keyStatic
					case SubjectTypeExpr:
						dest.Key, err = runExpr[string](route.keyExpr, update)
						if err != nil {
							resCh <- routingResult{idx: idx, err: err}
							return
						}
					}
				}

				resCh <- routingResult{idx: idx, cond: true, dest: dest}
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
					return []Destination{rr.dest}, nil
				}
			}
		}
	}

	seen := make(map[string]bool)
	var final []Destination

	for _, rr := range results {
		if rr.cond {
			key := rr.dest.Subject + rr.dest.Topic + rr.dest.Key
			if !seen[key] {
				seen[key] = true
				final = append(final, rr.dest)
			}
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
