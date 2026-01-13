# Changelog

All notable changes to the Specular CLI are documented here.

## Unreleased
- Removed the legacy `specular drift`, root-level `plan/build/eval`, and `specular debug status` commands so only the canonical lifecycle verbs remain (`spec new/plan create/build run/eval drift/status`).
- Introduced `docs/APPROVAL_BEST_PRACTICES.md`, enforced `specular approve <resource> --message ...`, and updated docs/tutorials to standardize approvals and drift detection.
- Retired the `specular provider health` alias in favor of `specular provider doctor` across the CLI, tests, and docs.
- Added a plugin template checklist and linked it in `docs/plugin-development/README.md` to guide contributors on replacing the TODO placeholders.
- Created Roady features for expanding packaging distribution coverage (Launchpad/Copr/OBS) and plugin template guidance so the remaining TODOs are tracked.
