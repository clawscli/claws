package view

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/log"
	"github.com/clawscli/claws/internal/render"
)

type listResourcesResult struct {
	resources []dao.Resource
	nextToken string
	err       error
}

func (r *ResourceBrowser) listResourcesWithContext(ctx context.Context, d dao.DAO) listResourcesResult {
	listCtx := ctx
	if r.fieldFilter != "" && r.fieldFilterValue != "" {
		listCtx = dao.WithFilter(ctx, r.fieldFilter, r.fieldFilterValue)
	}

	var resources []dao.Resource
	var nextToken string
	var err error
	if pagDAO, ok := d.(dao.PaginatedDAO); ok {
		resources, nextToken, err = pagDAO.ListPage(listCtx, r.pageSize, "")
	} else {
		resources, err = d.List(listCtx)
	}
	return listResourcesResult{resources: resources, nextToken: nextToken, err: err}
}

func (r *ResourceBrowser) listResources(d dao.DAO) listResourcesResult {
	return r.listResourcesWithContext(r.ctx, d)
}

type profileRegionKey struct {
	Profile string
	Region  string
}

type parallelFetchItem[K comparable] struct {
	key       K
	resources []dao.Resource
	nextToken string
	err       error
}

type parallelFetchResult[K comparable] struct {
	resources  []dao.Resource
	errors     []string
	pageTokens map[K]string
}

func fetchParallel[K comparable](
	ctx context.Context,
	keys []K,
	fetch func(context.Context, K) ([]dao.Resource, string, error),
	formatError func(K, error) string,
) parallelFetchResult[K] {
	ctx, cancel := context.WithTimeout(ctx, config.File().MultiRegionFetchTimeout())
	defer cancel()

	results := make(chan parallelFetchItem[K], len(keys))
	sem := make(chan struct{}, config.File().MaxConcurrentFetches())
	var wg sync.WaitGroup

	for _, key := range keys {
		wg.Add(1)
		go func(k K) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore
			resources, nextToken, err := fetch(ctx, k)
			results <- parallelFetchItem[K]{key: k, resources: resources, nextToken: nextToken, err: err}
		}(key)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	resultsByKey := make(map[K]parallelFetchItem[K])
	for result := range results {
		resultsByKey[result.key] = result
	}

	var allResources []dao.Resource
	var errors []string
	pageTokens := make(map[K]string)
	for _, key := range keys {
		result, ok := resultsByKey[key]
		if !ok {
			continue
		}
		if result.err != nil {
			errors = append(errors, formatError(key, result.err))
		} else {
			allResources = append(allResources, result.resources...)
			if result.nextToken != "" {
				pageTokens[key] = result.nextToken
			}
		}
	}

	return parallelFetchResult[K]{resources: allResources, errors: errors, pageTokens: pageTokens}
}

func (r *ResourceBrowser) fetchMultiProfileResources(profiles []config.ProfileSelection, regions []string, existingTokens map[profileRegionKey]string) parallelFetchResult[profileRegionKey] {
	profileMap := make(map[string]config.ProfileSelection, len(profiles))
	for _, sel := range profiles {
		profileMap[sel.ID()] = sel
	}

	var keys []profileRegionKey
	for _, sel := range profiles {
		for _, region := range regions {
			keys = append(keys, profileRegionKey{Profile: sel.ID(), Region: region})
		}
	}

	fetch := func(ctx context.Context, key profileRegionKey) ([]dao.Resource, string, error) {
		sel := profileMap[key.Profile]
		fetchCtx := aws.WithSelectionOverride(ctx, sel)
		fetchCtx = aws.WithRegionOverride(fetchCtx, key.Region)

		accountID := config.Global().GetAccountIDForProfile(key.Profile)
		if accountID == "" {
			if id := aws.FetchAccountIDForContext(fetchCtx); id != "" {
				config.Global().SetAccountIDForProfile(key.Profile, id)
				accountID = id
			}
		}

		d, err := r.registry.GetDAO(fetchCtx, r.service, r.resourceType)
		if err != nil {
			return nil, "", err
		}

		listResult := r.fetchWithDAO(fetchCtx, d, existingTokens[key])
		if listResult.err != nil {
			return nil, "", listResult.err
		}

		wrapped := make([]dao.Resource, len(listResult.resources))
		for i, res := range listResult.resources {
			wrapped[i] = dao.WrapWithProfile(dao.UnwrapResource(res), key.Profile, accountID, key.Region)
		}
		return wrapped, listResult.nextToken, nil
	}

	formatError := func(key profileRegionKey, err error) string {
		log.Debug("failed to fetch", "profile", key.Profile, "region", key.Region, "error", err)
		return fmt.Sprintf("%s/%s: %v", key.Profile, key.Region, err)
	}

	return fetchParallel(r.ctx, keys, fetch, formatError)
}

