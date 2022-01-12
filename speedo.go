package speedo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	Accumulation uint8 = 0
	Variation    uint8 = 1
	Progress     uint8 = 2
)

type Speedometer struct {
	id               string
	name             string
	log              bool
	server           string
	count            uint64
	total            uint64
	postIntervalSEC  int64
	printIntervalSEC int64
	guard            chan struct{}
	duration         time.Duration
	history          []uint64
	mutex            sync.RWMutex
	speedoType       uint8
}

type SpeedStat struct {
	Count    uint64 `json:"count"`
	Speed    int64  `json:"speed"`
	Progress uint64 `json:"progress"`
	Total    uint64 `json:"total"`
}

type Config struct {
	Name             string
	Log              bool
	Server           string
	PostIntervalSEC  int64
	PrintIntervalSEC int64
}

func (s *Speedometer) GetStat() SpeedStat {
	ss := SpeedStat{}
	ss.Count = s.count
	ss.Total = s.total
	var delta int64
	s.mutex.Lock()
	defer s.mutex.Unlock()
	count := len(s.history)
	if count <= 1 {
		return ss
	} else {
		deltaTime := int64(count-1) * int64(s.duration)
		delta = (int64)(s.history[count-1]) - int64(s.history[0])
		ss.Speed = (int64)(delta * int64(time.Minute) / deltaTime)
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
	s.SetValue(s.count + n)
}

func (s *Speedometer) SetValue(n uint64) {
	s.mutex.Lock()
	s.count = n
	s.mutex.Unlock()
}

func (s *Speedometer) SetTotal(n uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.total = n
}

func (s *Speedometer) GetStatusString() string {
	stat := s.GetStat()
	var statusWithoutName string

	switch s.speedoType {
	case Accumulation:
		statusWithoutName = fmt.Sprintf("Speed: %d/min Total: %d", stat.Speed, stat.Count)
	case Variation:
		statusWithoutName = fmt.Sprintf("Current: %d Variation: %d/min", stat.Count, stat.Speed)
	case Progress:
		statusWithoutName = fmt.Sprintf("Percent: %d%%, %d/%d", stat.Count*100/stat.Total, stat.Count, stat.Total)
	}

	if s.name != "" {
		return fmt.Sprintf("%s %s", s.name, statusWithoutName)
	} else {
		return statusWithoutName
	}
}

func (s *Speedometer) autoPrint() {
	ticker := time.NewTicker(time.Second * time.Duration(s.printIntervalSEC))
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

func (s *Speedometer) autoPost() {
	ticker := time.NewTicker(time.Second * time.Duration(s.postIntervalSEC))
	for {
		select {
		case <-ticker.C:
			s.postLog()
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

func (s *Speedometer) postLog() {
	data := s.GetStat()
	b, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
	}
	_, err = http.Post(
		fmt.Sprintf(`%s/stat/%s`, s.server, s.id),
		"application/json",
		bytes.NewReader(b),
	)
	if err != nil {
		log.Println(err)
	}
}

func (s *Speedometer) postInfo() {
	b, _ := json.Marshal(struct {
		Name string `json:"name"`
		Type uint8  `json:"type"`
	}{
		Name: s.name,
		Type: s.speedoType,
	})
	_, err := http.Post(
		fmt.Sprintf(`%s/info/%s`, s.server, s.id),
		"application/json",
		bytes.NewReader(b),
	)
	if err != nil {
		log.Println(err)
	}
}

func NewSpeedometer(config Config) *Speedometer {
	s := &Speedometer{
		name:             config.Name,
		log:              config.Log,
		server:           config.Server,
		postIntervalSEC:  config.PostIntervalSEC,
		printIntervalSEC: config.PrintIntervalSEC,
	}
	if s.postIntervalSEC == 0 {
		s.postIntervalSEC = 60
	}
	if s.printIntervalSEC == 0 {
		s.printIntervalSEC = 5
	}
	s.id = uuid.NewString()
	if s.duration == 0 {
		s.duration = time.Second * 1
	}
	if s.server != "" {
		go s.postInfo()
		go s.autoPost()
	}
	s.guard = make(chan struct{})
	go s.startTicker()
	if s.log {
		go s.autoPrint()
	}
	return s
}

func NewVariationSpeedometer(config Config) *Speedometer {
	s := NewSpeedometer(config)
	s.speedoType = Variation
	return s
}

func NewProgressSpeedometer(total uint64, config Config) *Speedometer {
	s := NewSpeedometer(config)
	s.total = total
	s.speedoType = Progress
	return s
}
