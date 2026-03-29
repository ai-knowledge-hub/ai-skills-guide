# Content Repurposing Plugin

Portable bundle for content reuse workflows across channels and formats.

## Includes
- Content adaptation and creative rules skills
- Output evaluation support
- Example review-before-publish hook

## Install Behavior

This plugin is a packaging layer.

On install:

- the plugin package itself is installed under `plugins/...`
- bundled skills are installed into the runtime `skills/...` directory
- packaged hooks remain inside this plugin's `hooks/` directory

Bundled skills do not live inside the plugin directory itself. That keeps the
plugin as a reusable composition package instead of a duplicated copy of its
dependencies.

## Use Case
Install when teams want a repeatable plugin for turning one source asset into multiple publish-ready variants with review controls.
