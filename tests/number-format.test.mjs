import test from "node:test";
import assert from "node:assert/strict";
import {
  compactNumber,
  exactNumber,
} from "../internal/web/assets/number-format.js";
test("token counts use K/M/B with at most two decimals, preserving small counts", () => {
  for (const [n, s] of [
    [0, "0"],
    [656, "656"],
    [999, "999"],
    [1000, "1K"],
    [3120, "3.12K"],
    [3120000, "3.12M"],
    [1374596160, "1.37B"],
    [999999, "1M"],
    [999999999, "1B"],
  ])
    assert.equal(compactNumber(n), s);
  for (const n of [null, undefined, NaN, Infinity])
    assert.equal(compactNumber(n), "Unknown");
  assert.equal(exactNumber(3120000), "3,120,000");
});
