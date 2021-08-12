package speedo

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Speedometer struct {
	id       string
	name     string
	log      bool
	count    uint64
	guard    chan struct{}
	duration time.Duration
	history  []uint64
	mutex    sync.RWMutex
}

type SpeedStat struct {
	Count uint64
	Speed uint64
}

type Config struct {
	Name string
	Log  bool
}

func (s *Speedometer) GetStat() SpeedStat {
	ss := SpeedStat{}
	ss.Count = s.count
	var delta uint64
	s.mutex.Lock()
	defer s.mutex.Unlock()
	count := len(s.history)
	if count <= 1 {
		ss.Speed = 0
		return ss
	} else {
		deltaTime := uint64(count-1) * uint64(s.duration)
		delta = s.history[count-1] - s.history[0]
		ss.Speed = delta * uint64(time.Minute) / deltaTime
		return ss
	}
}

func (s *Speedometer) startTicker() {
	s.mutex.Lock()
	ticker := time.NewTicker(s.duration)
	s.mutex.Unlock()

	l := 0
	for {
		select {
		case _, ok := <-s.guard:
			if !ok {
				ticker.Stop()
				return
			}
		case <-ticker.C:
			s.mutex.Lock()
			s.history = append(s.history, s.count)
			if l < 60 {
				l += 1
			} else {
				s.history = s.history[1:]
			}
			s.mutex.Unlock()
		}
	}
}

func (s *Speedometer) AddCount(n uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.count += n
}

func (s *Speedometer) GetStatusString() string {
	stat := s.GetStat()
	if s.name != "" {
		return fmt.Sprintf("%s Speed: %d/min Total: %d", s.name, stat.Speed, stat.Count)
	} else {
		return fmt.Sprintf("Speed: %d/min Total: %d", stat.Speed, stat.Count)
	}
}

func (s *Speedometer) autoPrint() {
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ticker.C:
			log.Println(s.GetStatusString())
		case _, ok := <-s.guard:
			if !ok {
				ticker.Stop()
				return
			}
		}
	}
}

func (s *Speedometer) Stop() {
	s.mutex.Lock()
	s.guard <- struct{}{}
	s.mutex.Unlock()
}

func NewSpeedometer(config Config) *Speedometer {
	s := &Speedometer{name: config.Name, log: config.Log}
	s.id = uuid.NewString()
	if s.duration == 0 {
		s.duration = time.Second * 1
	}
	s.guard = make(chan struct{})
	go s.startTicker()
	if s.log {
		go s.autoPrint()
	}
	return s
}