func (r *ResourceBrowser) fetchMultiRegionResources(regions []string, existingTokens map[string]string) parallelFetchResult[string] {
	fetch := func(ctx context.Context, region string) ([]dao.Resource, string, error) {
		regionCtx := aws.WithRegionOverride(ctx, region)
		d, err := r.registry.GetDAO(regionCtx, r.service, r.resourceType)
		if err != nil {
			return nil, "", err
		}

		token := ""
		if existingTokens != nil {
			token = existingTokens[region]
		}
		listResult := r.fetchWithDAO(regionCtx, d, token)
		if listResult.err != nil {
			return nil, "", listResult.err
		}

		wrapped := make([]dao.Resource, len(listResult.resources))
		for i, res := range listResult.resources {
			wrapped[i] = dao.WrapWithRegion(dao.UnwrapResource(res), region)
		}
		return wrapped, listResult.nextToken, nil
	}

	formatError := func(region string, err error) string {
		log.Debug("failed to fetch from region", "region", region, "error", err)
		return fmt.Sprintf("%s: %v", region, err)
	}

	return fetchParallel(r.ctx, regions, fetch, formatError)
}

func (r *ResourceBrowser) fetchWithDAO(ctx context.Context, d dao.DAO, token string) listResourcesResult {
	if pagDAO, ok := d.(dao.PaginatedDAO); ok {
		listCtx := ctx
		if r.fieldFilter != "" && r.fieldFilterValue != "" {
			listCtx = dao.WithFilter(ctx, r.fieldFilter, r.fieldFilterValue)
		}
		resources, nextToken, err := pagDAO.ListPage(listCtx, r.pageSize, token)
		return listResourcesResult{resources: resources, nextToken: nextToken, err: err}
	}
	return r.listResourcesWithContext(ctx, d)
}

