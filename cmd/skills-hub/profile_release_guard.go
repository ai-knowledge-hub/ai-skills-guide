//go:build installer_release && runtime_readiness

package main

// A public installer release must never include the deferred development profile.
var _ = installerReleaseCannotIncludeRuntimeReadiness
