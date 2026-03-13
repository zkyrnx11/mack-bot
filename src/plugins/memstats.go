package plugins

import "runtime/debug"

// SetAggressiveGC configures the Go runtime for minimal memory usage.
func SetAggressiveGC() {
	debug.SetGCPercent(50)
	debug.SetMemoryLimit(10 * 1024 * 1024) // 10MB soft limit
}
