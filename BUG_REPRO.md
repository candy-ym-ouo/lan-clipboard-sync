# Reproduction

Baseline: `base_bug_003`

Run:

```sh
go test -count=20 ./internal/core
```

Actual: `TestMessageAcceptsMaximumTextSize` fails because text at the documented size limit is rejected.

Expected: text equal to the maximum accepted size is valid; only larger text is rejected.
