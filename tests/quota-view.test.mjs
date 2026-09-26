import test from "node:test";
import assert from "node:assert/strict";
import { quotaAccounts } from "../internal/web/assets/quota-view.js";
const at = "2026-09-26T08:00:00Z";
const one = {
  provider: "codex",
  email: "synthetic@example.test",
  profile: "default",
  person: "alice",
  device: "mac",
  observed_at: at,
  windows: [
    {
      bucket: "codex",
      name: "primary",
      used_percent: 48,
      resets_at: 1790496000,
    },
  ],
};
test("shared quota is one account with device provenance, never summed", () => {
  const rows = quotaAccounts([one, { ...one, device: "pc" }], Date.parse(at));
  assert.equal(rows.length, 1);
  assert.equal(rows[0].devices.length, 2);
  assert.equal(rows[0].latest.windows[0].used_percent, 48);
});
test("failed read marks prior account stale and retains its last good percent", () => {
  const rows = quotaAccounts(
    [
      {
        ...one,
        email: "",
        error: "unavailable",
        windows: [],
        observed_at: "2026-09-26T08:01:00Z",
      },
      one,
    ],
    Date.parse(at) + 60000,
  );
  assert.equal(rows[0].health, "Read failed");
  assert.equal(rows[0].latest.windows[0].used_percent, 48);
});
test("scheduled reset passing invalidates current remaining claim", () => {
  const rows = quotaAccounts([one], 1790496001000);
  assert.equal(rows[0].health, "Needs refresh");
});
test("unknown quota remains null, old profile account changes are retained", () => {
  const rows = quotaAccounts(
    [
      {
        ...one,
        email: "second@example.test",
        observed_at: "2026-09-26T08:01:00Z",
        windows: [{ bucket: "codex", name: "primary", used_percent: null }],
      },
      one,
    ],
    Date.parse(at) + 60000,
  );
  assert.equal(rows.length, 2);
  assert.equal(rows[0].latest.windows[0].used_percent, null);
});
test("a provider may return no windows without breaking the account view", () => {
  const rows = quotaAccounts([{ ...one, windows: null }], Date.parse(at));
  assert.deepEqual(rows[0].latest.windows, []);
});
