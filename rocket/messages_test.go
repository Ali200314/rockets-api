package rocket

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageValidate(t *testing.T) {
	tests := []struct {
		name    string
		message Message
		wantErr string
	}{
		{
			name:    "valid launch",
			message: &launched{Type: "Falcon-9", LaunchSpeed: 500, Mission: "ARTEMIS"},
		},
		{
			name:    "invalid launch type",
			message: &launched{LaunchSpeed: 500, Mission: "ARTEMIS"},
			wantErr: "type is required",
		},
		{
			name:    "valid speed increase",
			message: &speedIncreased{By: 300},
		},
		{
			name:    "invalid speed increase",
			message: &speedIncreased{},
			wantErr: "by should be greater than 0",
		},
		{
			name:    "valid speed decrease",
			message: &speedDecreased{By: 100},
		},
		{
			name:    "invalid speed decrease",
			message: &speedDecreased{},
			wantErr: "by should be greater than 0",
		},
		{
			name:    "valid explosion",
			message: &exploded{Reason: "ENGINE_FAILURE"},
		},
		{
			name:    "invalid explosion",
			message: &exploded{},
			wantErr: "reason is required",
		},
		{
			name:    "valid mission change",
			message: &missionChanged{NewMission: "SHUTTLE-MIR"},
		},
		{
			name:    "invalid mission change",
			message: &missionChanged{},
			wantErr: "newMission is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.message.validate()

			if test.wantErr == "" {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.EqualError(t, err, test.wantErr)
		})
	}
}

func TestMessageApply(t *testing.T) {
	tests := []struct {
		name    string
		message Message
		rocket  Rocket
		want    Rocket
		wantErr string
	}{
		{
			name:    "launch",
			message: &launched{Type: "Falcon-9", LaunchSpeed: 500, Mission: "ARTEMIS"},
			rocket:  Rocket{next: 1, speed: 100},
			want:    Rocket{next: 1, rocketType: "Falcon-9", speed: 500, mission: "ARTEMIS"},
		},
		{
			name:    "increase speed",
			message: &speedIncreased{By: 300},
			rocket:  Rocket{next: 2, speed: 500},
			want:    Rocket{next: 2, speed: 800},
		},
		{
			name:    "decrease speed",
			message: &speedDecreased{By: 100},
			rocket:  Rocket{next: 2, speed: 500},
			want:    Rocket{next: 2, speed: 400},
		},
		{
			name:    "rejects decrease below zero",
			message: &speedDecreased{By: 600},
			rocket:  Rocket{next: 2, speed: 500},
			want:    Rocket{next: 2, speed: 500},
			wantErr: "speed cannot become negative",
		},
		{
			name:    "explode",
			message: &exploded{Reason: "ENGINE_FAILURE"},
			rocket:  Rocket{next: 2, speed: 500},
			want:    Rocket{next: 2, explodeReason: "ENGINE_FAILURE"},
		},
		{
			name:    "change mission",
			message: &missionChanged{NewMission: "SHUTTLE-MIR"},
			rocket:  Rocket{next: 2, mission: "ARTEMIS"},
			want:    Rocket{next: 2, mission: "SHUTTLE-MIR"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.message.apply(&test.rocket)
			if test.wantErr != "" {
				require.EqualError(t, err, test.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.want, test.rocket)
		})
	}
}
