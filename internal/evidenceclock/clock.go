// Package evidenceclock provides the clock for package evidence admission.
// It does not change credential, provider, model, or runtime evidence clocks.
package evidenceclock

import "time"

// Now defaults to the real clock in every production binary. Serial tests may
// replace it with a fixed observation instant and must restore it with cleanup.
// No environment variable or command-line flag can override this clock.
var Now = time.Now
