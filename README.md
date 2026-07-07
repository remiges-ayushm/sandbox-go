# sandbox-go

Go port of the [ion-sandbox-app](../ion-sandbox-app) TypeScript sandbox for Beckn v2.0.0 protocol
specifications, available at https://github.com/beckn/protocol-specifications-new

## Running locally

```bash
go run .
```

By default the server listens on port `3000` (override with `PORT`). Copy `.env.example` to `.env`
to configure `PORT`, `PERSONA`, `BPP_CALLBACK_ENDPOINT`, and `RESPONSE_FIXTURES_BASE_PATH`.

## Response fixtures

Same fixture resolution behavior as the TypeScript app. By default, the sandbox reads canned
responses from the legacy domain tree:

```text
webhook/jsons/<domain>/response/<action>.json
```

It can also read structured response fixtures that match the ION payload layout:

```text
webhook/responses/<sector>/<pattern>/<crc>/<action>.json
webhook/responses/<sector>/<pattern>/<action>.json
```

Set `RESPONSE_FIXTURES_BASE_PATH` to use another fixture root, for example `../ion-sandbox/payloads`
in local development. The resolver uses request metadata in this order:

- Headers: `x-ion-sector`, `x-ion-pattern`, `x-ion-crc`
- Body metadata: `_meta.sector`, `_meta.pattern`, `_meta.crc`
- Context: `context.domain` for sector
- Best effort CRC hints from known `trc-*` codes or CRC names in the body

If no structured fixture matches, the sandbox falls back to the legacy domain tree. `_meta` is used
only for fixture lookup and is not sent in callbacks.

These fixture directories (`webhook/jsons/`, `webhook/responses/`) are not tracked in this repo —
copy them over from the TypeScript app's `src/webhook/jsons` and `src/webhook/responses` before
running the sandbox against real domain traffic.

## Known parity notes

- JSON key ordering in responses may differ from the Node version (Go's `encoding/json` marshals
  map keys alphabetically; Node preserves insertion order). Functionally equivalent, cosmetic only.
- JSONata expressions (`jsonata:` prefixed template values) are evaluated with
  [`blues/jsonata-go`](https://github.com/blues/jsonata-go), which implements a large subset of the
  JSONata spec but isn't guaranteed byte-for-byte identical to the reference JS engine. Smoke-test
  fixture expressions after copying them in.

## Docker image builds

### Local build

```bash
docker buildx build --platform linux/amd64,linux/arm64 -t sandbox-2.0-go:local --load .
```

### Push to Docker Hub

This needs authorization to push to Docker Hub. You can use `docker login` to login to Docker Hub.

```bash
docker buildx build --platform linux/amd64,linux/arm64 -t fidedocker/sandbox-2.0-go:latest --push .
```
