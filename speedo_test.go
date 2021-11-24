package speedo

import (
	"testing"
	"time"
)

func TestSpeedo(t *testing.T) {
	s := NewSpeedometer(Config{Log: true, Server: "http://:8080", PostIntervalSEC: 5})
	ticker := time.NewTicker(time.Second)
	count := 20
	for i := 0; i < count; i++ {
		<-ticker.C
		s.AddCount(1)
	}
	if s.count != uint64(count) {
		t.Error("not equal")
	}
}
