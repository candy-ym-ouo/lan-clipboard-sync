# Reproduction

Baseline: `base_bug_002`

Run:

```sh
go test -count=20 ./internal/core
```

Actual: `TestDevicesRetainRecentDevice` fails because a recently announced device disappears from the list.

Expected: a device remains available until its configured discovery lifetime has elapsed.
