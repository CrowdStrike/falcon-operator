package v1alpha1

import "testing"

func TestFalconClusterGuardImageAnalyzerSpec_IsEnabled(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name    string
		enabled *bool
		want    bool
	}{
		{"nil — disabled by default", nil, false},
		{"false — disabled", &falseVal, false},
		{"true — enabled", &trueVal, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &FalconClusterGuardImageAnalyzerSpec{Enabled: tt.enabled}
			if got := s.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
