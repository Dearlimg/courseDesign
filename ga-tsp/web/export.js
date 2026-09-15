export function csv(rows) {
  return (
    "\uFEFF" +
    rows
      .map((row) =>
        row
          .map((value) => {
            let s = String(value ?? "");
            if (typeof value === "string" && /^[=+@\-\t\r]/.test(s))
              s = "'" + s;
            return '"' + s.replaceAll('"', '""') + '"';
          })
          .join(","),
      )
      .join("\r\n")
  );
}
export function analysisRows(result) {
  const rows = [
    ["骑迹重复实验"],
    ["问题", result.input.problem],
    ["数值单位", result.unit],
    ["标准差", "样本标准差 n-1；单次运行记为 0"],
    ["输入快照", JSON.stringify(result.input)],
    [],
    ["配置组", "最佳值", "平均值", "标准差", "平均耗时 ms"],
  ];
  for (const g of result.groups)
    rows.push([g.name, g.best, g.mean, g.stdDev, g.meanElapsedMs]);
  rows.push(
    [],
    ["配置组", "种子", "最终值", "耗时 ms", "实际参数", "历史最优曲线"],
  );
  for (const g of result.groups)
    for (const r of g.runs)
      rows.push([
        g.name,
        r.seed,
        r.value,
        r.elapsedMs,
        JSON.stringify(r.tsp || r.knapsack),
        JSON.stringify(r.curve),
      ]);
  return rows;
}
export function download(content, type, name) {
  const url = URL.createObjectURL(new Blob([content], { type })),
    a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 10000);
}
