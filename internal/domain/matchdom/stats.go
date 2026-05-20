package matchdom

import "time"

// CleanupDelay is how long after match_ended the engine waits before
// publishing the final snapshot and tearing down the match loop.
const CleanupDelay = 5 * time.Minute
