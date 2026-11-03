package rocket

import (
	"errors"
	"fmt"
	"time"
)

type state int

const (
	stateInactive state = iota
	stateActive
	stateExploded
)

type pending struct {
	message Message
	time    time.Time
}

type Rocket struct {
	channel       string
	rocketType    string
	speed         float32
	mission       string
	explodeReason string
	updatedAt     time.Time

	next    int
	pending map[int]pending
}

type Snapshot struct {
	Channel       string    `json:"channel"`
	Type          string    `json:"type"`
	Speed         float32   `json:"speed"`
	Mission       string    `json:"mission"`
	ExplodeReason string    `json:"explode_reason,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func New(channel string) Rocket {
	return Rocket{
		channel: channel,
		next:    1,
		pending: make(map[int]pending),
	}
}

func (r *Rocket) ProcessMessage(msgNumber int, msgTime time.Time, msg Message) error {
	if r.next > msgNumber {
		return nil
	}
	if r.state() == stateExploded {
		return errors.New("cannot process message after rocket exploded")
	}

	r.pending[msgNumber] = pending{message: msg, time: msgTime}

	for {
		pending, ok := r.pending[r.next]
		if !ok {
			return nil
		}
		if err := pending.message.apply(r); err != nil {
			delete(r.pending, r.next)
			return fmt.Errorf("unable to apply message %d: %w", r.next, err)
		}
		r.updatedAt = pending.time

		delete(r.pending, r.next)
		r.next++
	}
}

func (r Rocket) state() state {
	if r.explodeReason != "" {
		return stateExploded
	}
	if r.next == 1 {
		return stateInactive
	}
	return stateActive
}

func (r Rocket) Snapshot() Snapshot {
	return Snapshot{
		Channel:       r.channel,
		Type:          r.rocketType,
		Speed:         r.speed,
		Mission:       r.mission,
		ExplodeReason: r.explodeReason,
		UpdatedAt:     r.updatedAt,
	}
}

func (r Rocket) Next() int {
	return r.next
}
