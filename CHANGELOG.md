# Changelog

All notable changes to this project are documented in this file.

The project follows semantic versioning for release identifiers. Dates use ISO 8601.

## Unreleased

### Added

- Protocol-matrix regression tests for all request, non-stream response, and SSE directions.
- Cancellation propagation through protocol stream conversion.
- OpenAI Chat streaming tool-call conversion to Responses and Anthropic events.
- Frontend unit-test and lint quality gates.
- OpenAPI management-interface baseline and automated Windows packaging checks.
- Apache License 2.0 project licensing.

### Changed

- Chat-to-Responses requests now preserve the caller's stream preference.
- Usage extraction now selects fields by protocol and avoids mixed-field double counting.
- Frontend API access is split into resource-focused modules while retaining the existing import surface.
- Bridge and desktop builds use one repository version source.

## 2.0.1 - 2026-05-22

### Added

- Version injection for release builds.
- Local desktop packaging with `icoo_desktop.exe` and `bridge.exe`.

### Fixed

- Local gateway access controls, provider secret exposure, provider health checks, traffic queue accounting, request cancellation classification, bounded request bodies, and stream response-header timeouts.
