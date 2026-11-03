package rocketsapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"rockets-api/rocket"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewServer(t *testing.T) {
	ctrl := NewRocketStore()

	server := NewServer(ctrl)

	require.NotNil(t, server)
	assert.Same(t, ctrl, server.store)
}

func TestServerHandler(t *testing.T) {
	server := NewServer(NewRocketStore())
	handler := server.Handler()

	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{name: "create message", method: http.MethodPost, path: "/messages", want: http.StatusBadRequest},
		{name: "get rockets", method: http.MethodGet, path: "/rockets", want: http.StatusOK},
		{name: "get rocket", method: http.MethodGet, path: "/rockets/moon", want: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, nil)
			res := httptest.NewRecorder()

			handler.ServeHTTP(res, req)

			assert.Equal(t, test.want, res.Code)
		})
	}
}

func TestServerIntegration(t *testing.T) {
	t.Run("processes a launch and returns the rocket", func(t *testing.T) {
		// Given: a real server with a real rocket store.
		httpServer := httptest.NewServer(NewServer(NewRocketStore()).Handler())
		defer httpServer.Close()

		// When: a rocket launch is posted.
		body := bytes.NewBufferString(`{"metadata":{"channel":"rocket-a","messageType":"RocketLaunched","messageNumber":1,"messageTime":"2022-02-02T19:39:05.86337+01:00"},"message":{"type":"Falcon-9","launchSpeed":500,"mission":"ARTEMIS"}}`)
		postResponse, err := http.Post(httpServer.URL+"/messages", "application/json", body)
		require.NoError(t, err)
		defer func() { require.NoError(t, postResponse.Body.Close()) }()
		assert.Equal(t, http.StatusNoContent, postResponse.StatusCode)

		// Then: the processed rocket is returned by the API.
		getResponse, err := http.Get(httpServer.URL + "/rockets/rocket-a")
		require.NoError(t, err)
		defer func() { require.NoError(t, getResponse.Body.Close()) }()
		responseBody, err := io.ReadAll(getResponse.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, getResponse.StatusCode)
		assert.JSONEq(t, `{"channel":"rocket-a","type":"Falcon-9","speed":500,"mission":"ARTEMIS","updatedAt":"2022-02-02T19:39:05.86337+01:00"}`, string(responseBody))
	})

	t.Run("returns an error when applying a pending message fails", func(t *testing.T) {
		// Given: a real server with a launched rocket and a pending mission change.
		httpServer := httptest.NewServer(NewServer(NewRocketStore()).Handler())
		defer httpServer.Close()

		launchBody := bytes.NewBufferString(`{"metadata":{"channel":"rocket-a","messageType":"RocketLaunched","messageNumber":1,"messageTime":"2022-02-02T19:39:05.86337+01:00"},"message":{"type":"Falcon-9","launchSpeed":500,"mission":"ARTEMIS"}}`)
		launchResponse, err := http.Post(httpServer.URL+"/messages", "application/json", launchBody)
		require.NoError(t, err)
		require.NoError(t, launchResponse.Body.Close())
		require.Equal(t, http.StatusNoContent, launchResponse.StatusCode)

		pendingBody := bytes.NewBufferString(`{"metadata":{"channel":"rocket-a","messageType":"RocketMissionChanged","messageNumber":3,"messageTime":"2022-02-02T19:39:07.86337+01:00"},"message":{"newMission":"SHUTTLE-MIR"}}`)
		pendingResponse, err := http.Post(httpServer.URL+"/messages", "application/json", pendingBody)
		require.NoError(t, err)
		require.NoError(t, pendingResponse.Body.Close())
		require.Equal(t, http.StatusNoContent, pendingResponse.StatusCode)

		// When message 2 explodes the rocket, applying pending message 3 fails.
		explosionBody := bytes.NewBufferString(`{"metadata":{"channel":"rocket-a","messageType":"RocketExploded","messageNumber":2,"messageTime":"2022-02-02T19:39:06.86337+01:00"},"message":{"reason":"ENGINE_FAILURE"}}`)
		explosionResponse, err := http.Post(httpServer.URL+"/messages", "application/json", explosionBody)
		require.NoError(t, err)
		defer func() { require.NoError(t, explosionResponse.Body.Close()) }()

		// Then the request reports the pending message application error.
		assert.Equal(t, http.StatusBadRequest, explosionResponse.StatusCode)
		assert.Contains(t, responseBodyString(t, explosionResponse), "unable to apply message 3")
	})
}

