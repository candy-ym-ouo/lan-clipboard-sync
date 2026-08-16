# Reproduction

Baseline: `base_bug_005`

Run:

```sh
go test -count=20 ./internal/transport
```

Actual: `TestAnnouncementUsesAdvertisedHTTPPort` fails because the discovered receiver address uses the discovery port.

Expected: the receiver address uses the HTTP port advertised by the remote device.
