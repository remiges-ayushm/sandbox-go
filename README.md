# sandbox-go

Go port of the [ion-sandbox-app](../ion-sandbox-app) TypeScript sandbox for Beckn v2.0.0 protocol
specifications, available at https://github.com/beckn/protocol-specifications-new

## Running locally

```bash
go run .
```

By default the server listens on port `3000` (override with `PORT`). Copy `.env.example` to `.env`
to configure `PORT`, `PERSONA`, `BPP_CALLBACK_ENDPOINT`, and `RESPONSE_FIXTURES_BASE_URL`.

## Response fixtures

Structured response fixtures matching the ION payload layout are fetched over HTTP from GitHub:

```text
<RESPONSE_FIXTURES_BASE_URL>/<sector>/<pattern>/<crc>/<action>.json
<RESPONSE_FIXTURES_BASE_URL>/<sector>/<pattern>/<action>.json
```

Set `RESPONSE_FIXTURES_BASE_URL` to the raw-content URL of the directory that directly contains
the sector folders (`trade/`, `hospitality/`, ...), e.g.
`https://raw.githubusercontent.com/<org>/<repo>/<branch>/responses`. The resolver uses request
metadata in this order:

- Headers: `x-ion-sector`, `x-ion-pattern`, `x-ion-crc`
- Body metadata: `_meta.sector`, `_meta.pattern`, `_meta.crc`
- Context: `context.domain` for sector
- Best effort CRC hints from known `trc-*` codes or CRC names in the body

If `RESPONSE_FIXTURES_BASE_URL` is unset, or no structured fixture matches on GitHub, the sandbox
falls back to the legacy domain tree read from local disk:

```text
internal/webhook/jsons/<domain>/response/<action>.json
```

`_meta` is used only for fixture lookup and is not sent in callbacks.

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
