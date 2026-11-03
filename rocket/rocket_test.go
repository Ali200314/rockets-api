package rocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("creates a rocket with initial state", func(t *testing.T) {
		// When a rocket is created.
		got := New("rocket-1")

		// Then it has the expected initial state.
		expected := Rocket{
			channel: "rocket-1",
			next:    1,
			pending: map[int]pending{},
		}
		assert.Equal(t, expected, got)
	})
}

func TestRocketProcessMessage(t *testing.T) {
	t.Run("processes a rocket launch", func(t *testing.T) {
		// Given a new rocket and a launch message.
		rocket := New("rocket-1")
		msg := &launched{Type: "Falcon-9", LaunchSpeed: 500, Mission: "ARTEMIS"}

		// When the launch message is processed.
		require.NoError(t, rocket.ProcessMessage(1, time.Time{}, msg))

		// Then the rocket contains the launch state.
		expected := Rocket{
			channel:    "rocket-1",
			rocketType: "Falcon-9",
			speed:      500,
			mission:    "ARTEMIS",
			next:       2,
			pending:    map[int]pending{},
		}
		assert.Equal(t, expected, rocket)
	})

	t.Run("queues a later message without applying it", func(t *testing.T) {
		// Given a new rocket and a message received after the next expected number.
		rocket := New("rocket-1")
		msg := &speedIncreased{By: 300}

		// When the later message is processed.
		require.NoError(t, rocket.ProcessMessage(2, time.Time{}, msg))

		// Then only the constructor state is present and the message is pending.
		expected := Rocket{
			channel: "rocket-1",
			next:    1,
			pending: map[int]pending{
				2: {
					message: msg,
				},
			},
		}
		assert.Equal(t, expected, rocket)
	})

	t.Run("error when speed decreased by more than current speed", func(t *testing.T) {
		// Given an active rocket and a decrease that would make its speed negative.
		rocket := New("rocket-1")
		launch := &launched{Type: "Falcon-9", LaunchSpeed: 500, Mission: "ARTEMIS"}
		require.NoError(t, rocket.ProcessMessage(1, time.Time{}, launch))

		// When the invalid state transition is processed.
		err := rocket.ProcessMessage(2, time.Time{}, &speedDecreased{By: 600})

		// Then the error identifies the failed message and the failed message is removed.
		require.EqualError(t, err, "unable to apply message 2: speed cannot become negative")
		assert.Equal(t, 2, rocket.Next())
		assert.Equal(t, float32(500), rocket.speed)
		assert.Empty(t, rocket.pending)

		// When the same message number is posted with valid values.
		require.NoError(t, rocket.ProcessMessage(2, time.Time{}, &speedDecreased{By: 100}))

		// Then processing continues from the replaced message.
		assert.Equal(t, 3, rocket.Next())
		assert.Equal(t, float32(400), rocket.speed)
		assert.Empty(t, rocket.pending)
	})

	t.Run("processes the previous message before a pending message fails", func(t *testing.T) {
		// Given a pending mission change and a later explosion message.
		rocket := New("rocket-1")
		require.NoError(t, rocket.ProcessMessage(3, time.Time{}, &missionChanged{NewMission: "SHUTTLE-MIR"}))
		require.NoError(t, rocket.ProcessMessage(1, time.Time{}, &launched{Type: "Falcon-9", LaunchSpeed: 500, Mission: "ARTEMIS"}))

		// When message 2 is applied, message 3 is processed and fails.
		err := rocket.ProcessMessage(2, time.Time{}, &exploded{Reason: "ENGINE_FAILURE"})

		// Then message 2 was applied successfully before message 3 failed and was removed.
		require.EqualError(t, err, "unable to apply message 3: rocket must be active to change mission")
		assert.Equal(t, "ENGINE_FAILURE", rocket.explodeReason)
		assert.Equal(t, 3, rocket.Next())
		assert.Empty(t, rocket.pending)
	})

	t.Run("processes state changes in message order", func(t *testing.T) {
		// Given a launched rocket.
		rocket := New("rocket-1")
		launchTime := time.Date(2022, time.February, 2, 19, 39, 5, 0, time.UTC)
		speedIncreaseTime := launchTime.Add(time.Second)
		speedDecreaseTime := launchTime.Add(2 * time.Second)
		missionChangeTime := launchTime.Add(3 * time.Second)
		explosionTime := launchTime.Add(4 * time.Second)
		launch := &launched{Type: "Falcon-9", LaunchSpeed: 500, Mission: "ARTEMIS"}
		require.NoError(t, rocket.ProcessMessage(1, launchTime, launch))

		// When speed, mission, and explosion messages are processed.
		require.NoError(t, rocket.ProcessMessage(2, speedIncreaseTime, &speedIncreased{By: 300}))
		require.NoError(t, rocket.ProcessMessage(3, speedDecreaseTime, &speedDecreased{By: 100}))
		require.NoError(t, rocket.ProcessMessage(4, missionChangeTime, &missionChanged{NewMission: "SHUTTLE-MIR"}))
		require.NoError(t, rocket.ProcessMessage(5, explosionTime, &exploded{Reason: "PRESSURE_VESSEL_FAILURE"}))

		// Then all state changes are reflected in the snapshot.
		expected := Rocket{
			channel:       "rocket-1",
			rocketType:    "Falcon-9",
			speed:         0,
			mission:       "SHUTTLE-MIR",
			explodeReason: "PRESSURE_VESSEL_FAILURE",
			updatedAt:     explosionTime,
			next:          6,
			pending:       map[int]pending{},
		}
		assert.Equal(t, expected, rocket)
	})

	t.Run("returns an error when processing a message after explosion", func(t *testing.T) {
		// Given a rocket that has exploded.
		rocket := New("rocket-1")
		launch := &launched{Type: "Falcon-9", LaunchSpeed: 500, Mission: "ARTEMIS"}
		require.NoError(t, rocket.ProcessMessage(1, time.Time{}, launch))
		explosion := &exploded{Reason: "ENGINE_FAILURE"}
		require.NoError(t, rocket.ProcessMessage(2, time.Time{}, explosion))

		// When another message is processed.
		err := rocket.ProcessMessage(3, time.Time{}, &missionChanged{NewMission: "SHUTTLE-MIR"})

		// Then processing returns early without changing the rocket.
		require.EqualError(t, err, "cannot process message after rocket exploded")
		assert.Equal(t, 3, rocket.Next())
		assert.Equal(t, "ENGINE_FAILURE", rocket.explodeReason)
		assert.Empty(t, rocket.pending)
	})

	t.Run("processes messages that arrive out of order", func(t *testing.T) {
		// Given a new rocket and a speed change received before launch.
		rocket := New("rocket-1")
		launch := &launched{Type: "Falcon-9", LaunchSpeed: 500, Mission: "ARTEMIS"}

		// When the messages are processed out of order.
		require.NoError(t, rocket.ProcessMessage(2, time.Time{}, &speedIncreased{By: 300}))
		require.NoError(t, rocket.ProcessMessage(1, time.Time{}, launch))

		// Then the pending message is applied after the launch.
		expected := Rocket{
			channel:    "rocket-1",
			rocketType: "Falcon-9",
			speed:      800,
			mission:    "ARTEMIS",
			next:       3,
			pending:    map[int]pending{},
		}
		assert.Equal(t, expected, rocket)
	})

	t.Run("ignores duplicate messages", func(t *testing.T) {
		// Given a rocket that has processed a launch.
		rocket := New("rocket-1")
		msg := &launched{Type: "Falcon-9", LaunchSpeed: 500, Mission: "ARTEMIS"}
		require.NoError(t, rocket.ProcessMessage(1, time.Time{}, msg))

		// When the same message is processed again.
		require.NoError(t, rocket.ProcessMessage(1, time.Time{}, msg))

		// Then the rocket state is unchanged.
		expected := Rocket{
			channel:    "rocket-1",
			rocketType: "Falcon-9",
			speed:      500,
			mission:    "ARTEMIS",
			next:       2,
			pending:    map[int]pending{},
		}
		assert.Equal(t, expected, rocket)
	})

}