func (r *ResourceBrowser) loadResources() tea.Msg {
	start := time.Now()
	profiles := config.Global().Selections()
	regions := config.Global().Regions()
	isMultiProfile := len(profiles) > 1
	isMultiRegion := len(regions) > 1

	log.Debug("loading resources", "service", r.service, "resourceType", r.resourceType,
		"profiles", len(profiles), "regions", regions, "multiProfile", isMultiProfile, "multiRegion", isMultiRegion)

	renderer, err := r.registry.GetRenderer(r.service, r.resourceType)
	if err != nil {
		log.Error("failed to get renderer", "service", r.service, "resourceType", r.resourceType, "error", err)
		return resourcesErrorMsg{err: err}
	}

	if isMultiProfile {
		fetchResult := r.fetchMultiProfileResources(profiles, regions, nil)
		if len(fetchResult.resources) == 0 && len(fetchResult.errors) > 0 {
			return resourcesErrorMsg{err: fmt.Errorf("all profile/region pairs failed: %s", strings.Join(fetchResult.errors, "; "))}
		}

		log.Debug("multi-profile resources loaded", "count", len(fetchResult.resources),
			"profiles", len(profiles), "regions", len(regions), "errors", len(fetchResult.errors), "duration", time.Since(start))

		return resourcesLoadedMsg{
			dao:                 nil,
			renderer:            renderer,
			resources:           fetchResult.resources,
			nextMultiPageTokens: fetchResult.pageTokens,
			hasMorePages:        len(fetchResult.pageTokens) > 0,
			partialErrors:       fetchResult.errors,
		}
	}

	if !isMultiRegion {
		d, err := r.registry.GetDAO(r.ctx, r.service, r.resourceType)
		if err != nil {
			log.Error("failed to get DAO", "service", r.service, "resourceType", r.resourceType, "error", err)
			return resourcesErrorMsg{err: err}
		}

		result := r.listResources(d)
		if result.err != nil {
			log.Error("failed to list resources", "error", result.err, "duration", time.Since(start))
			return resourcesErrorMsg{err: result.err}
		}
		log.Debug("resources loaded", "count", len(result.resources), "duration", time.Since(start))

		return resourcesLoadedMsg{
			dao:          d,
			renderer:     renderer,
			resources:    result.resources,
			nextToken:    result.nextToken,
			hasMorePages: result.nextToken != "",
		}
	}

	fetchResult := r.fetchMultiRegionResources(regions, nil)
	if len(fetchResult.resources) == 0 && len(fetchResult.errors) > 0 {
		return resourcesErrorMsg{err: fmt.Errorf("all regions failed: %s", strings.Join(fetchResult.errors, "; "))}
	}

	log.Debug("multi-region resources loaded", "count", len(fetchResult.resources),
		"regions", len(regions), "errors", len(fetchResult.errors), "duration", time.Since(start))

	return resourcesLoadedMsg{
		dao:            nil,
		renderer:       renderer,
		resources:      fetchResult.resources,
		nextPageTokens: fetchResult.pageTokens,
		hasMorePages:   len(fetchResult.pageTokens) > 0,
		partialErrors:  fetchResult.errors,
	}
}

func (r *ResourceBrowser) reloadResources() tea.Msg {
	profiles := config.Global().Selections()
	regions := config.Global().Regions()
	isMultiProfile := len(profiles) > 1
	isMultiRegion := len(regions) > 1

	if isMultiProfile {
		fetchResult := r.fetchMultiProfileResources(profiles, regions, nil)
		if len(fetchResult.resources) == 0 && len(fetchResult.errors) > 0 {
			return resourcesErrorMsg{err: fmt.Errorf("all profile/region pairs failed: %s", strings.Join(fetchResult.errors, "; "))}
		}

		return resourcesLoadedMsg{
			dao:                 nil,
			renderer:            r.renderer,
			resources:           fetchResult.resources,
			nextMultiPageTokens: fetchResult.pageTokens,
			hasMorePages:        len(fetchResult.pageTokens) > 0,
			partialErrors:       fetchResult.errors,
		}
	}

	if !isMultiRegion {
		d := r.dao
		if d == nil {
			var err error
			d, err = r.registry.GetDAO(r.ctx, r.service, r.resourceType)
			if err != nil {
				return resourcesErrorMsg{err: err}
			}
		}

		result := r.listResources(d)
		if result.err != nil {
			return resourcesErrorMsg{err: result.err}
		}

		return resourcesLoadedMsg{
			dao:          d,
			renderer:     r.renderer,
			resources:    result.resources,
			nextToken:    result.nextToken,
			hasMorePages: result.nextToken != "",
		}
	}

	fetchResult := r.fetchMultiRegionResources(regions, nil)
	if len(fetchResult.resources) == 0 && len(fetchResult.errors) > 0 {
		return resourcesErrorMsg{err: fmt.Errorf("all regions failed: %s", strings.Join(fetchResult.errors, "; "))}
	}

	return resourcesLoadedMsg{
		dao:            nil,
		renderer:       r.renderer,
		resources:      fetchResult.resources,
		nextPageTokens: fetchResult.pageTokens,
		hasMorePages:   len(fetchResult.pageTokens) > 0,
		partialErrors:  fetchResult.errors,
	}
}

type resourcesLoadedMsg struct {
	dao                 dao.DAO
	renderer            render.Renderer
	resources           []dao.Resource
	nextToken           string
	nextPageTokens      map[string]string
	nextMultiPageTokens map[profileRegionKey]string
	hasMorePages        bool
	partialErrors       []string
}

