package view

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/config"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/metrics"
	"github.com/clawscli/claws/internal/render"
)

type metricsLoadedMsg struct {
	data         *metrics.MetricData
	err          error
	resourceType string
}

func (r *ResourceBrowser) loadMetricsCmd() tea.Cmd {
	spec := r.getMetricSpec()
	if spec == nil {
		return nil
	}

	type resourceInfo struct {
		fullID      string
		unwrappedID string
		region      string
	}
	infos := make([]resourceInfo, len(r.resources))
	for i, res := range r.resources {
		infos[i] = resourceInfo{
			fullID:      res.GetID(),
			unwrappedID: dao.UnwrapResource(res).GetID(),
			region:      dao.GetResourceRegion(res),
		}
	}
	resourceType := r.resourceType
	baseCtx := r.ctx

	return func() tea.Msg {
		if baseCtx.Err() != nil {
			return nil
		}

		ctx, cancel := context.WithTimeout(baseCtx, config.File().MetricsLoadTimeout())
		defer cancel()

		byRegion := make(map[string][]resourceInfo)
		for _, info := range infos {
			byRegion[info.region] = append(byRegion[info.region], info)
		}

		data := metrics.NewMetricData(spec)

		for region, regionInfos := range byRegion {
			regionCtx := ctx
			if region != "" {
				regionCtx = aws.WithRegionOverride(ctx, region)
			}

			fetcher, err := metrics.NewFetcher(regionCtx)
			if err != nil {
				continue
			}

			unwrappedIDs := make([]string, len(regionInfos))
			for i, info := range regionInfos {
				unwrappedIDs[i] = info.unwrappedID
			}

			regionData, err := fetcher.Fetch(regionCtx, unwrappedIDs, spec)
			if err != nil {
				continue
			}

			for i, info := range regionInfos {
				if result := regionData.Get(unwrappedIDs[i]); result != nil {
					result.ResourceID = info.fullID
					data.Results[info.fullID] = result
				}
			}
		}

		return metricsLoadedMsg{data: data, err: nil, resourceType: resourceType}
	}
}

func (r *ResourceBrowser) getMetricSpec() *render.MetricSpec {
	if r.renderer == nil {
		return nil
	}
	if provider, ok := r.renderer.(render.MetricSpecProvider); ok {
		return provider.MetricSpec()
	}
	return nil
}
