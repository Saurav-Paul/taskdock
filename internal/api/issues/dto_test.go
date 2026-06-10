package issues

import "testing"

func TestBranchName(t *testing.T) {
	SetBranchPrefix("feature/")
	defer SetBranchPrefix("")

	cases := []struct {
		key, title, want string
	}{
		// long title cuts at a word boundary, never mid-word
		{
			"PRO-944", "Phase 6: notification attachment bridge e2e tests",
			"feature/pro-944-phase-6-notification-attachment-bridge-e2e-tests",
		},
		{"TD-5", "short one", "feature/td-5-short-one"},
		{
			"AUR-8", "Fix crash on device rotation during video playback in fullscreen landscape mode",
			"feature/aur-8-fix-crash-on-device-rotation-during-video-playback-in-fullscreen",
		},
		// punctuation collapses to single dashes
		{"TD-9", "weird:: title!! (v2)", "feature/td-9-weird-title-v2"},
	}

	for _, c := range cases {
		if got := BranchName(c.key, c.title); got != c.want {
			t.Errorf("BranchName(%q, %q) = %q, want %q", c.key, c.title, got, c.want)
		}
	}

	// slug part (after prefix) never exceeds the cap
	long := BranchName("TD-1", "a-very-long-single-word-that-cannot-be-split-anywhere-soooooooooooooooooooooo-long")
	if len(long) > len("feature/")+maxBranchSlugLen {
		t.Errorf("slug exceeds cap: %d chars: %s", len(long), long)
	}
}
