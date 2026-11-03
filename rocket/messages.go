package rocket

import (
	"errors"
	"strings"
)

type Message interface {
	apply(*Rocket) error
	validate() error
}

type launched struct {
	Type        string  `json:"type"`
	LaunchSpeed float32 `json:"launchSpeed"`
	Mission     string  `json:"mission"`
}

type speedIncreased struct {
	By float32 `json:"by"`
}

type speedDecreased struct {
	By float32 `json:"by"`
}

type exploded struct {
	Reason string `json:"reason"`
}

type missionChanged struct {
	NewMission string `json:"newMission"`
}

func (m *launched) apply(r *Rocket) error {
	if r.state() != stateInactive {
		return errors.New("rocket must be inactive to launch")
	}

	r.rocketType = m.Type
	r.speed = m.LaunchSpeed
	r.mission = m.Mission
	return nil
}

func (m *launched) validate() error {
	if strings.TrimSpace(m.Type) == "" {
		return errors.New("type is required")
	}
	if m.LaunchSpeed <= 0 {
		return errors.New("launchSpeed should be greater than 0")
	}
	if strings.TrimSpace(m.Mission) == "" {
		return errors.New("mission is required")
	}
	return nil
}

func (m *speedIncreased) apply(r *Rocket) error {
	if r.state() != stateActive {
		return errors.New("rocket must be active to increase speed")
	}

	r.speed += m.By
	return nil
}

func (m *speedIncreased) validate() error {
	if m.By <= 0 {
		return errors.New("by should be greater than 0")
	}
	return nil
}

func (m *speedDecreased) apply(r *Rocket) error {
	if r.state() != stateActive {
		return errors.New("rocket must be active to decrease speed")
	}
	if m.By > r.speed {
		return errors.New("speed cannot become negative")
	}

	r.speed -= m.By
	return nil
}

func (m *speedDecreased) validate() error {
	if m.By <= 0 {
		return errors.New("by should be greater than 0")
	}
	return nil
}

func (m *exploded) apply(r *Rocket) error {
	if r.state() != stateActive {
		return errors.New("rocket must be active to explode")
	}

	r.explodeReason = m.Reason
	r.speed = 0
	return nil
}

func (m *exploded) validate() error {
	if strings.TrimSpace(m.Reason) == "" {
		return errors.New("reason is required")
	}
	return nil
}

func (m *missionChanged) apply(r *Rocket) error {
	if r.state() != stateActive {
		return errors.New("rocket must be active to change mission")
	}

	r.mission = m.NewMission
	return nil
}

func (m *missionChanged) validate() error {
	if strings.TrimSpace(m.NewMission) == "" {
		return errors.New("newMission is required")
	}
	return nil
}
