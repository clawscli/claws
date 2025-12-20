package profile

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadProfiles(t *testing.T) {
	// Create temp ~/.aws directory
	tmpHome := t.TempDir()
	awsDir := filepath.Join(tmpHome, ".aws")
	require.NoError(t, os.MkdirAll(awsDir, 0755))

	// Override HOME for testing
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	// Create config file
	configContent := `[default]
region = us-west-2
output = json

[profile dev]
region = us-east-1
role_arn = arn:aws:iam::123456789012:role/DevRole
source_profile = default

[profile sso-test]
sso_start_url = https://my-sso.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = AdministratorAccess
region = us-west-2
`
	require.NoError(t, os.WriteFile(filepath.Join(awsDir, "config"), []byte(configContent), 0600))

	// Create credentials file
	credContent := `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[prod]
aws_access_key_id = AKIAI44QH8DHBEXAMPLE
aws_secret_access_key = je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY
`
	require.NoError(t, os.WriteFile(filepath.Join(awsDir, "credentials"), []byte(credContent), 0600))

	// Load profiles
	profiles, err := loadProfiles()
	require.NoError(t, err)

	// Verify default profile
	defaultProfile := profiles["default"]
	require.NotNil(t, defaultProfile)
	assert.Equal(t, "us-west-2", defaultProfile.Region)
	assert.Equal(t, "json", defaultProfile.Output)
	assert.True(t, defaultProfile.HasCredentials)
	assert.True(t, defaultProfile.InConfig)
	assert.True(t, defaultProfile.InCredentials)

	// Verify dev profile (assume role)
	devProfile := profiles["dev"]
	require.NotNil(t, devProfile)
	assert.Equal(t, "us-east-1", devProfile.Region)
	assert.Equal(t, "arn:aws:iam::123456789012:role/DevRole", devProfile.RoleArn)
	assert.Equal(t, "default", devProfile.SourceProfile)
	assert.True(t, devProfile.InConfig)
	assert.False(t, devProfile.InCredentials)

	// Verify sso-test profile
	ssoProfile := profiles["sso-test"]
	require.NotNil(t, ssoProfile)
	assert.Equal(t, "https://my-sso.awsapps.com/start", ssoProfile.SSOStartURL)
	assert.Equal(t, "us-east-1", ssoProfile.SSORegion)
	assert.Equal(t, "123456789012", ssoProfile.SSOAccountID)
	assert.Equal(t, "AdministratorAccess", ssoProfile.SSORoleName)

	// Verify prod profile (credentials only)
	prodProfile := profiles["prod"]
	require.NotNil(t, prodProfile)
	assert.True(t, prodProfile.HasCredentials)
	assert.False(t, prodProfile.InConfig)
	assert.True(t, prodProfile.InCredentials)
}

func TestProfileDAO_List(t *testing.T) {
	// Create temp ~/.aws directory
	tmpHome := t.TempDir()
	awsDir := filepath.Join(tmpHome, ".aws")
	require.NoError(t, os.MkdirAll(awsDir, 0755))

	// Override HOME for testing
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	// Create minimal config
	configContent := `[default]
region = us-west-2
`
	require.NoError(t, os.WriteFile(filepath.Join(awsDir, "config"), []byte(configContent), 0600))

	// Create DAO and list
	dao, err := NewProfileDAO(context.Background())
	require.NoError(t, err)

	resources, err := dao.List(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, resources)

	// Check that default profile exists
	found := false
	for _, r := range resources {
		if r.GetName() == "default" {
			found = true
			pr := r.(*ProfileResource)
			assert.Equal(t, "us-west-2", pr.Data.Region)
			break
		}
	}
	assert.True(t, found, "default profile should exist")
}
