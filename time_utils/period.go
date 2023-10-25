package timeutils

import "time"

// Period represents an absolute period between two instances in time, e.g. "2023/10/19 16:00:00 to 2023/10/19 18:00:00".
type Period struct {
	Start time.Time
	End   time.Time
}