type nextPageLoadedMsg struct {
	resources           []dao.Resource
	nextToken           string
	nextPageTokens      map[string]string
	nextMultiPageTokens map[profileRegionKey]string
	hasMorePages        bool
}

type resourcesErrorMsg struct {
	err error
}

func (r *ResourceBrowser) shouldLoadNextPage() bool {
	if !r.hasMorePages || r.isLoadingMore || r.loading {
		return false
	}
	if r.nextPageToken == "" && len(r.nextPageTokens) == 0 && len(r.nextMultiPageTokens) == 0 {
		return false
	}
	if r.filterText != "" && len(r.filtered) < 10 {
		return false
	}
	if len(r.filtered) == 0 {
		return false
	}
	buffer := 10
	return r.table.Cursor() >= len(r.filtered)-buffer
}

func (r *ResourceBrowser) loadNextPage() tea.Msg {
	if len(r.nextMultiPageTokens) > 0 {
		return r.loadNextPageMultiProfile()
	}

	if len(r.nextPageTokens) > 0 {
		return r.loadNextPageMultiRegion()
	}

	if r.nextPageToken == "" {
		return nil
	}

	pagDAO, ok := r.dao.(dao.PaginatedDAO)
	if !ok {
		return nil
	}

	start := time.Now()
	log.Debug("loading next page", "service", r.service, "resourceType", r.resourceType, "token", r.nextPageToken[:min(logTokenMaxLen, len(r.nextPageToken))])

	listCtx := r.ctx
	if r.fieldFilter != "" && r.fieldFilterValue != "" {
		listCtx = dao.WithFilter(r.ctx, r.fieldFilter, r.fieldFilterValue)
	}

	resources, nextToken, err := pagDAO.ListPage(listCtx, r.pageSize, r.nextPageToken)
	if err != nil {
		log.Error("failed to load next page", "error", err, "duration", time.Since(start))
		return resourcesErrorMsg{err: err}
	}

	log.Debug("next page loaded", "count", len(resources), "hasMore", nextToken != "", "duration", time.Since(start))

	return nextPageLoadedMsg{
		resources:    resources,
		nextToken:    nextToken,
		hasMorePages: nextToken != "",
	}
}

func (r *ResourceBrowser) loadNextPageMultiRegion() tea.Msg {
	configRegions := config.Global().Regions()
	regions := make([]string, 0, len(r.nextPageTokens))
	for _, region := range configRegions {
		if _, ok := r.nextPageTokens[region]; ok {
			regions = append(regions, region)
		}
	}

	start := time.Now()
	log.Debug("loading next page multi-region", "service", r.service, "resourceType", r.resourceType, "regions", len(regions))

	fetchResult := r.fetchMultiRegionResources(regions, r.nextPageTokens)

	log.Debug("next page multi-region loaded", "count", len(fetchResult.resources), "hasMore", len(fetchResult.pageTokens) > 0, "duration", time.Since(start))

	return nextPageLoadedMsg{
		resources:      fetchResult.resources,
		nextPageTokens: fetchResult.pageTokens,
		hasMorePages:   len(fetchResult.pageTokens) > 0,
	}
}

func (r *ResourceBrowser) loadNextPageMultiProfile() tea.Msg {
	profiles := config.Global().Selections()
	regions := config.Global().Regions()

	tokensToFetch := make(map[profileRegionKey]string)
	for key, token := range r.nextMultiPageTokens {
		tokensToFetch[key] = token
	}

	start := time.Now()
	log.Debug("loading next page multi-profile", "service", r.service, "resourceType", r.resourceType, "pairs", len(tokensToFetch))

	fetchResult := r.fetchMultiProfileResources(profiles, regions, tokensToFetch)

	log.Debug("next page multi-profile loaded", "count", len(fetchResult.resources), "hasMore", len(fetchResult.pageTokens) > 0, "duration", time.Since(start))

	return nextPageLoadedMsg{
		resources:           fetchResult.resources,
		nextMultiPageTokens: fetchResult.pageTokens,
		hasMorePages:        len(fetchResult.pageTokens) > 0,
	}
}
