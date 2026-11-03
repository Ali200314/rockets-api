 # Rockets API

A small Go service that consumes rocket state-change messages and exposes the current state of each rocket over HTTP.

## Requirements

- Go 1.27 or newer
- The `rockets` test-program executable, if available

## Run locally

Start the API on port `8088`:

```bash
go run ./cmd/server
```

Run the automated tests:

```bash
go test ./...
```

The service listens on `http://localhost:8088`.

## API

### Post a message

`POST /messages`

The request body must contain message metadata and a typed message payload:

```json
{
	"metadata": {
		"channel": "rocket-a",
		"messageNumber": 1,
		"messageTime": "2022-02-02T19:39:05.86337+01:00",
		"messageType": "RocketLaunched"
	},
	"message": {
		"type": "Falcon-9",
		"launchSpeed": 500,
		"mission": "ARTEMIS"
	}
}
```

The endpoint returns `204 No Content` when the message is accepted. Invalid JSON, metadata, unsupported message types, and invalid message payloads return `400 Bad Request` with an explanatory error message.

Supported message types and payload examples:

| Message type | Payload |
| --- | --- |
| `RocketLaunched` | `{"type":"Falcon-9","launchSpeed":500,"mission":"ARTEMIS"}` |
| `RocketSpeedIncreased` | `{"by":3000}` |
| `RocketSpeedDecreased` | `{"by":2500}` |
| `RocketExploded` | `{"reason":"PRESSURE_VESSEL_FAILURE"}` |
| `RocketMissionChanged` | `{"newMission":"SHUTTLE_MIR"}` |

### Get one rocket

`GET /rockets/{channel}`

Example:

```bash
curl http://localhost:8088/rockets/rocket-a
```

Successful responses contain a snapshot:

```json
{
	"channel": "rocket-a",
	"type": "Falcon-9",
	"speed": 500,
	"mission": "ARTEMIS",
	"updatedAt": "2022-02-02T19:39:05.86337+01:00"
}
```

The endpoint returns `404 Not Found` with an empty body when the rocket does not exist or has not processed its first message.

### Get all rockets

`GET /rockets`

```bash
curl http://localhost:8088/rockets
```

The response is a JSON array of snapshots. Rockets are sorted by descending speed; rockets with equal speed are sorted by channel in ascending lexicographic order. An empty store returns `[]`.

## Behavior and design

- Duplicate message numbers that have already been processed are ignored.
- A pending message that cannot be applied in the rocket's current state returns an error and is discarded. To retry it, post the same message number again with valid values.
- Rocket state is protected by a read/write mutex, allowing concurrent reads while serializing updates.
- A rocket is not returned by the query endpoints until its first message has been processed.
- State is held in memory and is lost when the process stops.

## Test-program integration

With the API running, point the challenge executable at the message endpoint:

```bash
./rockets launch "http://localhost:8088/messages" --message-delay=500ms --concurrency-level=1
```

