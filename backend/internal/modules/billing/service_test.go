package billing

import "testing"

func TestSubscriptionStatusRequiresCredentialLimit(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{name: "active", status: "active", want: false},
		{name: "trialing", status: "trialing", want: false},
		{name: "past due", status: "past_due", want: true},
		{name: "expired", status: "expired", want: true},
		{name: "suspended", status: "suspended", want: true},
		{name: "mixed case trimmed", status: " Past_Due ", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := subscriptionStatusRequiresCredentialLimit(tt.status); got != tt.want {
				t.Fatalf("subscriptionStatusRequiresCredentialLimit(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
