package v1alpha1

import "testing"

func TestFalconClusterGuardFalconAPI_API(t *testing.T) {
	var nilAPI *FalconClusterGuardFalconAPI
	if got := nilAPI.API(); got != nil {
		t.Errorf("API() on nil = %+v, want nil", got)
	}

	fa := &FalconClusterGuardFalconAPI{
		FalconAPI:    FalconAPI{CloudRegion: "us-1", ClientId: "id", ClientSecret: "secret"},
		HostOverride: "api.example.com",
	}
	got := fa.API()
	if got.HostOverride != "api.example.com" || got.CloudRegion != "us-1" || got.ClientId != "id" {
		t.Errorf("API() = %+v, want the FalconAPI with HostOverride api.example.com", got)
	}
	if cfg := got.ApiConfig(); cfg.Host() != "api.example.com" {
		t.Errorf("ApiConfig().Host() = %q, want api.example.com", cfg.Host())
	}

	got.ClientId = "changed"
	if fa.ClientId != "id" {
		t.Error("API() must return a copy, not a pointer into the spec")
	}
}

func TestFalconClusterGuard_SetFalconAPISpecKeepsHostOverride(t *testing.T) {
	fcg := &FalconClusterGuard{Spec: FalconClusterGuardSpec{
		FalconAPI: &FalconClusterGuardFalconAPI{HostOverride: "api.example.com"},
	}}

	// As InjectFalconSecretData does with the credentials from a FalconSecret.
	api := fcg.GetFalconAPISpec()
	api.ClientId = "injected-id"
	fcg.SetFalconAPISpec(api)
	if got := fcg.Spec.FalconAPI; got.ClientId != "injected-id" || got.HostOverride != "api.example.com" {
		t.Errorf("Spec.FalconAPI = %+v, want the injected client ID and HostOverride api.example.com", got)
	}

	empty := &FalconClusterGuard{}
	if empty.GetFalconAPISpec() != nil {
		t.Error("GetFalconAPISpec() should be nil when falcon_api is not set")
	}
	empty.SetFalconAPISpec(&FalconAPI{ClientId: "injected-id"})
	if empty.Spec.FalconAPI == nil || empty.Spec.FalconAPI.ClientId != "injected-id" {
		t.Errorf("SetFalconAPISpec() on an unset falcon_api = %+v, want the injected client ID", empty.Spec.FalconAPI)
	}
}
