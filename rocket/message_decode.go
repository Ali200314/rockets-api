package rocket

import (
	"encoding/json"
	"fmt"
)

const (
	messageTypeLaunched       = "RocketLaunched"
	messageTypeSpeedIncreased = "RocketSpeedIncreased"
	messageTypeSpeedDecreased = "RocketSpeedDecreased"
	messageTypeExploded       = "RocketExploded"
	messageTypeMissionChanged = "RocketMissionChanged"
)

func DecodeMessage(msgType string, payload json.RawMessage) (Message, error) {
	var message Message

	switch msgType {
	case messageTypeLaunched:
		message = &launched{}
	case messageTypeSpeedIncreased:
		message = &speedIncreased{}
	case messageTypeSpeedDecreased:
		message = &speedDecreased{}
	case messageTypeExploded:
		message = &exploded{}
	case messageTypeMissionChanged:
		message = &missionChanged{}
	default:
		return nil, fmt.Errorf("unsupported message type: %s", msgType)
	}

	if err := json.Unmarshal(payload, message); err != nil {
		return nil, fmt.Errorf("unable to unmarshal %s: %w", msgType, err)
	}

	if err := message.validate(); err != nil {
		return nil, err
	}

	return message, nil
}
