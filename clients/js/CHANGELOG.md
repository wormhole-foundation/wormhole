# Changelog

## 0.0.4

### Changed

- Add unit tests for all worm CLI commands (following `worm <command> --help` pattern)
- Add unit tests to check functionality on the following commands:
  - `worm parse`
  - `worm recover`
  - `worm info chain-id`
  - `worm info rpc`
  - `worm info contract`
- Generate HTML tests report for every `npm run test` execution (`client/js/html-report/worm-cli-tests-report.html`)
- Github Actions CI pipeline for Worm CLI `build` & `test` processes (`.github/workflows/worm-cli.yml`)

## 0.0.3

### Changed

Build a minified bundle

## 0.0.2

### Changed

Make `worm` directory agnostic

## 0.0.1

Initial release
