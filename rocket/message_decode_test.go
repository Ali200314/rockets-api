package rocket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeMessage(t *testing.T) {
	t.Run("decodes supported message types", func(t *testing.T) {
		tests := []struct {
			name        string
			messageType string
			message     json.RawMessage
			want        Message
		}{
			{
				name:        "launch",
				messageType: messageTypeLaunched,
				message:     json.RawMessage(`{"type":"Falcon-9","launchSpeed":500,"mission":"ARTEMIS"}`),
				want:        &launched{Type: "Falcon-9", LaunchSpeed: 500, Mission: "ARTEMIS"},
			},
			{
				name:        "speed increase",
				messageType: messageTypeSpeedIncreased,
				message:     json.RawMessage(`{"by":300}`),
				want:        &speedIncreased{By: 300},
			},
			{
				name:        "speed decrease",
				messageType: messageTypeSpeedDecreased,
				message:     json.RawMessage(`{"by":100}`),
				want:        &speedDecreased{By: 100},
			},
			{
				name:        "explosion",
				messageType: messageTypeExploded,
				message:     json.RawMessage(`{"reason":"ENGINE_FAILURE"}`),
				want:        &exploded{Reason: "ENGINE_FAILURE"},
			},
			{
				name:        "mission change",
				messageType: messageTypeMissionChanged,
				message:     json.RawMessage(`{"newMission":"SHUTTLE-MIR"}`),
				want:        &missionChanged{NewMission: "SHUTTLE-MIR"},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				got, err := DecodeMessage(test.messageType, test.message)

				require.NoError(t, err)
				assert.Equal(t, test.want, got)
			})
		}
	})

	t.Run("rejects invalid message types and payloads", func(t *testing.T) {
		tests := []struct {
			name        string
			messageType string
			message     json.RawMessage
			errContains string
		}{
			{
				name:        "invalid launch payload",
				messageType: messageTypeLaunched,
				message:     json.RawMessage(`{`),
				errContains: "unable to unmarshal RocketLaunched: unexpected end of JSON input",
			},
			{
				name:        "invalid speed increase payload",
				messageType: messageTypeSpeedIncreased,
				message:     json.RawMessage(`{`),
				errContains: "unable to unmarshal RocketSpeedIncreased: unexpected end of JSON input",
			},
			{
				name:        "invalid speed decrease payload",
				messageType: messageTypeSpeedDecreased,
				message:     json.RawMessage(`{`),
				errContains: "unable to unmarshal RocketSpeedDecreased: unexpected end of JSON input",
			},
			{
				name:        "invalid explosion payload",
				messageType: messageTypeExploded,
				message:     json.RawMessage(`{`),
				errContains: "unable to unmarshal RocketExploded: unexpected end of JSON input",
			},
			{
				name:        "invalid mission change payload",
				messageType: messageTypeMissionChanged,
				message:     json.RawMessage(`{`),
				errContains: "unable to unmarshal RocketMissionChanged: unexpected end of JSON input",
			},
			{
				name:        "unknown message type",
				messageType: "UnsupportedMessage",
				message:     json.RawMessage(`{}`),
				errContains: "unsupported message type",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				// When the invalid message is decoded.
				_, err := DecodeMessage(test.messageType, test.message)

				// Then decoding fails with the expected error.
				require.Error(t, err)
				assert.ErrorContains(t, err, test.errContains)
			})
		}
	})
}
