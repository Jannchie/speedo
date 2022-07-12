package speedo

import (
	"testing"
	"time"
)

func TestSpeedo(t *testing.T) {
	NewSpeedometer(Config{Log: true, Server: "http://:34747", PostIntervalSEC: 5, Name: "Consume"})
	ticker := time.NewTicker(time.Second)
	<-ticker.C
}

func TestSpeedoWithServer(t *testing.T) {
	s := NewSpeedometer(Config{Log: true, Server: "http://:80", PostIntervalSEC: 5})
	ticker := time.NewTicker(time.Second)
	count := 20
	for i := 0; i < count; i++ {
		<-ticker.C
		s.AddValue(1)
	}
	if int(s.value) != count {
		t.Error("not equal")
	}
}

func TestProgressSpeedo(t *testing.T) {
	s := NewProgressSpeedometer(15, Config{Name: "test", Log: true, Server: "http://:80", PrintIntervalSEC: 1, PostIntervalSEC: 1})
	s.SetTotal(50)
	ticker := time.NewTicker(time.Second / 10)
	count := 50
	for i := 0; i < count; i++ {
		<-ticker.C
		s.AddValue(1)
	}
	if int(s.value) != count {
		t.Error("not equal")
	}
}

func TestVariationSpeedo(t *testing.T) {
	s := NewVariationSpeedometer(Config{Name: "Raw Queue", Log: true, Server: "http://:80", PrintIntervalSEC: 1, PostIntervalSEC: 1})
	ticker := time.NewTicker(time.Second / 100)
	count := 1000
	for i := 0; i < count; i++ {
		<-ticker.C
		if count > 100 {
			s.SetValue(int64(1000 - i))
		} else {
			s.AddValue(1)
		}
	}
}
