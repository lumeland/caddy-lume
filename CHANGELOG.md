# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/)
and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.1.6] - 2026-01-02
### Added
- More process logs

### Changed
- Added `LUME_LOGS=WARNING` env variable to reduce the number of logs

### Fixed
- Fixed SIGSEGV after closing the process
- Ensure all process resources are released after killing it.

## [0.1.5] - 2025-12-29
### Added
- `idle_timeout` option.

### Fixed
- Improved the idle timeout watcher

[0.1.6]: https://github.com/lumeland/caddy-lume/compare/v0.1.5...v0.1.6
[0.1.5]: https://github.com/lumeland/caddy-lume/releases/tag/v0.1.5
