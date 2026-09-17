package manifestschemas

import "embed"

// ManifestFiles contains the canonical module schemas used by both CI and the
// runtime package admission boundary.
//
//go:embed skill.schema.json agent.schema.json tool.schema.json plugin.schema.json runtime-package.schema.json smoke-evidence.schema.json doctor-evidence.schema.json auth-driver.schema.json
var ManifestFiles embed.FS
