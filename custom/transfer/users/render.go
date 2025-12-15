package users

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

// UserRenderer renders Transfer Family users.
type UserRenderer struct {
	render.BaseRenderer
}

// NewUserRenderer creates a new UserRenderer.
func NewUserRenderer() render.Renderer {
	return &UserRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "transfer",
			Resource: "users",
			Cols: []render.Column{
				{Name: "USERNAME", Width: 30, Getter: func(r dao.Resource) string { return r.GetID() }},
				{Name: "HOME DIRECTORY", Width: 40, Getter: getHomeDirectory},
				{Name: "TYPE", Width: 10, Getter: getHomeDirectoryType},
				{Name: "SSH KEYS", Width: 10, Getter: getSshKeyCount},
			},
		},
	}
}

func getHomeDirectory(r dao.Resource) string {
	user, ok := r.(*UserResource)
	if !ok {
		return ""
	}
	dir := user.HomeDirectory()
	if len(dir) > 37 {
		return dir[:37] + "..."
	}
	return dir
}

func getHomeDirectoryType(r dao.Resource) string {
	user, ok := r.(*UserResource)
	if !ok {
		return ""
	}
	return user.HomeDirectoryType()
}

func getSshKeyCount(r dao.Resource) string {
	user, ok := r.(*UserResource)
	if !ok {
		return ""
	}
	count := user.SshPublicKeyCount()
	if count == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", count)
}

// RenderDetail renders the detail view for a Transfer Family user.
func (r *UserRenderer) RenderDetail(resource dao.Resource) string {
	user, ok := resource.(*UserResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("Transfer Family User", user.UserName())

	// Basic Info
	d.Section("Basic Information")
	d.Field("Username", user.UserName())
	d.Field("ARN", user.GetARN())
	d.Field("Server ID", user.ServerId)

	// Home Directory
	d.Section("Home Directory")
	d.Field("Type", user.HomeDirectoryType())
	if dir := user.HomeDirectory(); dir != "" {
		d.Field("Path", dir)
	}

	// Home Directory Mappings
	if mappings := user.HomeDirectoryMappings(); len(mappings) > 0 {
		d.Section("Directory Mappings")
		for _, mapping := range mappings {
			if mapping.Entry != nil && mapping.Target != nil {
				d.Field(*mapping.Entry, *mapping.Target)
			}
		}
	}

	// IAM Role
	if role := user.Role(); role != "" {
		d.Section("Access")
		d.Field("IAM Role ARN", role)
	}

	// POSIX Profile
	if posix := user.PosixProfile(); posix != nil {
		d.Section("POSIX Profile")
		if posix.Uid != nil {
			d.Field("UID", fmt.Sprintf("%d", *posix.Uid))
		}
		if posix.Gid != nil {
			d.Field("GID", fmt.Sprintf("%d", *posix.Gid))
		}
		if len(posix.SecondaryGids) > 0 {
			gids := ""
			for i, gid := range posix.SecondaryGids {
				if i > 0 {
					gids += ", "
				}
				gids += fmt.Sprintf("%d", gid)
			}
			d.Field("Secondary GIDs", gids)
		}
	}

	// SSH Keys
	d.Section("SSH Public Keys")
	d.Field("Count", fmt.Sprintf("%d", user.SshPublicKeyCount()))
	if keys := user.SshPublicKeys(); len(keys) > 0 {
		for i, key := range keys {
			keyLabel := fmt.Sprintf("Key %d", i+1)
			if key.SshPublicKeyId != nil {
				d.Field(keyLabel+" ID", *key.SshPublicKeyId)
			}
			if key.DateImported != nil {
				d.Field(keyLabel+" Imported", key.DateImported.Format("2006-01-02 15:04:05"))
			}
		}
	}

	// Tags
	if tags := user.Tags(); len(tags) > 0 {
		d.Section("Tags")
		for _, tag := range tags {
			if tag.Key != nil && tag.Value != nil {
				d.Field(*tag.Key, *tag.Value)
			}
		}
	}

	// Session Policy (at bottom for readability)
	if policy := user.Policy(); policy != "" {
		d.Section("Session Policy")
		d.Line(prettyJSON(policy))
	}

	return d.String()
}

// prettyJSON formats JSON string with indentation
func prettyJSON(s string) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(s), "", "  "); err != nil {
		return s
	}
	return buf.String()
}

// RenderSummary renders summary fields for a Transfer Family user.
func (r *UserRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	user, ok := resource.(*UserResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Username", Value: user.UserName()},
		{Label: "ARN", Value: user.GetARN()},
		{Label: "Server ID", Value: user.ServerId},
	}

	if dir := user.HomeDirectory(); dir != "" {
		fields = append(fields, render.SummaryField{Label: "Home Directory", Value: dir})
	}

	return fields
}
