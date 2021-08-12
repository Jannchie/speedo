package speedo

import (
	"testing"
)

func TestSpeedo(t *testing.T) {
	s := NewSpeedometer(Config{Log: true})
	for i := 0; i < 100; i++ {
		s.AddCount(1)
	}
	if s.count != 100 {
		t.Error("not equal")
	}
}
