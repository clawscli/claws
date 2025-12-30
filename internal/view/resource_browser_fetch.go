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

const multiRegionFetchTimeout = 30 * time.Second

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

type multiRegionFetchResult struct {
	resources  []dao.Resource
	errors     []string
	pageTokens map[string]string
}

func (r *ResourceBrowser) fetchMultiRegionResources(regions []string, existingTokens map[string]string) multiRegionFetchResult {
	type regionResult struct {
		region    string
		resources []dao.Resource
		nextToken string
		err       error
	}

	ctx, cancel := context.WithTimeout(r.ctx, multiRegionFetchTimeout)
	defer cancel()

	results := make(chan regionResult, len(regions))
	var wg sync.WaitGroup

	for _, region := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()

			regionCtx := aws.WithRegionOverride(ctx, region)
			d, err := r.registry.GetDAO(regionCtx, r.service, r.resourceType)
			if err != nil {
				results <- regionResult{region: region, err: err}
				return
			}

			var listResult listResourcesResult
			if pagDAO, ok := d.(dao.PaginatedDAO); ok {
				token := ""
				if existingTokens != nil {
					token = existingTokens[region]
				}
				listCtx := regionCtx
				if r.fieldFilter != "" && r.fieldFilterValue != "" {
					listCtx = dao.WithFilter(regionCtx, r.fieldFilter, r.fieldFilterValue)
				}
				resources, nextToken, err := pagDAO.ListPage(listCtx, r.pageSize, token)
				listResult = listResourcesResult{resources: resources, nextToken: nextToken, err: err}
			} else {
				listResult = r.listResourcesWithContext(regionCtx, d)
			}

			if listResult.err != nil {
				results <- regionResult{region: region, err: listResult.err}
				return
			}

			results <- regionResult{region: region, resources: listResult.resources, nextToken: listResult.nextToken}
		}(region)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	resultsByRegion := make(map[string]regionResult)
	for result := range results {
		resultsByRegion[result.region] = result
	}

	var allResources []dao.Resource
	var errors []string
	pageTokens := make(map[string]string)
	for _, region := range regions {
		result, ok := resultsByRegion[region]
		if !ok {
			continue
		}
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.region, result.err))
			log.Warn("failed to fetch from region", "region", result.region, "error", result.err)
		} else {
			allResources = append(allResources, result.resources...)
			if result.nextToken != "" {
				pageTokens[result.region] = result.nextToken
			}
		}
	}

	return multiRegionFetchResult{resources: allResources, errors: errors, pageTokens: pageTokens}
}

func (r *ResourceBrowser) loadResources() tea.Msg {
	start := time.Now()
	regions := config.Global().Regions()
	isMultiRegion := len(regions) > 1

	log.Debug("loading resources", "service", r.service, "resourceType", r.resourceType,
		"regions", regions, "multiRegion", isMultiRegion)

	renderer, err := r.registry.GetRenderer(r.service, r.resourceType)
	if err != nil {
		log.Error("failed to get renderer", "service", r.service, "resourceType", r.resourceType, "error", err)
		return resourcesErrorMsg{err: err}
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
	regions := config.Global().Regions()
	isMultiRegion := len(regions) > 1

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
	dao            dao.DAO
	renderer       render.Renderer
	resources      []dao.Resource
	nextToken      string
	nextPageTokens map[string]string
	hasMorePages   bool
	partialErrors  []string
}

type nextPageLoadedMsg struct {
	resources      []dao.Resource
	nextToken      string
	nextPageTokens map[string]string
	hasMorePages   bool
}

type resourcesErrorMsg struct {
	err error
}

func (r *ResourceBrowser) shouldLoadNextPage() bool {
	if !r.hasMorePages || r.isLoadingMore || r.loading {
		return false
	}
	if r.nextPageToken == "" && len(r.nextPageTokens) == 0 {
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
