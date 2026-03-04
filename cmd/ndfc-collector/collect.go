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
	"ndfc-collector/pkg/requests"
)

// resolvedReq is a request with all {placeholder} values substituted.
type resolvedReq struct {
	template    requests.Request  // original template (for child lookup)
	url         string            // resolved URL (placeholders filled in)
	resolvedKey string            // resolved db_key (placeholders filled in)
	ctx         map[string]string // accumulated placeholder context from ancestor items
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

// extractCtx returns a new context containing all entries from parent merged
// with values extracted from item using the provided key-to-placeholder
// mappings (item values take precedence; only explicitly mapped keys are added).
func extractCtx(
	parent map[string]string,
	item gjson.Result,
	keyMappings map[string]string,
) map[string]string {
	ctx := make(map[string]string, len(parent)+len(keyMappings))
	for k, v := range parent {
		ctx[k] = v
	}
	for placeholder, key := range keyMappings {
		if val := item.Get(key); val.Exists() && val.Type != gjson.JSON {
			ctx[placeholder] = val.String()
		}
	}
	return ctx
}

// cartesianCtx returns the Cartesian product of multiple slices of context maps.
// Each result is a slice containing one element from each input group.
func cartesianCtx(groups [][]map[string]string) [][]map[string]string {
	if len(groups) == 0 {
		return [][]map[string]string{{}}
	}
	rest := cartesianCtx(groups[1:])
	var result [][]map[string]string
	for _, item := range groups[0] {
		for _, r := range rest {
			combo := make([]map[string]string, 0, len(groups))
			combo = append(combo, item)
			combo = append(combo, r...)
			result = append(result, combo)
		}
	}
	return result
}

// buildLevels groups requests into topological depth levels so that
// every parent request is in a strictly earlier level than its children.
func buildLevels(reqs []requests.Request) [][]requests.Request {
	byURL := make(map[string]requests.Request, len(reqs))
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
		if !ok || len(r.DependsOn) == 0 {
			depth[url] = 0
			return 0
		}
		maxParentDepth := 0
		for _, dep := range r.DependsOn {
			if d := calcDepth(dep.URL); d > maxParentDepth {
				maxParentDepth = d
			}
		}
		d := maxParentDepth + 1
		depth[url] = d
		return d
	}

	maxDepth := 0
	for _, r := range reqs {
		if d := calcDepth(r.URL); d > maxDepth {
			maxDepth = d
		}
	}

	levels := make([][]requests.Request, maxDepth+1)
	for _, r := range reqs {
		d := depth[r.URL]
		levels[d] = append(levels[d], r)
	}
	return levels
}

// expandLevel produces a resolved request for every combination of parent
// result items and child request templates at the given level.
// Root requests (empty DependsOn) produce exactly one resolved request each.
// For dependent requests, each placeholder in DependsOn is resolved from the
// specified parent URL's response items using the mapped Key field name.
// When placeholders reference multiple parent URLs the combinations are
// expanded as a Cartesian product.
func expandLevel(
	levelReqs []requests.Request,
	allParentResults map[string][]parentResult,
) []resolvedReq {
	var expanded []resolvedReq

	for _, r := range levelReqs {
		if len(r.DependsOn) == 0 {
			expanded = append(expanded, resolvedReq{
				template:    r,
				url:         r.URL,
				resolvedKey: r.DBKey,
				ctx:         map[string]string{},
			})
			continue
		}

		// Group dependency entries by parent URL so we know which keys to
		// extract from each parent's response items.
		// byParentURL: parentURL -> {placeholder -> jsonKey}
		byParentURL := make(map[string]map[string]string)
		for placeholder, dep := range r.DependsOn {
			if byParentURL[dep.URL] == nil {
				byParentURL[dep.URL] = make(map[string]string)
			}
			byParentURL[dep.URL][placeholder] = dep.Key
		}

		// For each parent URL, expand its results into individual context maps
		// (one per response item) with the parent's accumulated ctx merged in.
		var groups [][]map[string]string
		for parentURL, keyMappings := range byParentURL {
			var ctxSets []map[string]string
			for _, pr := range allParentResults[parentURL] {
				process := func(item gjson.Result) {
					ctxSets = append(ctxSets, extractCtx(pr.ctx, item, keyMappings))
				}
				if pr.result.IsArray() {
					pr.result.ForEach(func(_, item gjson.Result) bool {
						process(item)
						return true
					})
				} else if pr.result.IsObject() {
					process(pr.result)
				}
			}
			groups = append(groups, ctxSets)
		}

		// Expand the Cartesian product of all parent groups and emit one
		// resolved request per combination.
		for _, combo := range cartesianCtx(groups) {
			mergedCtx := make(map[string]string)
			for _, ctx := range combo {
				for k, v := range ctx {
					mergedCtx[k] = v
				}
			}
			expanded = append(expanded, resolvedReq{
				template:    r,
				url:         substituteURL(r.URL, mergedCtx),
				resolvedKey: substituteURL(r.DBKey, mergedCtx),
				ctx:         mergedCtx,
			})
		}
	}

	return expanded
}

// collectFabric executes all requests in topological dependency order.
// Within each dependency level the expanded requests are batched and run
// in parallel (up to cfg.BatchSize concurrent requests), preserving the
// original homegrown batching behaviour.
func collectFabric(
	client ndfc.Client,
	arc archive.Writer,
	reqs []requests.Request,
	cfg *config.Config,
) error {
	logger := log.New()

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

		batchSize := cfg.BatchSize
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
					fetchReq.DBKey = er.resolvedKey

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
