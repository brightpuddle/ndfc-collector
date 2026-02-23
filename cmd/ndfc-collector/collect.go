package main

import (
	"regexp"
	"sync"

	"github.com/brightpuddle/gobits/log"
	"github.com/tidwall/gjson"
	"golang.org/x/sync/errgroup"

	"ndfc-collector/pkg/archive"
	"ndfc-collector/pkg/cli"
	"ndfc-collector/pkg/config"
	"ndfc-collector/pkg/ndfc"
	"ndfc-collector/pkg/req"
)

// resolvedReq is a request with all {placeholder} values substituted.
type resolvedReq struct {
	template req.Request       // original template (for child lookup)
	url      string            // resolved URL (placeholders filled in)
	ctx      map[string]string // accumulated placeholder context from ancestor items
}

// parentResult pairs the accumulated context that produced a response with that response.
type parentResult struct {
	ctx    map[string]string
	result gjson.Result
}

// placeholderRe matches {placeholder} patterns in URLs.
var placeholderRe = regexp.MustCompile(`\{([^}]+)\}`)

// substituteURL replaces {key} placeholders in url using values from ctx.
// Placeholders with no matching key are left unchanged.
func substituteURL(url string, ctx map[string]string) string {
	return placeholderRe.ReplaceAllStringFunc(url, func(match string) string {
		key := match[1 : len(match)-1]
		if val, ok := ctx[key]; ok {
			return val
		}
		return match
	})
}

// mergeCtx returns a new context containing all entries from parent merged with
// all top-level scalar fields from item (item values take precedence).
func mergeCtx(parent map[string]string, item gjson.Result) map[string]string {
	ctx := make(map[string]string, len(parent))
	for k, v := range parent {
		ctx[k] = v
	}
	item.ForEach(func(key, value gjson.Result) bool {
		if value.Type != gjson.JSON {
			ctx[key.String()] = value.String()
		}
		return true
	})
	return ctx
}

// buildLevels groups requests into topological depth levels so that
// every parent request is in a strictly earlier level than its children.
func buildLevels(reqs []req.Request) [][]req.Request {
	byURL := make(map[string]req.Request, len(reqs))
	for _, r := range reqs {
		byURL[r.URL] = r
	}

	depth := make(map[string]int, len(reqs))
	var calcDepth func(url string) int
	calcDepth = func(url string) int {
		if d, ok := depth[url]; ok {
			return d
		}
		r, ok := byURL[url]
		if !ok || r.DependsOn == "" {
			depth[url] = 0
			return 0
		}
		d := calcDepth(r.DependsOn) + 1
		depth[url] = d
		return d
	}

	maxDepth := 0
	for _, r := range reqs {
		if d := calcDepth(r.URL); d > maxDepth {
			maxDepth = d
		}
	}

	levels := make([][]req.Request, maxDepth+1)
	for _, r := range reqs {
		d := depth[r.URL]
		levels[d] = append(levels[d], r)
	}
	return levels
}

// expandLevel produces a resolved request for every combination of parent
// result items and child request templates at the given level.
// Root requests (DependsOn == "") produce exactly one resolved request each.
func expandLevel(
	levelReqs []req.Request,
	allParentResults map[string][]parentResult,
) []resolvedReq {
	var expanded []resolvedReq

	for _, r := range levelReqs {
		if r.DependsOn == "" {
			expanded = append(expanded, resolvedReq{
				template: r,
				url:      r.URL,
				ctx:      map[string]string{},
			})
			continue
		}

		for _, parent := range allParentResults[r.DependsOn] {
			expand := func(item gjson.Result) {
				ctx := mergeCtx(parent.ctx, item)
				expanded = append(expanded, resolvedReq{
					template: r,
					url:      substituteURL(r.URL, ctx),
					ctx:      ctx,
				})
			}
			if parent.result.IsArray() {
				parent.result.ForEach(func(_, item gjson.Result) bool {
					expand(item)
					return true
				})
			} else if parent.result.IsObject() {
				expand(parent.result)
			}
		}
	}

	return expanded
}

// collectFabric executes all requests in topological dependency order.
// Within each dependency level the expanded requests are batched and run
// in parallel (up to cfg.GetBatchSize() concurrent requests), preserving the
// original homegrown batching behaviour.
func collectFabric(
	client ndfc.Client,
	arc archive.Writer,
	reqs []req.Request,
	cfg config.FabricConfig,
) error {
	var logger log.Logger
	if cfg.GetFabricName() != "" {
		logger = log.With().Str("fabric", cfg.GetFabricName()).Logger()
	} else {
		logger = log.New()
	}

	levels := buildLevels(reqs)

	// allParentResults accumulates responses keyed by URL template for use
	// when expanding child requests in later levels.
	allParentResults := map[string][]parentResult{}

	var firstErr error

	for levelIdx, levelReqs := range levels {
		expanded := expandLevel(levelReqs, allParentResults)
		if len(expanded) == 0 {
			continue
		}

		logger.Info().Msgf("Fetching request level %d (%d requests)", levelIdx, len(expanded))

		type levelResult struct {
			r   resolvedReq
			res gjson.Result
		}

		var resultMu sync.Mutex
		levelResults := make([]levelResult, 0, len(expanded))

		batchSize := cfg.GetBatchSize()
		for i := 0; i < len(expanded); i += batchSize {
			end := i + batchSize
			if end > len(expanded) {
				end = len(expanded)
			}

			var g errgroup.Group
			for _, er := range expanded[i:end] {
				er := er
				g.Go(func() error {
					fetchReq := er.template
					fetchReq.URL = er.url

					res, err := cli.FetchResult(client, fetchReq, arc, cfg)
					if err != nil {
						return err
					}

					resultMu.Lock()
					levelResults = append(levelResults, levelResult{er, res})
					resultMu.Unlock()
					return nil
				})
			}

			if err := g.Wait(); err != nil {
				logger.Error().Err(err).Msg("Error fetching data.")
				if firstErr == nil {
					firstErr = err
				}
			}
		}

		for _, lr := range levelResults {
			allParentResults[lr.r.template.URL] = append(
				allParentResults[lr.r.template.URL],
				parentResult{ctx: lr.r.ctx, result: lr.res},
			)
		}
	}

	return firstErr
}
