# Reproduction

Baseline: `base_bug_004`

Run:

```sh
go test -count=20 ./internal/core
```

Actual: `TestHistoryListsNewestMessageFirst` fails because an older history entry is returned before a newer entry.

Expected: history is displayed with the most recently recorded message first.
