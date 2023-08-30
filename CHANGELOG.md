# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- Upcoming changes...

## [1.3.0] - 2023-08-30
### Added
- Added Open Telemetry (OTEL) support
  - Requires an OTEL Collector running gRPC

## [1.2.3] - 2023-08-25
### Added
- Added HEAD support for the following endpoints:
  - `/`, `/api/`, `/api/health`, `/api/health-check`

## [1.2.2] - 2023-08-24
### Added
- Added support for `/api/kb/details` endpoint

## [1.2.1] - 2023-08-09
### Added
- Added option to enable/disable HPSM processing (`HPSMEnabled`)

## [1.2.0] - 2023-05-18
### Added
- Added support for password protected TLS Key Files (`Password`)

## [1.1.0] - 2023-05-08
### Added
- Added log output configuration options (`OutputPaths`)
  - Default `stderr`
- Added custom response URL config option (`ScanningURL`)
  - Default `https://osskb.org/api`

## [1.0.0] - 2023-01-23
### Added
- Added GitHub action to build/unit test on push/pr
- Added supporting README files

## [0.7.0] - 2023-01-22
### Added
- Added unit tests
- Added integration tests
- Added End-to-end container testing

## [0.5.0] - 2023-01-07
### Added
- Added attribution endpoint
- Added license details endpoint

## [0.4.1] - 2023-01-04
### Added
- Added API health
- Added API metrics
### Fixed
- Fixed issue with scanning form parameters

## [0.0.1] - 2022-12-15
### Added
- Added scanning endpoint
- Added file contents endpoint
- Added threaded scanning workers

[0.0.1]: https://github.com/scanoss/api.go/compare/v0.0.0...v0.0.1
[0.4.1]: https://github.com/scanoss/api.go/compare/v0.0.1...v0.4.1
[0.5.0]: https://github.com/scanoss/api.go/compare/v0.4.1...v0.5.0
[0.7.0]: https://github.com/scanoss/api.go/compare/v0.5.0...v0.7.0
[1.0.0]: https://github.com/scanoss/api.go/compare/v0.7.0...v1.0.0
[1.1.0]: https://github.com/scanoss/api.go/compare/v1.0.0...v1.1.0
[1.2.0]: https://github.com/scanoss/api.go/compare/v1.1.0...v1.2.0
[1.2.1]: https://github.com/scanoss/api.go/compare/v1.2.0...v1.2.1
[1.2.2]: https://github.com/scanoss/api.go/compare/v1.2.1...v1.2.2
[1.2.3]: https://github.com/scanoss/api.go/compare/v1.2.2...v1.2.3
[1.3.0]: https://github.com/scanoss/api.go/compare/v1.2.3...v1.3.0
