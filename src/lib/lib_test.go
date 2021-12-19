package lib

import (
	"testing"
)

func TestLib(t *testing.T) {
	if got := getRankings(); got == nil  {
		t.Error("getRankings did not work.")
	}
}
