import test from "node:test";
import assert from "node:assert/strict";
import { csv, analysisRows } from "./export.js";

test("CSV preserves complete experiment data and neutralizes spreadsheet formulas", () => {
  const result = {
    unit: "cents",
    input: {
      problem: "knapsack",
      seeds: [7],
      selection: { capacity: 10 },
      groups: [],
    },
    groups: [
      {
        name: "=test",
        best: 3100,
        mean: 3100,
        stdDev: 0,
        meanElapsedMs: 2,
        runs: [
          {
            seed: 7,
            value: 3100,
            elapsedMs: 2,
            knapsack: { greedyRepair: true },
            curve: [2800, 3100],
          },
        ],
      },
    ],
  };
  const rows = analysisRows(result),
    out = csv(rows);
  assert.ok(out.startsWith("\uFEFF"));
  assert.ok(out.includes("'=test"));
  assert.ok(out.includes("[2800,3100]"));
  assert.ok(out.includes("greedyRepair"));
  assert.ok(out.includes("capacity"));
  assert.equal(csv([['a"b', "x,y"]]), '\uFEFF"a""b","x,y"');
});
