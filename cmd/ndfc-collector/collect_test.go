package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"

	"ndfc-collector/pkg/req"
)

// --- substituteURL ---

func TestSubstituteURL_NoPlaceholders(t *testing.T) {
	url := "/lan-fabric/rest/control/fabrics"
	result := substituteURL(url, map[string]string{"fabricName": "f1"})
	assert.Equal(t, url, result)
}

func TestSubstituteURL_SinglePlaceholder(t *testing.T) {
	url := "/lan-fabric/rest/control/fabrics/{fabricName}/inventory/switchesByFabric"
	result := substituteURL(url, map[string]string{"fabricName": "MyFabric"})
	assert.Equal(t, "/lan-fabric/rest/control/fabrics/MyFabric/inventory/switchesByFabric", result)
}

func TestSubstituteURL_MultiplePlaceholders(t *testing.T) {
	url := "/fabrics/{fabricName}/switches/{serialNumber}/config"
	result := substituteURL(url, map[string]string{
		"fabricName":   "prod",
		"serialNumber": "SN001",
	})
	assert.Equal(t, "/fabrics/prod/switches/SN001/config", result)
}

func TestSubstituteURL_MissingKey_LeftUnchanged(t *testing.T) {
	url := "/fabrics/{fabricName}/switches/{serialNumber}/config"
	result := substituteURL(url, map[string]string{"fabricName": "prod"})
	assert.Equal(t, "/fabrics/prod/switches/{serialNumber}/config", result)
}

// --- mergeCtx ---

func TestMergeCtx_AddsScalarFields(t *testing.T) {
	parent := map[string]string{"fabricName": "f1"}
	item := gjson.Parse(`{"serialNumber":"SN1","switchName":"sw1","nested":{"x":1}}`)
	ctx := mergeCtx(parent, item)
	assert.Equal(t, "f1", ctx["fabricName"])
	assert.Equal(t, "SN1", ctx["serialNumber"])
	assert.Equal(t, "sw1", ctx["switchName"])
	_, hasNested := ctx["nested"]
	assert.False(t, hasNested, "nested JSON objects should not be added to context")
}

func TestMergeCtx_ItemOverridesParent(t *testing.T) {
	parent := map[string]string{"key": "old"}
	item := gjson.Parse(`{"key":"new"}`)
	ctx := mergeCtx(parent, item)
	assert.Equal(t, "new", ctx["key"])
}

func TestMergeCtx_NilParent(t *testing.T) {
	item := gjson.Parse(`{"fabricName":"f1"}`)
	ctx := mergeCtx(nil, item)
	assert.Equal(t, "f1", ctx["fabricName"])
}

// --- buildLevels ---

func TestBuildLevels_RootsOnly(t *testing.T) {
	reqs := []req.Request{
		{URL: "/a"},
		{URL: "/b"},
	}
	levels := buildLevels(reqs)
	assert.Len(t, levels, 1)
	assert.Len(t, levels[0], 2)
}

func TestBuildLevels_OneDependency(t *testing.T) {
	reqs := []req.Request{
		{URL: "/fabrics"},
		{URL: "/fabrics/{fabricName}/switches", DependsOn: "/fabrics"},
	}
	levels := buildLevels(reqs)
	assert.Len(t, levels, 2)

	level0URLs := urlsFromLevel(levels[0])
	assert.Contains(t, level0URLs, "/fabrics")

	level1URLs := urlsFromLevel(levels[1])
	assert.Contains(t, level1URLs, "/fabrics/{fabricName}/switches")
}

func TestBuildLevels_Chain(t *testing.T) {
	reqs := []req.Request{
		{URL: "/a"},
		{URL: "/a/{x}/b", DependsOn: "/a"},
		{URL: "/a/{x}/b/{y}/c", DependsOn: "/a/{x}/b"},
	}
	levels := buildLevels(reqs)
	assert.Len(t, levels, 3)
	assert.Contains(t, urlsFromLevel(levels[0]), "/a")
	assert.Contains(t, urlsFromLevel(levels[1]), "/a/{x}/b")
	assert.Contains(t, urlsFromLevel(levels[2]), "/a/{x}/b/{y}/c")
}

// --- expandLevel ---

func TestExpandLevel_RootRequest(t *testing.T) {
	levelReqs := []req.Request{{URL: "/fabrics"}}
	expanded := expandLevel(levelReqs, map[string][]parentResult{})
	assert.Len(t, expanded, 1)
	assert.Equal(t, "/fabrics", expanded[0].url)
}

func TestExpandLevel_DependentRequest_Array(t *testing.T) {
	levelReqs := []req.Request{
		{URL: "/fabrics/{fabricName}/switches", DependsOn: "/fabrics"},
	}
	parentResults := map[string][]parentResult{
		"/fabrics": {
			{
				ctx:    map[string]string{},
				result: gjson.Parse(`[{"fabricName":"f1"},{"fabricName":"f2"}]`),
			},
		},
	}
	expanded := expandLevel(levelReqs, parentResults)
	assert.Len(t, expanded, 2)

	urls := make([]string, len(expanded))
	for i, e := range expanded {
		urls[i] = e.url
	}
	assert.Contains(t, urls, "/fabrics/f1/switches")
	assert.Contains(t, urls, "/fabrics/f2/switches")
}

func TestExpandLevel_DependentRequest_Object(t *testing.T) {
	levelReqs := []req.Request{
		{URL: "/fabrics/{fabricName}/detail", DependsOn: "/fabrics"},
	}
	parentResults := map[string][]parentResult{
		"/fabrics": {
			{
				ctx:    map[string]string{},
				result: gjson.Parse(`{"fabricName":"prod"}`),
			},
		},
	}
	expanded := expandLevel(levelReqs, parentResults)
	assert.Len(t, expanded, 1)
	assert.Equal(t, "/fabrics/prod/detail", expanded[0].url)
}

func TestExpandLevel_DependentRequest_ContextPropagated(t *testing.T) {
	// Simulate level-2 expansion: /a/{x}/b/{y}/c depends on /a/{x}/b,
	// which was itself expanded from /a with fabricName=f1.
	levelReqs := []req.Request{
		{URL: "/fabrics/{fabricName}/switches/{serialNumber}/config", DependsOn: "/fabrics/{fabricName}/switches"},
	}
	parentResults := map[string][]parentResult{
		"/fabrics/{fabricName}/switches": {
			{
				ctx:    map[string]string{"fabricName": "f1"},
				result: gjson.Parse(`[{"serialNumber":"SN1"},{"serialNumber":"SN2"}]`),
			},
		},
	}
	expanded := expandLevel(levelReqs, parentResults)
	assert.Len(t, expanded, 2)

	urls := make([]string, len(expanded))
	for i, e := range expanded {
		urls[i] = e.url
	}
	assert.Contains(t, urls, "/fabrics/f1/switches/SN1/config")
	assert.Contains(t, urls, "/fabrics/f1/switches/SN2/config")
}

func TestExpandLevel_NoParentResults_ProducesNothing(t *testing.T) {
	// If the parent fetch failed (no results stored), children are skipped.
	levelReqs := []req.Request{
		{URL: "/fabrics/{fabricName}/switches", DependsOn: "/fabrics"},
	}
	expanded := expandLevel(levelReqs, map[string][]parentResult{})
	assert.Empty(t, expanded)
}

// helpers

func urlsFromLevel(level []req.Request) []string {
	urls := make([]string, len(level))
	for i, r := range level {
		urls[i] = r.URL
	}
	return urls
}