func responseBodyString(t *testing.T, response *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return string(body)
}

func mustMarshalJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return string(data)
}

func TestServerHandleRocketMessage(t *testing.T) {
	t.Run("success when valid message", func(t *testing.T) {
		// Given: server
		mockRocketStore := NewMockRocketStore(gomock.NewController(t))
		server := NewServer(mockRocketStore)

		// Given: store processes rocket message with expected values
		messageTime, err := time.Parse(time.RFC3339Nano, "2022-02-02T19:39:05.86337+01:00")
		require.NoError(t, err)
		mockRocketStore.EXPECT().processRocketMessage("rocket-a", 1, messageTime, gomock.Any())

		// When: a valid message is posted.
		body := bytes.NewBufferString(`{"metadata":{"channel":"rocket-a","messageType":"RocketLaunched","messageNumber":1,"messageTime":"2022-02-02T19:39:05.86337+01:00"},"message":{"type":"Falcon-9","launchSpeed":500,"mission":"ARTEMIS"}}`)
		request := httptest.NewRequest(http.MethodPost, "/messages", body)
		response := httptest.NewRecorder()
		server.handleRocketMessage(response, request)

		// Then: success
		assert.Equal(t, http.StatusNoContent, response.Code)
	})

	t.Run("returns bad request when unmarshalling payload fails", func(t *testing.T) {
		// Given: server
		server := NewServer(NewMockRocketStore(gomock.NewController(t)))

		// When: malformed message is posted
		request := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewBufferString("{"))
		response := httptest.NewRecorder()
		server.handleRocketMessage(response, request)

		// Then: bad request
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "invalid JSON")
	})

	t.Run("returns bad request when metadata is empty", func(t *testing.T) {
		// Given: server
		server := NewServer(NewMockRocketStore(gomock.NewController(t)))

		// When: the message is posted.
		body := bytes.NewBufferString(`{"metadata":{},"message":{}}`)
		request := httptest.NewRequest(http.MethodPost, "/messages", body)
		response := httptest.NewRecorder()
		server.handleRocketMessage(response, request)

		// Then: bad request
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "channel is required")
	})

	t.Run("returns bad request for unsupported message type", func(t *testing.T) {
		// Given: server
		server := NewServer(NewMockRocketStore(gomock.NewController(t)))

		// When: a message with an unsupported type is posted.
		body := bytes.NewBufferString(`{"metadata":{"channel":"rocket-a","messageType":"Unknown","messageNumber":1,"messageTime":"2022-02-02T19:39:05.86337+01:00"},"message":{}}`)
		request := httptest.NewRequest(http.MethodPost, "/messages", body)
		response := httptest.NewRecorder()
		server.handleRocketMessage(response, request)

		// Then: the request is rejected.
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "unsupported message type: Unknown")
	})

	t.Run("returns bad request for invalid message", func(t *testing.T) {
		// Given: server
		server := NewServer(NewMockRocketStore(gomock.NewController(t)))

		// When: a message with invalid launch data is posted.
		body := bytes.NewBufferString(`{"metadata":{"channel":"rocket-a","messageType":"RocketLaunched","messageNumber":1,"messageTime":"2022-02-02T19:39:05.86337+01:00"},"message":{"type":"","launchSpeed":500,"mission":"ARTEMIS"}}`)
		request := httptest.NewRequest(http.MethodPost, "/messages", body)
		response := httptest.NewRecorder()
		server.handleRocketMessage(response, request)

		// Then: the request is rejected.
		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Contains(t, response.Body.String(), "type is required")
	})

	t.Run("returns bad request for metadata validation errors", func(t *testing.T) {
		tests := []struct {
			name     string
			metadata metadata
			wantErr  string
		}{
			{
				name:     "missing channel",
				metadata: metadata{MessageType: "RocketLaunched", MessageNumber: 1, MessageTime: time.Now()},
				wantErr:  "channel is required",
			},
			{
				name:     "blank channel",
				metadata: metadata{Channel: " ", MessageType: "RocketLaunched", MessageNumber: 1, MessageTime: time.Now()},
				wantErr:  "channel is required",
			},
			{
				name:     "missing message type",
				metadata: metadata{Channel: "rocket-a", MessageNumber: 1, MessageTime: time.Now()},
				wantErr:  "messageType is required",
			},
			{
				name:     "blank message type",
				metadata: metadata{Channel: "rocket-a", MessageType: "\t", MessageNumber: 1, MessageTime: time.Now()},
				wantErr:  "messageType is required",
			},
			{
				name:     "zero message number",
				metadata: metadata{Channel: "rocket-a", MessageType: "RocketLaunched", MessageTime: time.Now()},
				wantErr:  "messageNumber must be greater than zero",
			},
			{
				name:     "negative message number",
				metadata: metadata{Channel: "rocket-a", MessageType: "RocketLaunched", MessageNumber: -1, MessageTime: time.Now()},
				wantErr:  "messageNumber must be greater than zero",
			},
			{
				name:     "missing message time",
				metadata: metadata{Channel: "rocket-a", MessageType: "RocketLaunched", MessageNumber: 1},
				wantErr:  "messageTime is required",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				server := NewServer(NewMockRocketStore(gomock.NewController(t)))
				body := bytes.NewBufferString(`{"metadata":` + mustMarshalJSON(t, test.metadata) + `,"message":{}}`)
				request := httptest.NewRequest(http.MethodPost, "/messages", body)
				response := httptest.NewRecorder()

				server.handleRocketMessage(response, request)

				assert.Equal(t, http.StatusBadRequest, response.Code)
				assert.Contains(t, response.Body.String(), test.wantErr)
			})
		}
	})
}

