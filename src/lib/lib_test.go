package lib

import (
	"testing"
)

func TestLib(t *testing.T) {
	if got := GenRatings(); got == nil  {
		t.Error("getRankings did not work.")
	}
}
