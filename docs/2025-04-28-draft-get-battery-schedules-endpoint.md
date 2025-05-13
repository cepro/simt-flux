# Draft of GET battery schedule endpoint

Python typing of the response of the GET battery schedule endpoint:

```py
from pydantic import BaseModel, validator
from datetime import datetime

class BatteryScheduleStep(BaseModel):
    start_timestamp: datetime
    end_timestamp: datetime
    step_type: Literal["charge", "discharge"]
    rate_pct: float

    allow_deviation: bool # whether the battery is allowed to deviate from the schedule

    @validator("start_timestamp", "end_timestamp")
    def is_tz_aware(cls, v):
        if v.tzinfo is None:
            raise ValueError("Timestamp must be timezone aware")
        return v

    @validator("rate_pct")
    def check_rate_pct(cls, v):
        if not (0 <= v <= 100):
            raise ValueError("Charge/discharge rate must be between 0 and 100")
        return v


class BatteryScheduleResponse(BaseModel):
    result: list[BatteryScheduleStep]
```

Example response:

```py
import json

example_response = """
{
    "result": [
        {
            "start_timestamp": "2025-12-01T00:00:00+00:00",
            "end_timestamp": "2025-12-01T03:00:00+00:00",
            "step_type": "charge",
            "rate_pct": 100,
            "allow_deviation": false
        },
        {
            "start_timestamp": "2025-12-02T17:00:00+00:00",
            "end_timestamp": "2025-12-02T19:00:00+00:00",
            "step_type": "charge",
            "rate_pct": 0,
            "allow_deviation": false
        },
        {
            "start_timestamp": "2025-12-01T00:00:00+00:00",
            "end_timestamp": "2025-12-01T03:00:00+00:00",
            "step_type": "charge",
            "rate_pct": 0,
            "allow_deviation": true
        },
        {
            "start_timestamp": "2025-12-02T17:00:00+00:00",
            "end_timestamp": "2025-12-02T19:00:00+00:00",
            "step_type": "discharge",
            "rate_pct": 100,
            "allow_deviation": true
        }
    ]
}
"""

response_example = BatteryScheduleResponse.parse_obj(json.loads(example_response))
```

In this example schedule, we have two trading windows, between 00:00 and 03:00, and between 17:00 and 19:00. Let's assume the 1st December is a day where we're building a baseline, while the 2nd December is a day where we're trading. The schedule is as follows:

1. On the baseline day, we really don't want the battery to deviate, as this baseline will be used for up to 60 days. This schedule will tell the battery to charge the battery at 100% between 00:00 and 03:00, and make sure we don't perform the usual export at 17:00-19:00.

2. On the trading day, we want to avoid charging during the 00:00-03:00 window, and encourage a discharge between 17:00 and 19:00. We also want to allow deviation, as any deviation from the schedule will only impact this single day.
