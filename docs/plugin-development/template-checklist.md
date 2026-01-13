# Plugin Template Checklist

Plugin templates ship with `TODO` comments so that authors know where to plug in logic. Before committing a plugin based on the scaffolds under `internal/plugin/templates`, please do the following:

1. **Update metadata** (`plugin.yaml`): replace placeholder `name`, `description`, and any `TODO` tags with real values, version, and contact info.
2. **Replace logic stubs**: remove every `// TODO implement` (or `# TODO` in shell/python) with the actual provider/formatter/notifier/hook logic. Keep the scaffolds as minimal as possible and document any non-obvious behavior in the file.
3. **Document configuration**: update README or `docs/plugin-development/manifest-reference.md` if the new plugin introduces new config options or environment variables.
4. **Test the plugin**: ensure the plugin can be built (`go test ./...` for Go, `npm test` for Node, etc.) and is runnable with a sample manifest.
5. **Link to the checklist**: if you add new onboarding material, mention this checklist so future authors know to drop the TODO comments.

Once those steps are done, the plugin should be ready for contribution without any `TODO` placeholders.
