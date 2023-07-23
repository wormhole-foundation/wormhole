# Changelog

## 0.0.4

### Changed

- Add Github Actions CI pipeline for Worm CLI `build` & `test` processes (`.github/workflows/worm-cli.yml`)
- Add unit test to verify all worm CLI commands are in sync with documentation (`cmds.test.ts`)
- Add unit tests to check functionality on the following commands:
  - `worm submit` (using msw)
  - `worm parse`
  - `worm recover`
  - `worm info chain-id`
  - `worm info rpc`
  - `worm info contract`

## 0.0.3

### Changed

Build a minified bundle

## 0.0.2

### Changed

Make `worm` directory agnostic

## 0.0.1

Initial release
