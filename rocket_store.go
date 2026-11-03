package rocketsapi

import (
	"rockets-api/rocket"
	"sort"
	"sync"
	"time"
)

type RocketStore interface {
	processRocketMessage(string, int, time.Time, rocket.Message) error
	getRocket(string) (rocket.Snapshot, bool)
	getRockets() []rocket.Snapshot
}

type rocketStore struct {
	rockets map[string]rocket.Rocket
	mtx     sync.RWMutex
}

func NewRocketStore() *rocketStore {
	return &rocketStore{
		rockets: make(map[string]rocket.Rocket),
	}
}

func (s *rocketStore) processRocketMessage(channel string, msgNumber int, msgTime time.Time, msg rocket.Message) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	r, ok := s.rockets[channel]
	if !ok {
		r = rocket.New(channel)
	}

	err := r.ProcessMessage(msgNumber, msgTime, msg)
	s.rockets[channel] = r
	return err
}

func (s *rocketStore) getRocket(channel string) (rocket.Snapshot, bool) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	r, ok := s.rockets[channel]
	if !ok || r.Next() <= 1 {
		return rocket.Snapshot{}, false
	}

	return r.Snapshot(), true
}

func (s *rocketStore) getRockets() []rocket.Snapshot {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	rockets := make([]rocket.Snapshot, 0)
	for _, rocket := range s.rockets {
		if rocket.Next() > 1 {
			rockets = append(rockets, rocket.Snapshot())
		}
	}

	sort.Slice(rockets, func(i, j int) bool {
		if rockets[i].Speed != rockets[j].Speed {
			return rockets[i].Speed > rockets[j].Speed
		}

		return rockets[i].Channel < rockets[j].Channel
	})

	return rockets
}
