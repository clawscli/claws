package render

import (
	"testing"
	"time"

	"github.com/clawscli/claws/internal/dao"
)

func TestFormatAge(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{"zero time", time.Time{}, ""},
		{"30 seconds ago", now.Add(-30 * time.Second), "30s"},
		{"5 minutes ago", now.Add(-5 * time.Minute), "5m"},
		{"2 hours ago", now.Add(-2 * time.Hour), "2h"},
		{"3 days ago", now.Add(-3 * 24 * time.Hour), "3d"},
		{"45 days ago", now.Add(-45 * 24 * time.Hour), "1mo"},
		{"400 days ago", now.Add(-400 * 24 * time.Hour), "1y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatAge(tt.time)
			if got != tt.want {
				t.Errorf("FormatAge() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1024 * 1024, "1.0 MiB"},
		{1024 * 1024 * 1024, "1.0 GiB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TiB"},
		{1536 * 1024 * 1024, "1.5 GiB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestFormatTags(t *testing.T) {
	tests := []struct {
		name   string
		tags   map[string]string
		maxLen int
		want   string
	}{
		{
			name:   "empty tags",
			tags:   map[string]string{},
			maxLen: 50,
			want:   "",
		},
		{
			name:   "nil tags",
			tags:   nil,
			maxLen: 50,
			want:   "",
		},
		{
			name:   "single tag",
			tags:   map[string]string{"Env": "prod"},
			maxLen: 50,
			want:   "Env=prod",
		},
		{
			name:   "priority tag shown first",
			tags:   map[string]string{"Foo": "bar", "Environment": "dev"},
			maxLen: 50,
			want:   "Environment=dev, Foo=bar",
		},
		{
			name:   "Name tag excluded",
			tags:   map[string]string{"Name": "my-instance", "Env": "prod"},
			maxLen: 50,
			want:   "Env=prod",
		},
		{
			name:   "truncated with ellipsis",
			tags:   map[string]string{"VeryLongTagName": "VeryLongTagValue"},
			maxLen: 15,
			want:   "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTags(tt.tags, tt.maxLen)
			if got != tt.want {
				t.Errorf("FormatTags() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStateColorer(t *testing.T) {
	colorer := StateColorer()

	tests := []struct {
		value     string
		expectNil bool // true if style should be empty (no foreground)
	}{
		{"running", false},
		{"available", false},
		{"stopped", false},
		{"terminated", false},
		{"pending", false},
		{"in-use", false},
		{"unknown-state", true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			style := colorer(tt.value)
			// Check that style was returned (we can't easily check the color value)
			_ = style.Render(tt.value)
		})
	}
}

// mockResource implements dao.Resource for testing
type mockResource struct {
	id   string
	name string
	arn  string
	tags map[string]string
	data any
}

func (r *mockResource) GetID() string              { return r.id }
func (r *mockResource) GetName() string            { return r.name }
func (r *mockResource) GetARN() string             { return r.arn }
func (r *mockResource) GetTags() map[string]string { return r.tags }
func (r *mockResource) Raw() any                   { return r.data }

func TestBaseRenderer(t *testing.T) {
	renderer := &BaseRenderer{
		Service:  "ec2",
		Resource: "instances",
		Cols: []Column{
			{Name: "ID", Width: 20, Getter: func(r dao.Resource) string { return r.GetID() }},
			{Name: "NAME", Width: 30, Getter: func(r dao.Resource) string { return r.GetName() }},
		},
	}

	t.Run("ServiceName", func(t *testing.T) {
		if got := renderer.ServiceName(); got != "ec2" {
			t.Errorf("ServiceName() = %q, want %q", got, "ec2")
		}
	})

	t.Run("ResourceType", func(t *testing.T) {
		if got := renderer.ResourceType(); got != "instances" {
			t.Errorf("ResourceType() = %q, want %q", got, "instances")
		}
	})

	t.Run("Columns", func(t *testing.T) {
		cols := renderer.Columns()
		if len(cols) != 2 {
			t.Errorf("Columns() returned %d columns, want 2", len(cols))
		}
	})

	t.Run("RenderRow", func(t *testing.T) {
		res := &mockResource{id: "i-12345", name: "test-instance"}
		row := renderer.RenderRow(res, renderer.Cols)

		if len(row) != 2 {
			t.Fatalf("RenderRow() returned %d values, want 2", len(row))
		}
		if row[0] != "i-12345" {
			t.Errorf("RenderRow()[0] = %q, want %q", row[0], "i-12345")
		}
		if row[1] != "test-instance" {
			t.Errorf("RenderRow()[1] = %q, want %q", row[1], "test-instance")
		}
	})

	t.Run("RenderRow with nil getter", func(t *testing.T) {
		r := &BaseRenderer{
			Cols: []Column{{Name: "TEST", Width: 10, Getter: nil}},
		}
		res := &mockResource{id: "test"}
		row := r.RenderRow(res, r.Cols)

		if row[0] != "" {
			t.Errorf("RenderRow() with nil getter = %q, want empty", row[0])
		}
	})

	t.Run("RenderSummary with name", func(t *testing.T) {
		res := &mockResource{id: "i-12345", name: "test-instance"}
		fields := renderer.RenderSummary(res)

		if len(fields) != 2 {
			t.Fatalf("RenderSummary() returned %d fields, want 2", len(fields))
		}
		if fields[0].Label != "ID" || fields[0].Value != "i-12345" {
			t.Errorf("RenderSummary()[0] = {%s, %s}, want {ID, i-12345}", fields[0].Label, fields[0].Value)
		}
		if fields[1].Label != "Name" || fields[1].Value != "test-instance" {
			t.Errorf("RenderSummary()[1] = {%s, %s}, want {Name, test-instance}", fields[1].Label, fields[1].Value)
		}
	})

	t.Run("RenderSummary without name", func(t *testing.T) {
		res := &mockResource{id: "i-12345", name: ""}
		fields := renderer.RenderSummary(res)

		if len(fields) != 1 {
			t.Fatalf("RenderSummary() returned %d fields, want 1", len(fields))
		}
	})

	t.Run("RenderSummary with same name as ID", func(t *testing.T) {
		res := &mockResource{id: "i-12345", name: "i-12345"}
		fields := renderer.RenderSummary(res)

		// Should not add name if it's same as ID
		if len(fields) != 1 {
			t.Fatalf("RenderSummary() returned %d fields, want 1", len(fields))
		}
	})

	t.Run("RenderDetail returns empty", func(t *testing.T) {
		res := &mockResource{id: "test"}
		detail := renderer.RenderDetail(res)
		if detail != "" {
			t.Errorf("RenderDetail() = %q, want empty", detail)
		}
	})
}

func TestTagsColumn(t *testing.T) {
	col := TagsColumn(40, 5)

	if col.Name != "TAGS" {
		t.Errorf("Name = %q, want %q", col.Name, "TAGS")
	}
	if col.Width != 40 {
		t.Errorf("Width = %d, want %d", col.Width, 40)
	}
	if col.Priority != 5 {
		t.Errorf("Priority = %d, want %d", col.Priority, 5)
	}

	// Test getter
	res := &mockResource{tags: map[string]string{"Env": "prod", "Team": "platform"}}
	got := col.Getter(res)
	// Should contain both tags (order may vary due to priority)
	if got == "" {
		t.Error("Getter() returned empty string")
	}
}

func TestStyleHelpers(t *testing.T) {
	// These just verify the functions don't panic
	_ = SuccessStyle().Render("test")
	_ = WarningStyle().Render("test")
	_ = DangerStyle().Render("test")
	_ = DimStyle().Render("test")
	_ = DefaultStyle().Render("test")
}
