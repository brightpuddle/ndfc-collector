package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"

	"ndfc-collector/pkg/requests"
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

func TestSubstituteQuery(t *testing.T) {
	query := map[string]string{
		"fabricName": "{fabricName}",
		"static":     "value",
	}
	result := substituteQuery(query, map[string]string{"fabricName": "prod"})
	assert.Equal(t, map[string]string{
		"fabricName": "prod",
		"static":     "value",
	}, result)
}

// --- extractCtx ---

func TestExtractCtx_MapsSpecifiedKeys(t *testing.T) {
	parent := map[string]string{"fabricName": "f1"}
	item := gjson.Parse(`{"serialNumber":"SN1","switchName":"sw1","nested":{"x":1}}`)
	keyMappings := map[string]string{"serialNumber": "serialNumber", "switchName": "switchName"}
	ctx := extractCtx(parent, item, keyMappings)
	assert.Equal(t, "f1", ctx["fabricName"])
	assert.Equal(t, "SN1", ctx["serialNumber"])
	assert.Equal(t, "sw1", ctx["switchName"])
	_, hasNested := ctx["nested"]
	assert.False(t, hasNested, "nested JSON objects should not be added to context")
}

func TestExtractCtx_KeyRemapping(t *testing.T) {
	// placeholder "fabricName" resolved from JSON key "name"
	parent := map[string]string{}
	item := gjson.Parse(`{"name":"MyFabric"}`)
	keyMappings := map[string]string{"fabricName": "name"}
	ctx := extractCtx(parent, item, keyMappings)
	assert.Equal(t, "MyFabric", ctx["fabricName"])
	_, hasName := ctx["name"]
	assert.False(t, hasName, "unmapped JSON key should not appear in context")
}

func TestExtractCtx_ParentCtxPropagated(t *testing.T) {
	parent := map[string]string{"fabricName": "f1"}
	item := gjson.Parse(`{"serialNumber":"SN1"}`)
	keyMappings := map[string]string{"serialNumber": "serialNumber"}
	ctx := extractCtx(parent, item, keyMappings)
	assert.Equal(t, "f1", ctx["fabricName"])
	assert.Equal(t, "SN1", ctx["serialNumber"])
}

func TestExtractCtx_ItemOverridesParent(t *testing.T) {
	parent := map[string]string{"key": "old"}
	item := gjson.Parse(`{"newKey":"new"}`)
	keyMappings := map[string]string{"key": "newKey"}
	ctx := extractCtx(parent, item, keyMappings)
	assert.Equal(t, "new", ctx["key"])
}

func TestExtractCtx_NilParent(t *testing.T) {
	item := gjson.Parse(`{"fabricName":"f1"}`)
	keyMappings := map[string]string{"fabricName": "fabricName"}
	ctx := extractCtx(nil, item, keyMappings)
	assert.Equal(t, "f1", ctx["fabricName"])
}

// --- cartesianCtx ---

func TestCartesianCtx_EmptyGroups(t *testing.T) {
	result := cartesianCtx(nil)
	assert.Len(t, result, 1)
	assert.Empty(t, result[0])
}

func TestCartesianCtx_SingleGroup(t *testing.T) {
	groups := [][]map[string]string{
		{{"a": "1"}, {"a": "2"}},
	}
	result := cartesianCtx(groups)
	assert.Len(t, result, 2)
}

func TestCartesianCtx_TwoGroups(t *testing.T) {
	groups := [][]map[string]string{
		{{"x": "1"}, {"x": "2"}},
		{{"y": "a"}, {"y": "b"}},
	}
	result := cartesianCtx(groups)
	assert.Len(t, result, 4)
}

// --- buildLevels ---

func TestBuildLevels_RootsOnly(t *testing.T) {
	reqs := []requests.Request{
		{URL: "/a"},
		{URL: "/b"},
	}
	levels := buildLevels(reqs)
	assert.Len(t, levels, 1)
	assert.Len(t, levels[0], 2)
}

func TestBuildLevels_OneDependency(t *testing.T) {
	reqs := []requests.Request{
		{URL: "/fabrics"},
		{
			URL: "/fabrics/{fabricName}/switches",
			DependsOn: map[string]requests.Dependency{
				"fabricName": {URL: "/fabrics", Key: "fabricName"},
			},
			Query: map[string]string{"fabricName": "{fabricName}"},
		},
	}
	levels := buildLevels(reqs)
	assert.Len(t, levels, 2)

	level0URLs := urlsFromLevel(levels[0])
	assert.Contains(t, level0URLs, "/fabrics")

	level1URLs := urlsFromLevel(levels[1])
	assert.Contains(t, level1URLs, "/fabrics/{fabricName}/switches")
}

