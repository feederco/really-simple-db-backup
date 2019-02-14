package pkg

import "testing"

func TestLastLines(t *testing.T) {
	noLastLines := lastLines([]string{}, 3)
	if noLastLines != "" {
		t.Error("Wrong. Got", noLastLines)
	}

	oneLine := lastLines([]string{"a"}, 3)
	if oneLine != "a" {
		t.Error("Wrong. Got", oneLine)
	}

	threeLines := lastLines([]string{"a", "b", "c"}, 3)
	if threeLines != "a\nb\nc" {
		t.Error("Wrong. Got", threeLines)
	}

	sixLines := lastLines([]string{"a", "b", "c", "d", "e", "f"}, 3)
	if sixLines != "d\ne\nf" {
		t.Error("Wrong. Got", sixLines)
	}
}
