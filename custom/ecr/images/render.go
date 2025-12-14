package images

import (
	"fmt"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// ImageRenderer renders ECR images
// Ensure ImageRenderer implements render.Navigator
var _ render.Navigator = (*ImageRenderer)(nil)

type ImageRenderer struct {
	render.BaseRenderer
}

// NewImageRenderer creates a new ImageRenderer
func NewImageRenderer() *ImageRenderer {
	return &ImageRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "ecr",
			Resource: "images",
			Cols: []render.Column{
				{Name: "TAG", Width: 25, Getter: getTag},
				{Name: "DIGEST", Width: 20, Getter: getDigest},
				{Name: "SIZE", Width: 12, Getter: getSize},
				{Name: "SCAN", Width: 12, Getter: getScanStatus},
				{Name: "PUSHED", Width: 20, Getter: getPushed},
			},
		},
	}
}

func getTag(r dao.Resource) string {
	if img, ok := r.(*ImageResource); ok {
		return img.TagsFormatted()
	}
	return ""
}

func getDigest(r dao.Resource) string {
	if img, ok := r.(*ImageResource); ok {
		digest := img.ImageDigest()
		// Shorten sha256:abc123... to sha256:abc123
		if strings.HasPrefix(digest, "sha256:") && len(digest) > 19 {
			return digest[:19] + "..."
		}
		return digest
	}
	return ""
}

func getSize(r dao.Resource) string {
	if img, ok := r.(*ImageResource); ok {
		return img.ImageSizeFormatted()
	}
	return "-"
}

func getScanStatus(r dao.Resource) string {
	if img, ok := r.(*ImageResource); ok {
		status := img.ScanStatus()
		if status == "" {
			return "-"
		}
		findings := img.ScanFindingsCount()
		if findings > 0 {
			return fmt.Sprintf("%s (%d)", status, findings)
		}
		return status
	}
	return "-"
}

func getPushed(r dao.Resource) string {
	if img, ok := r.(*ImageResource); ok {
		return img.PushedAt()
	}
	return "-"
}

// RenderDetail renders detailed image information
func (r *ImageRenderer) RenderDetail(resource dao.Resource) string {
	img, ok := resource.(*ImageResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("ECR Image", img.TagsFormatted())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Repository", img.RepositoryName)
	d.Field("Digest", img.ImageDigest())

	// Tags
	if tags := img.ImageTags(); len(tags) > 0 {
		d.Field("Tags", strings.Join(tags, ", "))
	} else {
		d.Field("Tags", "<untagged>")
	}

	d.Field("Size", img.ImageSizeFormatted())

	// Media Types
	if mediaType := img.ArtifactMediaType(); mediaType != "" {
		d.Field("Artifact Media Type", mediaType)
	}
	if manifestType := img.ImageManifestMediaType(); manifestType != "" {
		d.Field("Manifest Media Type", manifestType)
	}

	// Scan Status
	d.Section("Scan Status")
	if status := img.ScanStatus(); status != "" {
		d.Field("Status", status)
		d.Field("Findings Count", fmt.Sprintf("%d", img.ScanFindingsCount()))

		// Show severity breakdown if available
		if img.Image.ImageScanFindingsSummary != nil && img.Image.ImageScanFindingsSummary.FindingSeverityCounts != nil {
			for severity, count := range img.Image.ImageScanFindingsSummary.FindingSeverityCounts {
				d.Field(string(severity), fmt.Sprintf("%d", count))
			}
		}
	} else {
		d.Field("Status", "Not scanned")
	}

	// Timestamps
	d.Section("Timestamps")
	if pushed := img.PushedAt(); pushed != "" {
		d.Field("Pushed At", pushed)
	}
	if img.Image.LastRecordedPullTime != nil {
		d.Field("Last Pulled", img.Image.LastRecordedPullTime.Format("2006-01-02 15:04:05"))
	}

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *ImageRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	img, ok := resource.(*ImageResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Repository", Value: img.RepositoryName},
		{Label: "Tag", Value: img.TagsFormatted()},
		{Label: "Digest", Value: img.ImageDigest()},
		{Label: "Size", Value: img.ImageSizeFormatted()},
	}

	if status := img.ScanStatus(); status != "" {
		fields = append(fields, render.SummaryField{Label: "Scan", Value: status})
	}

	if pushed := img.PushedAt(); pushed != "" {
		fields = append(fields, render.SummaryField{Label: "Pushed", Value: pushed})
	}

	return fields
}

// Navigations returns navigation shortcuts
func (r *ImageRenderer) Navigations(resource dao.Resource) []render.Navigation {
	return nil
}