func TestBuildLevels_Chain(t *testing.T) {
	reqs := []requests.Request{
		{URL: "/a"},
		{
			URL: "/a/{x}/b",
			DependsOn: map[string]requests.Dependency{
				"x": {URL: "/a", Key: "x"},
			},
		},
		{
			URL: "/a/{x}/b/{y}/c",
			DependsOn: map[string]requests.Dependency{
				"y": {URL: "/a/{x}/b", Key: "y"},
			},
		},
	}
	levels := buildLevels(reqs)
	assert.Len(t, levels, 3)
	assert.Contains(t, urlsFromLevel(levels[0]), "/a")
	assert.Contains(t, urlsFromLevel(levels[1]), "/a/{x}/b")
	assert.Contains(t, urlsFromLevel(levels[2]), "/a/{x}/b/{y}/c")
}

func TestBuildLevels_MultipleParents(t *testing.T) {
	// A request with two placeholders sourced from two different level-0 parents
	// should land at level 1 (max(0,0)+1).
	reqs := []requests.Request{
		{URL: "/a"},
		{URL: "/b"},
		{
			URL: "/c/{x}/{y}",
			DependsOn: map[string]requests.Dependency{
				"x": {URL: "/a", Key: "x"},
				"y": {URL: "/b", Key: "y"},
			},
		},
	}
	levels := buildLevels(reqs)
	assert.Len(t, levels, 2)
	assert.Contains(t, urlsFromLevel(levels[1]), "/c/{x}/{y}")
}

// --- expandLevel ---

func TestExpandLevel_RootRequest(t *testing.T) {
	levelReqs := []requests.Request{{URL: "/fabrics", Query: map[string]string{"limit": "10"}}}
	expanded := expandLevel(levelReqs, map[string][]parentResult{})
	assert.Len(t, expanded, 1)
	assert.Equal(t, "/fabrics", expanded[0].url)
	assert.Equal(t, "10", expanded[0].query["limit"])
}

