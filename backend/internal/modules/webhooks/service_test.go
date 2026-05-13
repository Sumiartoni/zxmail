package webhooks

import "testing"

func TestIsHardBounce(t *testing.T) {
	tests := []struct {
		name       string
		bounceType string
		reason     string
		expected   bool
	}{
		{name: "explicit hard", bounceType: "hard", reason: "", expected: true},
		{name: "permanent keyword", bounceType: "", reason: "Permanent failure: user unknown", expected: true},
		{name: "soft bounce", bounceType: "soft", reason: "mailbox full", expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := isHardBounce(test.bounceType, test.reason)
			if actual != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}
