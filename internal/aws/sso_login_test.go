package aws

import "testing"

func TestSSOScopesDefaultsToAccountAccess(t *testing.T) {
	scopes := ssoScopes(ProfileInfo{})
	if len(scopes) != 1 || scopes[0] != ssoDefaultScope {
		t.Fatalf("ssoScopes() = %v, want [%s]", scopes, ssoDefaultScope)
	}
}

func TestSSOScopesSplitsConfiguredScopes(t *testing.T) {
	scopes := ssoScopes(ProfileInfo{SSOScopes: "sso:account:access, custom:scope other:scope"})
	want := []string{"sso:account:access", "custom:scope", "other:scope"}
	if len(scopes) != len(want) {
		t.Fatalf("ssoScopes length = %d, want %d: %v", len(scopes), len(want), scopes)
	}
	for i := range want {
		if scopes[i] != want[i] {
			t.Fatalf("ssoScopes[%d] = %q, want %q", i, scopes[i], want[i])
		}
	}
}

func TestValidateSSOProfileRequiresCompleteSettings(t *testing.T) {
	err := validateSSOProfile(ProfileInfo{Name: "dev", SSOStartURL: "https://example.awsapps.com/start"})
	if err == nil {
		t.Fatal("validateSSOProfile() error = nil, want missing settings error")
	}
}

func TestSSOTokenCachePathUsesSessionNameWhenPresent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	withSession, err := ssoTokenCachePath(ProfileInfo{SSOSession: "dev-session", SSOStartURL: "https://example.awsapps.com/start"})
	if err != nil {
		t.Fatalf("ssoTokenCachePath() with session error: %v", err)
	}
	withoutSession, err := ssoTokenCachePath(ProfileInfo{SSOStartURL: "https://example.awsapps.com/start"})
	if err != nil {
		t.Fatalf("ssoTokenCachePath() without session error: %v", err)
	}
	if withSession == withoutSession {
		t.Fatalf("cache path with session should differ from legacy start URL path: %q", withSession)
	}
}
