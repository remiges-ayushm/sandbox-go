# Trade Sector Response Fixtures

Callback response fixtures for the `trade` sector, served by `readRequestResponse()` in
[`src/utils/index.ts`](../../../utils/index.ts) and consumed by the handlers in
[`src/webhook/controller.ts`](../controller.ts).

Spec reference: [ion-specs / flows/trade](https://github.com/indonesiaopennetwork/ion-specs/tree/ion-launch/flows/trade)

## Taxonomy

- **Sector** — top-level domain of commerce. The spec defines 6 (`trade`, `hospitality`,
  `logistics`, `mobility`, `finance`, `services`). This directory only implements `trade`
  (`ion:trade`).
- **Pattern** — a complete commerce scenario/blueprint, from catalog publish to
  reconciliation. Trade has 12 patterns, e.g. `storefront` (the reference pattern — B2C
  browse/buy), `forward-auction`, `subscription`, `business-procurement`, etc.
- **Variant** — a conditional modifier that activates *within* a pattern when something
  happens (not an alternate pattern). Trade has 6: `during-transaction`, `cancellation`,
  `returns`, `rto`, `mid-transaction-changes`, `cross-cutting`. Example: a `storefront`
  transaction picks up the `cancellation` variant the moment `on_confirm` fires, until the
  package is `DISPATCHED` (see the pattern's `variantWindows`). Multiple variants can be
  active on one transaction simultaneously.
- **CRC (Resource Category Code)** — the goods taxonomy, e.g. `TRC-fashion`,
  `TRC-health-beauty`, `TRC-arts`. Trade has 19 categories in the spec; 4 are
  `INACTIVE`/`GOVERNED` and are intentionally not represented here.
- **Spec path notation**: `trade / category-code / resource-category-code / pattern / variant`,
  e.g. `trade / TRD-04 / TRC-health-beauty / storefront / delivery-time-kyc`.

## How routing actually works here

`readRequestResponse()` resolves `{sector, pattern, crc}` from request headers → `_meta` →
`context` → best-effort inference, then tries candidate file paths in this priority order,
returning the first one that exists on disk:

1. `sector/pattern/crc/persona/action.json`
2. `sector/pattern/persona/action.json`
3. `sector/persona/action.json`
4. `sector/pattern/crc/action.json`
5. `sector/pattern/action.json` ← what most requests actually hit today
6. `sector/action.json`
7. legacy fallback (`src/webhook/jsons/<domain>/response/<action>.json`)

CRC resolution is forgiving: `CRC_NAME_TO_CODE` in `src/utils/index.ts` maps human category
names to folder codes, and a deep scan also matches literal `trc-*` strings anywhere in the
request body. Pattern resolution is similarly aliased via `PATTERN_NAME_TO_CODE`, which maps
the spec's real pattern names (e.g. `storefront`, `forward-auction`) to this app's folder
codes (e.g. `B2C-SF`, `AUC-F`) — without it, a realistic ION request that sends the spec's
actual pattern name would never match these folders, since the abbreviations below are purely
a local invention and don't appear anywhere in the spec.

## Pattern folder ↔ spec pattern name

| Local folder | Spec pattern name        |
|--------------|---------------------------|
| `AUC-F`      | `forward-auction`          |
| `AUC-R`      | `reverse-auction`          |
| `B2B-PP`     | `business-procurement`     |
| `B2C-DIG`    | `digital-goods`            |
| `B2C-LIVE`   | `live-commerce`            |
| `B2C-MTO`    | `made-to-order`            |
| `B2C-SF`     | `storefront` (reference)   |
| `B2C-SUB`    | `subscription`             |
| `B2G`        | `government`               |
| `MP-IH`      | `marketplace-inhouse`      |
| `MP-IL`      | `marketplace-listed`       |
| `XB`         | `cross-border`             |
| `B2B-CR`     | *(not a pattern — see below)* |

All 12 real trade patterns are represented.

## Known deviations from the spec (documented, not fixed)

- **`B2B-CR` is a variant, not a pattern.** In the spec, `business-credit` is a *variant* of
  the `business-procurement` pattern, not a standalone pattern. Here it's implemented as a
  flat sibling folder to `B2B-PP` with its own full `on_*.json` set. The router matches it via
  `meta.variant` being read as a `pattern` fallback in `resolveResponseRoute()` — it works, but
  is conceptually inconsistent with the spec's pattern/variant hierarchy. Left as-is
  intentionally; not restructured.
- **`trc-industrial` vs. spec's `TRC-b2b`.** The spec's actual code for "Business & Industrial"
  is `TRC-b2b`; this app uses `trc-industrial` (see `CRC_NAME_TO_CODE`). Left as-is
  intentionally; not renamed.
- **Missing catalog fixtures for 4 patterns.** `B2B-CR`, `B2B-PP`, `B2C-SUB`, and `MP-IL` have
  no CRC-level `publish_catalog.json` files (every other pattern has 15, one per active CRC).
  A `catalog/publish` call for these 4 patterns falls through every structured candidate to
  the legacy fallback. Known gap; not filled in.

## `on_status*` files: two different purposes

- **Auto-served**: `on_status.json` is the *only* filename `readRequestResponse` ever requests
  for the `/status` route — `controller.ts`'s `onStatus` handler always calls with the literal
  action `"on_status"`.
- **Manual reference payloads**, not auto-loaded: `on_status[DELIVERED].json`,
  `on_status[RETURN_DELIVERED].json`, and (only in `B2C-SF`) `on_status_packed.json`,
  `on_status_dispatched.json`, `on_status_out_for_delivery.json`, `on_status_delivered.json`.
  These exist as copy-paste source material for manually POSTing to the unsolicited
  `/trigger/on_status` (and `/trigger/on_cancel`, `/trigger/on_update`) routes, which relay
  whatever `{context, message}` body you send them rather than loading a fixture. `B2C-SF`
  uses underscore-style names (`on_status_delivered.json`) while other patterns use
  bracket-style names (`on_status[DELIVERED].json`) for the same purpose — inconsistent, but
  harmless since neither is auto-loaded.

## The realism gap (why JSONata is being introduced)

Every handler in `controller.ts` does `{ ...template, context: buildResponseContext(context,
action) }` — only `context` is ever rebuilt from the incoming request; `message` (the actual
order/catalog/payment content) is 100% static from the JSON fixture, and the request's own
`message` is destructured but unused. So `on_confirm` never reflects what was actually
selected/inited, buyer info is always the same hardcoded person, totals never recompute, etc.

The intended seam for a JSONata-based fix is right after the template is loaded, in each
handler:

```ts
const template = await readRequestResponse(req.body, "on_confirm", getPersona(), req.headers);
const transformed = applyJsonata(template, req.body); // <- seam for JSONata-based realism
const responsePayload = {
  ...transformed,
  context: buildResponseContext(context, "confirm"),
};
```

`applyJsonata` is not implemented yet — no `jsonata` dependency is installed, and no
expressions have been authored. This is left for follow-up work.
