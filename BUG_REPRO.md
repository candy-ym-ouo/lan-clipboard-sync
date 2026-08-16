# Reproduction

Baseline: `base_bug_001`

Run:

```sh
go test -count=20 ./internal/core
```

Actual: `TestSignature` fails because a valid signature is not accepted and an invalid key does not remain rejected.

Expected: valid signatures are accepted and signatures made with another key are rejected.
