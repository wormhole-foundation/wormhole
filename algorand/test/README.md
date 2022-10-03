# Adversarial and extra testing
README for the adversarial and extra tests added by Coinspect during the audit.

## Requirements
Some of the new tests require `pytest`, which was already declared as a
dependency.

Two new randomized test use the approach that previous tests where using
and are simply in `test.py`. See `Usage`.

## Usage

### `test.py`
`test.py` now accept two additional flags: 

`--loops` defines how many times the tests should run

`--bigset` defines if it should use the big guardian set

For example, to run the tests with 10 loops and a big validator set.
Note the `--loop` flag will not affect previous tests.

```
python test.py --loops 10 --bigset
```

### Running Pytest tests
Simple do:
```
$ pytest
```

## Notes
Shared fixtures are declared in `conftest.py`
