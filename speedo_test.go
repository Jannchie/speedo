package speedo

import (
	"log"
	"testing"
	"time"
)

func TestSpeedoWithServer(t *testing.T) {
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

func TestProgressSpeedo(t *testing.T) {
	s := NewProgressSpeedometer(15, Config{Log: true, PrintIntervalSEC: 1})
	ticker := time.NewTicker(time.Second / 10)
	count := 50
	for i := 0; i < count; i++ {
		<-ticker.C
		s.AddCount(1)
		log.Println(s.GetStatusString())
	}
	if s.count != uint64(count) {
		t.Error("not equal")
	}
}

func TestVariationSpeedo(t *testing.T) {
	s := NewVariationSpeedometer(Config{Log: true, PrintIntervalSEC: 1})
	ticker := time.NewTicker(time.Second / 100)
	count := 1000
	for i := 0; i < count; i++ {
		<-ticker.C
		if count > 100 {
			s.SetValue((uint64)(1000 - i))
		} else {
			s.AddCount(1)
		}
	}
}