func TestExpandLevel_DependentRequest_Array(t *testing.T) {
	levelReqs := []requests.Request{
		{
			URL: "/fabrics/{fabricName}/switches",
			DependsOn: map[string]requests.Dependency{
				"fabricName": {URL: "/fabrics", Key: "fabricName"},
			},
			Query: map[string]string{"fabricName": "{fabricName}"},
		},
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
	assert.Equal(t, "f1", expanded[0].query["fabricName"])
	assert.Equal(t, "f2", expanded[1].query["fabricName"])
}

func TestExpandLevel_DependentRequest_Object(t *testing.T) {
	levelReqs := []requests.Request{
		{
			URL: "/fabrics/{fabricName}/detail",
			DependsOn: map[string]requests.Dependency{
				"fabricName": {URL: "/fabrics", Key: "fabricName"},
			},
		},
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
	levelReqs := []requests.Request{
		{
			URL: "/fabrics/{fabricName}/switches/{serialNumber}/config",
			DependsOn: map[string]requests.Dependency{
				"serialNumber": {URL: "/fabrics/{fabricName}/switches", Key: "serialNumber"},
			},
		},
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
	levelReqs := []requests.Request{
		{
			URL: "/fabrics/{fabricName}/switches",
			DependsOn: map[string]requests.Dependency{
				"fabricName": {URL: "/fabrics", Key: "fabricName"},
			},
		},
	}
	expanded := expandLevel(levelReqs, map[string][]parentResult{})
	assert.Empty(t, expanded)
}

func TestExpandLevel_KeyRemapping(t *testing.T) {
	// Parent returns items with "name" field, but placeholder is "fabricName".
	levelReqs := []requests.Request{
		{
			URL: "/fabrics/{fabricName}/switches",
			DependsOn: map[string]requests.Dependency{
				"fabricName": {URL: "/fabrics", Key: "name"},
			},
		},
	}
	parentResults := map[string][]parentResult{
		"/fabrics": {
			{
				ctx:    map[string]string{},
				result: gjson.Parse(`[{"name":"fabric-a"},{"name":"fabric-b"}]`),
			},
		},
	}
	expanded := expandLevel(levelReqs, parentResults)
	assert.Len(t, expanded, 2)

	urls := make([]string, len(expanded))
	for i, e := range expanded {
		urls[i] = e.url
	}
	assert.Contains(t, urls, "/fabrics/fabric-a/switches")
	assert.Contains(t, urls, "/fabrics/fabric-b/switches")
}

func TestExpandLevel_MultipleParentURLs_CartesianProduct(t *testing.T) {
	// Two placeholders sourced from two different parent URLs.
	levelReqs := []requests.Request{
		{
			URL: "/report/{fabricId}/{switchId}",
			DependsOn: map[string]requests.Dependency{
				"fabricId": {URL: "/fabrics", Key: "id"},
				"switchId": {URL: "/switches", Key: "id"},
			},
		},
	}
	parentResults := map[string][]parentResult{
		"/fabrics": {
			{ctx: map[string]string{}, result: gjson.Parse(`[{"id":"f1"},{"id":"f2"}]`)},
		},
		"/switches": {
			{ctx: map[string]string{}, result: gjson.Parse(`[{"id":"s1"}]`)},
		},
	}
	expanded := expandLevel(levelReqs, parentResults)
	// 2 fabrics × 1 switch = 2 combinations
	assert.Len(t, expanded, 2)

	urls := make([]string, len(expanded))
	for i, e := range expanded {
		urls[i] = e.url
	}
	assert.Contains(t, urls, "/report/f1/s1")
	assert.Contains(t, urls, "/report/f2/s1")
}

// --- extractListResult ---

func TestExtractListResult_WrappedObject(t *testing.T) {
	// Typical NDFC response: the array is inside a named wrapper key.
	// The /api/v1/manage/fabrics endpoint uses "name" (not "fabricName") per the OpenAPI schema.
	result := gjson.Parse(`{"fabrics":[{"name":"f1"},{"name":"f2"}]}`)
	extracted := extractListResult(result, "fabrics")
	assert.True(t, extracted.IsArray())
	assert.Len(t, extracted.Array(), 2)
	assert.Equal(t, "f1", extracted.Array()[0].Get("name").String())
}

func TestExtractListResult_RootArray(t *testing.T) {
	// list_path "@this": response is already a root array (e.g. the VRF endpoint).
	result := gjson.Parse(`[{"id":1},{"id":2}]`)
	extracted := extractListResult(result, "@this")
	assert.True(t, extracted.IsArray())
	assert.Len(t, extracted.Array(), 2)
}

func TestExtractListResult_EmptyPath(t *testing.T) {
	// list_path "": single-object endpoint — return as-is.
	result := gjson.Parse(`{"key":"value"}`)
	extracted := extractListResult(result, "")
	assert.Equal(t, result.Raw, extracted.Raw)
}

func TestExtractListResult_MissingPath(t *testing.T) {
	// list_path points to a non-existent key — return the original result unchanged.
	result := gjson.Parse(`{"other":"value"}`)
	extracted := extractListResult(result, "nothere")
	assert.Equal(t, result.Raw, extracted.Raw)
}

func TestExpandLevel_VRFPipeline_WrappedFabricsResponse(t *testing.T) {
	// Full-pipeline integration test mirroring the production VRF request config.
	// The /api/v1/manage/fabrics endpoint returns {"fabrics":[{"name":"..."},...]}
	// and extractListResult unwraps it before it is stored as a parentResult.
	// expandLevel must then substitute {fabricName} from the "name" JSON field.
	levelReqs := []requests.Request{
		{
			URL:   "/appcenter/cisco/ndfc/api/v1/lan-fabric/rest/top-down/fabrics/{fabricName}/vrfs",
			DBKey: "fabrics/{fabricName}/vrfs",
			DependsOn: map[string]requests.Dependency{
				"fabricName": {URL: "/api/v1/manage/fabrics", Key: "name"},
			},
		},
	}

	rawFabricsResponse := gjson.Parse(`{"fabrics":[{"name":"DC1-FABRIC"},{"name":"DC2-FABRIC"}]}`)
	// Simulate what collectFabric does when storing level-0 results:
	// extractListResult unwraps the array before saving.
	unwrapped := extractListResult(rawFabricsResponse, "fabrics")

	parentResults := map[string][]parentResult{
		"/api/v1/manage/fabrics": {
			{ctx: map[string]string{}, result: unwrapped},
		},
	}

	expanded := expandLevel(levelReqs, parentResults)
	assert.Len(t, expanded, 2)

	urls := make([]string, len(expanded))
	keys := make([]string, len(expanded))
	for i, e := range expanded {
		urls[i] = e.url
		keys[i] = e.resolvedKey
	}
	assert.Contains(t, urls, "/appcenter/cisco/ndfc/api/v1/lan-fabric/rest/top-down/fabrics/DC1-FABRIC/vrfs")
	assert.Contains(t, urls, "/appcenter/cisco/ndfc/api/v1/lan-fabric/rest/top-down/fabrics/DC2-FABRIC/vrfs")
	assert.Contains(t, keys, "fabrics/DC1-FABRIC/vrfs")
	assert.Contains(t, keys, "fabrics/DC2-FABRIC/vrfs")
}

// helpers

func urlsFromLevel(level []requests.Request) []string {
	urls := make([]string, len(level))
	for i, r := range level {
		urls[i] = r.URL
	}
	return urls
}
