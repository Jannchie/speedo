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

type PostData struct {
	Name      string    `json:"name" gorm:"index"`
	SID       string    `json:"sid" gorm:"primaryKey"`
	Value     int64     `json:"Value"`
	CreatedAt time.Time `json:"created_at"`
}

type PostInfo struct {
	ID              uint64    `json:"id" gorm:"primaryKey"`
	SID             string    `json:"sid" form:"sid" gorm:"uniqueIndex"`
	Name            string    `json:"name" form:"name" gorm:"index"`
	Type            uint8     `json:"type" form:"total"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	PostIntervalSEC int64     `json:"post_interval_sec" form:"post_interval_sec"`
	Total           uint64    `json:"total" form:"total"`
}

type Speedometer struct {
	id               string
	name             string
	log              bool
	server           string
	value            int64
	lastValue        int64
	total            uint64
	postIntervalSEC  int64
	printIntervalSEC int64
	guard            chan struct{}
	duration         time.Duration
	history          []int64
	mutex            sync.RWMutex
	speedoType       uint8
}

type SpeedStat struct {
	Value int64  `json:"value"`
	Speed int64  `json:"speed"`
	Total uint64 `json:"total"`
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
	ss.Value = s.value
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
			s.history = append(s.history, s.value)
			if l < 60 {
				l += 1
			} else {
				s.history = s.history[1:]
			}
			s.mutex.Unlock()
		}
	}
}

func (s *Speedometer) AddValue(n int64) {
	s.SetValue(s.value + n)
}

func (s *Speedometer) SetValue(n int64) {
	s.mutex.Lock()
	s.value = n
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
		statusWithoutName = fmt.Sprintf("  Speed: %10d/min,     Total: %8d", stat.Speed, stat.Value)
	case Variation:
		statusWithoutName = fmt.Sprintf("Current: %14d, Variation: %+7d/min", stat.Value, stat.Speed)
	case Progress:
		statusWithoutName = fmt.Sprintf("Percent: %13d%%,    Values: %5d/%5d", stat.Value*100/int64(stat.Total), stat.Value, stat.Total)
	}
	if s.name != "" {
		return fmt.Sprintf("%36s %s", s.name, statusWithoutName)
	} else {
		return fmt.Sprintf("%36s %s", s.id, statusWithoutName)
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
	var lastPost int64 = 0
	for {
		select {
		case <-ticker.C:
			s.postLog(&lastPost)
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

func (s *Speedometer) postLog(lastPost *int64) {
	data := s.GetStat()
	postData := PostData{
		SID:   s.id,
		Name:  s.name,
		Value: data.Value,
	}
	b, err := json.Marshal(postData)
	if err != nil {
		log.Println(err)
	}
	_, err = http.Post(
		fmt.Sprintf(`%s/stat`, s.server),
		"application/json",
		bytes.NewReader(b),
	)
	if err != nil {
		log.Println(err)
	}
}

func (s *Speedometer) postInfo() {
	ticker := time.NewTicker(time.Second * time.Duration(s.postIntervalSEC) * 10)
	for {
		select {
		case <-ticker.C:
			b, _ := json.Marshal(PostInfo{
				SID:             s.id,
				Name:            s.name,
				Type:            s.speedoType,
				Total:           s.total,
				PostIntervalSEC: s.postIntervalSEC,
			})
			_, err := http.Post(
				fmt.Sprintf(`%s/info`, s.server),
				"application/json",
				bytes.NewReader(b),
			)
			if err != nil {
				log.Println(err)
			}
		case _, ok := <-s.guard:
			if !ok {
				ticker.Stop()
				return
			}
		}
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
