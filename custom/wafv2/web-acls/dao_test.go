package webacls

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"

	appaws "github.com/clawscli/claws/internal/aws"
)

func TestListSkipsCloudFrontScopeOutsideUSEast1(t *testing.T) {
	client := &recordingHTTPClient{}
	d := newTestWebACLDAO(client)

	_, err := d.List(appaws.WithRegionOverride(context.Background(), "us-west-2"))
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if got := client.scopeCalls("REGIONAL"); got != 1 {
		t.Fatalf("REGIONAL calls = %d, want 1; bodies=%v", got, client.bodies)
	}
	if got := client.scopeCalls("CLOUDFRONT"); got != 0 {
		t.Fatalf("CLOUDFRONT calls = %d, want 0; bodies=%v", got, client.bodies)
	}
}

func TestListIncludesCloudFrontScopeInUSEast1(t *testing.T) {
	client := &recordingHTTPClient{}
	d := newTestWebACLDAO(client)

	_, err := d.List(appaws.WithRegionOverride(context.Background(), cloudFrontWebACLRegion))
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}

	if got := client.scopeCalls("REGIONAL"); got != 1 {
		t.Fatalf("REGIONAL calls = %d, want 1; bodies=%v", got, client.bodies)
	}
	if got := client.scopeCalls("CLOUDFRONT"); got != 1 {
		t.Fatalf("CLOUDFRONT calls = %d, want 1; bodies=%v", got, client.bodies)
	}
}

func newTestWebACLDAO(httpClient aws.HTTPClient) *WebACLDAO {
	return &WebACLDAO{
		client: wafv2.NewFromConfig(aws.Config{
			Region:     cloudFrontWebACLRegion,
			HTTPClient: httpClient,
			Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
				return aws.Credentials{AccessKeyID: "test", SecretAccessKey: "test", Source: "test"}, nil
			}),
		}),
	}
}

type recordingHTTPClient struct {
	bodies []string
}

func (c *recordingHTTPClient) Do(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	c.bodies = append(c.bodies, string(body))

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/x-amz-json-1.1"}},
		Body:       io.NopCloser(bytes.NewBufferString(`{"WebACLs":[]}`)),
	}, nil
}

func (c *recordingHTTPClient) scopeCalls(scope string) int {
	needle := `"Scope":"` + scope + `"`
	count := 0
	for _, body := range c.bodies {
		if strings.Contains(body, needle) {
			count++
		}
	}
	return count
}
