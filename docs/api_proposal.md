# Controller Interface Proposal

## Requirements
- Communicate a schedule of asset activity from the Axle cloud to the on-site controller
- Communicate telemetry about the site from the on-site controller to the Axle cloud (updating every minute or so)

## Websocket

The controller will make an outbound websocket connection directly to the Axle servers over TLS. For example, to a URI like `wss://axle.energy/controllers/<controller-uuid>`

Websockets have the following advantages:
- Two-way comms
- Can use a single outbound connection from the edge device, helping navigate industrial firewalls etc
- Should support existing auth systems (e.g. HTTP/token-based systems), or mutual TLS
- It's probably the simplest option


## Messages

If we are sending telemetry every minute or so, then JSON-based messages should be fast enough.

The proposed message structure has a `header` section detailing the:
- `topic`, which defines the type of message and resource, similar to a HTTP URL, or an MQTT topic.
- `created_at`, which is the time that the message was created
- `version` number
- `id`, which is a UUID for the message

The `payload` section contains the actual data and is structured differently for each message. Telemetry payloads are quite 'flat' so they are easier to put straight into a database.

### Controller Config message
This message is sent once from the controller to the cloud when the websocket connection is first established, and informs the cloud about any BESS that are configured at the site.
```JSON
{
  "header": {
    "topic": "/controller/<controller-uuid>/config",
    "version": "1.0.0",
    "created_at": "2025-01-06T15:00:00+00:00",
    "id": "<message-uuid>"
  }
  "payload": {
     "bess": {
        "<bess-uuid>": {
            "nameplate_energy": 1620,
            "nameplate_power": 565,
            other...
        }
     }
  }
}
```


### Schedule message
This message is sent from the cloud to the controller and defines the plan for the asset over the next 72 hours or so. This message will be re-sent every time the schedule changes- only the latest schedule is used.
It is useful to have at least 24hours worth of scheduling stored on the controller so that even if there is a server or network outage the asset can deliver it's pre-defined plan.

There are two types of scheduled activity:
- `active_power` which defines a ramped profile of time-power points. If multiple profiles are specified then these will be added together linearly for delivery.
- `frequency_response` which defines the ramped profile of frequency response 'envelopes', where the actual power delivered is determined in real-time by the associated frequency function. Again, these will be added together linearly for delivery.
```JSON
{
  "header": {
   "topic": "/bess/<bess-uuid>/schedule",
   "created_at": "2025-01-06T15:00:00+00:00",
   "version": "1.0.0",
   "id": "<message-uuid>"
  },
  "payload": {
    "active_power": [
        {
            "profile": [
                {
                    "time": "2025-02-06T13:00:00+00:00",
                    "power": 0.0
                },
                {
                    "time": "2025-02-06T13:01:00+00:00",
                    "power": 1000.0
                },
                {
                    "time": "2025-02-06T13:59:00+00:00",
                    "power": 1000.0
                },
                {
                    "time": "2025-02-06T14:00:00+00:00",
                    "power": 0.0
                }
            ],
            "avoid_deviation": false,
        }
    ],
     "frequency_response": {
        "schedules": [
            {
                "function": "dr_low",
                "envelope_profile": [
                    {
                        "time": "2025-02-06T13:00:00+00:00",
                        "power": 0.0
                    },
                    {
                        "time": "2025-02-06T13:01:00+00:00",
                        "power": 1000.0
                    },
                    {
                        "time": "2025-02-06T13:59:00+00:00",
                        "power": 1000.0
                    },
                    {
                        "time": "2025-02-06T14:00:00+00:00",
                        "power": 0.0
                    }
                ]
            }
        },
        "functions": {
            "dr_low": [
                {
                    "frequency": 0,
                    "power_pct": 100
                },
                {
                    "frequency": 48.8,
                    "power_pct": 100
                },
                {
                    "frequency": 49.985,
                    "power_pct": 0
                }
            ]
        }        
     ]
  }
}
```

Note: Cepro does NIV-chasing in real-time, and by default this API permits deviation from the planned active power schedule for NIV chasing. However, if there is a circumstance where this should be avoided, then the  `avoid_deviation` can be set to `true`. This may be useful for situations like Capacity Market activation etc.

### Meter reading message
This message is sent from the controller to the cloud, and delivers real-time info from an on-site meter
```JSON
{
  "header": {
    "topic": "/meter/<meter-uuid>/reading",
    "created_at": "2025-01-06T15:00:00+00:00",
    "version": "1.0.0",
    "id": "<message-uuid>"
  }
  "payload": {
     "time": "2025-02-06T12:01:00+00:00",
     "frequency": 50.01,
     "voltage_line_average": 415.1,
     "current_phase_average": ...,
     "power_total_active": ...,
     "power_total_reactive": ...,
     "power_total_apparent": ...,
     "energy_imported_active": ...,
     others...
  }
}
```



### BESS reading message
This message is sent from the controller to the cloud, and delivers real-time info about an on-site BESS.
```JSON
{
  "header": {
   "topic": "/bess/<bess-uuid>/reading",
   "created_at": "2025-01-06T15:00:00+00:00",
   "version": "1.0.0",
   "id": "<message-uuid>"
  }
  "payload": {
     "time": "2025-02-06T12:01:00+00:00",
     "soe": 1023.3, (state of energy)
     "target_power": 0.0 (the power that the bess is trying to deliver)
  }
}
```



## Sequencing and Reliability
Once the websocket connection is established the controller will first send the `/controller/<controller-uuid>/config` message to the server. The server must then send a schedule for each BESS defined in the configuration. This means that state can be properly syncronised after any server or controller reboots or crashes. Thereafter, the server only needs to send schedules messages when there are updates.

Once the connection is established, telemetry messages are sent regularly for each on-site device of interest (e.g. once every minute for each device). In situations where the connection fails then the telemetry is stored locally on the controller disk and uploaded in a rate-limited manner once the connection is re-established (e.g. 10 readings per minute for each device until the backlog is cleared).

There remains a small risk of lost telemetry in cases where the controller has sent telemetry messages onto the websocket TCP connection, but the server crashes before properly recieving or processing these messages. This could be addressed with acknowledgement messages, but at this stage it doesn't seem worth the added complexity.

### Units / formats

- Time: ISO 8601 formatted strings with timezone info
- Energy: kWh
- Power: kW
- Voltage: volts
- Current: amps