func TestServerGetRocket(t *testing.T) {
	t.Run("returns an existing rocket", func(t *testing.T) {
		// Given: server
		mockRocketStore := NewMockRocketStore(gomock.NewController(t))
		server := NewServer(mockRocketStore)

		// Given: store returns existing rocket
		mockRocketStore.EXPECT().getRocket("rocket-a").Return(rocket.Snapshot{Channel: "rocket-a", Speed: 500}, true)

		// When: getRocket is called
		request := httptest.NewRequest(http.MethodGet, "/rockets/rocket-a", nil)
		request.SetPathValue("channel", "rocket-a")
		response := httptest.NewRecorder()
		server.getRocket(response, request)

		// Then: rocket is returned
		assert.Equal(t, http.StatusOK, response.Code)
		assert.JSONEq(t, `{"channel":"rocket-a","type":"","speed":500,"mission":"","updatedAt":"0001-01-01T00:00:00Z"}`, response.Body.String())
	})

	t.Run("returns not found for a missing rocket", func(t *testing.T) {
		// Given: server
		mockRocketStore := NewMockRocketStore(gomock.NewController(t))
		server := NewServer(mockRocketStore)

		// Given: store cannot find the rocket
		mockRocketStore.EXPECT().getRocket("missing").Return(rocket.Snapshot{}, false)

		// When: getRocket is called
		request := httptest.NewRequest(http.MethodGet, "/rockets/missing", nil)
		request.SetPathValue("channel", "missing")
		response := httptest.NewRecorder()
		server.getRocket(response, request)

		// Then: rocket is not found
		assert.Equal(t, http.StatusNotFound, response.Code)
		assert.Empty(t, response.Body.String())
	})
}

func TestServerGetRockets(t *testing.T) {
	t.Run("returns rockets", func(t *testing.T) {
		// Given: server
		mockRocketStore := NewMockRocketStore(gomock.NewController(t))
		server := NewServer(mockRocketStore)

		// Given: store returns rockets
		expectedRocket := rocket.Snapshot{
			Channel: "rocket-a",
			Type:    "Falcon-9",
			Speed:   500,
			Mission: "ARTEMIS",
		}
		mockRocketStore.EXPECT().getRockets().Return([]rocket.Snapshot{expectedRocket})

		// When: getRockets is called
		request := httptest.NewRequest(http.MethodGet, "/rockets", nil)
		response := httptest.NewRecorder()
		server.getRockets(response, request)

		// Then: rockets are returned
		assert.Equal(t, http.StatusOK, response.Code)
		assert.JSONEq(t, `[{
			"channel":"rocket-a",
			"type":"Falcon-9",
			"speed":500,
			"mission":"ARTEMIS",
			"updatedAt":"0001-01-01T00:00:00Z"
		}]`, response.Body.String())
	})
}
