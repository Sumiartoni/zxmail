package quota

import "testing"

func TestDeriveStatus(t *testing.T) {
	perMinute := 10
	daily := 100
	monthly := 1000

	tests := []struct {
		name          string
		enabled       bool
		perMinuteUsed int
		dailyUsed     int
		monthlyUsed   int
		expected      string
		limited       bool
		exceededCount int
	}{
		{
			name:          "disabled credential stays disabled",
			enabled:       false,
			perMinuteUsed: 99,
			dailyUsed:     999,
			monthlyUsed:   9999,
			expected:      "disabled",
			limited:       false,
			exceededCount: 0,
		},
		{
			name:          "enabled within limits",
			enabled:       true,
			perMinuteUsed: 3,
			dailyUsed:     40,
			monthlyUsed:   400,
			expected:      "enabled",
			limited:       false,
			exceededCount: 0,
		},
		{
			name:          "per minute limit exceeded",
			enabled:       true,
			perMinuteUsed: 10,
			dailyUsed:     40,
			monthlyUsed:   400,
			expected:      "limited",
			limited:       true,
			exceededCount: 1,
		},
		{
			name:          "multiple limits exceeded",
			enabled:       true,
			perMinuteUsed: 15,
			dailyUsed:     100,
			monthlyUsed:   1200,
			expected:      "limited",
			limited:       true,
			exceededCount: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status, limited, exceeded := deriveStatus(
				test.enabled,
				&perMinute,
				test.perMinuteUsed,
				&daily,
				test.dailyUsed,
				&monthly,
				test.monthlyUsed,
			)
			if status != test.expected {
				t.Fatalf("expected status %q, got %q", test.expected, status)
			}
			if limited != test.limited {
				t.Fatalf("expected limited=%v, got %v", test.limited, limited)
			}
			if len(exceeded) != test.exceededCount {
				t.Fatalf("expected %d exceeded limits, got %d", test.exceededCount, len(exceeded))
			}
		})
	}
}
