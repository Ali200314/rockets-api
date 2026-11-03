package rocketsapi

import (
	"encoding/json"
	"rockets-api/rocket"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestControllerProcessRocketMessage(t *testing.T) {
	t.Run("successfully proccesses rocket launch", func(t *testing.T) {
		// Given: controller
		store := NewRocketStore()

		// Given: message
		msg, err := rocket.DecodeMessage(
			"RocketLaunched",
			json.RawMessage(`{"type":"Falcon-9","launchSpeed":500,"mission":"ARTEMIS"}`),
		)
		require.NoError(t, err)

		// When: is called
		require.NoError(t, store.processRocketMessage("123", 1, time.Time{}, msg))

		// Then: rocket is stored
		_, ok := store.rockets["123"]
		assert.True(t, ok)
	})

	t.Run("persists state applied before a later pending message fails", func(t *testing.T) {
		// Given a launched rocket with a later mission change waiting for message 2.
		store := NewRocketStore()
		launchRocket(t, store, "rocket-a", 500)

		missionChange := testStoreMessage(t, "RocketMissionChanged", `{"newMission":"SHUTTLE-MIR"}`)
		require.NoError(t, store.processRocketMessage("rocket-a", 3, time.Time{}, missionChange))

		// When message 2 explodes the rocket and processing message 3 fails.
		explosion := testStoreMessage(t, "RocketExploded", `{"reason":"ENGINE_FAILURE"}`)
		err := store.processRocketMessage("rocket-a", 2, time.Time{}, explosion)

		// Then the successful state change from message 2 is persisted.
		require.EqualError(t, err, "unable to apply message 3: rocket must be active to change mission")
		got, ok := store.getRocket("rocket-a")
		require.True(t, ok)
		assert.Equal(t, float32(0), got.Speed)
		assert.Equal(t, "ENGINE_FAILURE", got.ExplodeReason)
	})
}

func TestControllerGetRocket(t *testing.T) {
	t.Run("returns an existing rocket", func(t *testing.T) {
		// Given: a controller containing one rocket.
		controller := NewRocketStore()
		launchRocket(t, controller, "rocket-a", 500)

		// When the existing rocket is requested.
		got, ok := controller.getRocket("rocket-a")

		// Then its current snapshot is returned.
		require.True(t, ok)
		assert.Equal(t, "rocket-a", got.Channel)
		assert.Equal(t, float32(500), got.Speed)
	})

	t.Run("returns no rocket when the channel is missing", func(t *testing.T) {
		// Given: an empty controller.
		controller := NewRocketStore()

		// When a missing r is requested.
		r, ok := controller.getRocket("missing")

		// Then: no rocket is returned
		assert.False(t, ok)
		assert.Equal(t, rocket.Snapshot{}, r)
	})

	t.Run("returns no rocket when no message has been applied", func(t *testing.T) {
		// Given a controller containing only a newly constructed rocket.
		controller := NewRocketStore()
		controller.rockets["rocket-a"] = rocket.Rocket{}

		// When the rocket is requested.
		r, ok := controller.getRocket("rocket-a")

		// Then: no rocket is returned
		assert.False(t, ok)
		assert.Equal(t, rocket.Snapshot{}, r)
	})
}

func TestControllerGetRockets(t *testing.T) {
	t.Run("returns rockets sorted by speed and channel", func(t *testing.T) {
		// Given rockets with different speeds and equal-speed tie-breakers.
		controller := NewRocketStore()
		launchRocket(t, controller, "rocket-b", 500)
		launchRocket(t, controller, "rocket-a", 500)
		launchRocket(t, controller, "rocket-c", 1000)

		// When rockets are requested.
		rockets := controller.getRockets()

		// Then rockets are returned by descending speed, with channel as a tie-breaker.
		assert.Equal(t, []string{"rocket-c", "rocket-a", "rocket-b"}, []string{
			rockets[0].Channel,
			rockets[1].Channel,
			rockets[2].Channel,
		})
	})

	t.Run("returns no rockets when the store is empty", func(t *testing.T) {
		// Given an empty controller.
		controller := NewRocketStore()

		// When rockets are requested.
		rockets := controller.getRockets()

		// Then no rockets are returned.
		assert.Empty(t, rockets)
		assert.NotNil(t, rockets)
	})
}

func launchRocket(t *testing.T, controller *rocketStore, channel string, speed float32) {
	t.Helper()

	message, err := json.Marshal(map[string]any{
		"type":        "Falcon-9",
		"launchSpeed": speed,
		"mission":     "ARTEMIS",
	})
	require.NoError(t, err)

	decoded, err := rocket.DecodeMessage("RocketLaunched", message)
	require.NoError(t, err)
	require.NoError(t, controller.processRocketMessage(channel, 1, time.Time{}, decoded))
}

func testStoreMessage(t *testing.T, messageType string, payload string) rocket.Message {
	t.Helper()

	message, err := rocket.DecodeMessage(messageType, json.RawMessage(payload))
	require.NoError(t, err)
	return message
}
