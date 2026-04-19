package interactive

import "testing"

func TestSelectHeightForOptionsIncludesTitleAndDescriptionRows(t *testing.T) {
	cases := []struct {
		name    string
		options int
		want    int
	}{
		{name: "zero still leaves one option row", options: 0, want: 3},
		{name: "top level actions", options: 4, want: 6},
		{name: "selected worktree actions", options: 3, want: 5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := selectHeightForOptions(tc.options); got != tc.want {
				t.Fatalf("selectHeightForOptions(%d) = %d, want %d", tc.options, got, tc.want)
			}
		})
	}
}

func TestCappedSelectHeightForOptionsLimitsVisibleRows(t *testing.T) {
	if got := cappedSelectHeightForOptions(12, 5); got != 7 {
		t.Fatalf("cappedSelectHeightForOptions(12, 5) = %d, want 7", got)
	}
	if got := cappedSelectHeightForOptions(2, 5); got != 4 {
		t.Fatalf("cappedSelectHeightForOptions(2, 5) = %d, want 4", got)
	}
}

func TestWorktreeBrowserSelectHeightIsFiveVisibleRows(t *testing.T) {
	if got := selectHeightForVisibleOptions(worktreeBrowserVisibleRows); got != 7 {
		t.Fatalf("worktree browser select height = %d, want 7", got)
	}
}

func TestTopLevelActionSetIsFixed(t *testing.T) {
	want := []Action{ActionCreate, ActionList, ActionClean, ActionQuit}
	if len(topLevelActions) != len(want) {
		t.Fatalf("topLevelActions length = %d, want %d", len(topLevelActions), len(want))
	}
	for i := range want {
		if topLevelActions[i] != want[i] {
			t.Fatalf("topLevelActions[%d] = %q, want %q", i, topLevelActions[i], want[i])
		}
	}
	if got := len(topLevelActionOptions()); got != len(want) {
		t.Fatalf("topLevelActionOptions length = %d, want %d", got, len(want))
	}
}
