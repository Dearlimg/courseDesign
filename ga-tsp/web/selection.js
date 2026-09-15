import { $, current, api, money, text, stats, notify } from "./dispatch.js";
import { chart, playback } from "./charts.js";
import { configPanel } from "./config.js";
const host = $("selectContent");
host.className = "";
host.innerHTML =
  '<button id="selectRun" class="primary">生成接单建议</button><p id="selectStatus" role="status"></p><div id="selectStats" class="stats"></div><div id="selectOrders" class="cards"></div><div id="selectExcluded"></div><button id="toRoute" hidden>规划这些订单的路线 →</button><details><summary>查看遗传优化过程</summary><p class="muted">0/1 表示接或不接。绿色为当前代入选；不可行代仅用于研究，不能作为最终接单结果。</p><div id="selectPlayback" class="playback"></div><p id="selectFrame"></p><div id="selectGenes" class="cards"></div><canvas id="selectCurve" width="900" height="280" aria-label="接单收益收敛曲线"></canvas></details>';
let result = null,
  stop = () => {},
  version = 0,
  controller = null;
const selectParams = configPanel(host, "knapsack", () => {
  version++;
  controller?.abort();
  stop();
  result = null;
  $("toRoute").hidden = true;
  $("selectStatus").textContent = "配置已修改，请重新生成建议。";
  document.dispatchEvent(new Event("selectioninvalid"));
});
document.addEventListener("orderschanged", () => {
  version++;
  controller?.abort();
  stop();
  result = null;
  $("toRoute").hidden = true;
  $("selectStatus").textContent = "订单已修改，原结果已过期，请重新生成。";
  $("selectStatus").className = "stale";
  document.dispatchEvent(new Event("selectioninvalid"));
});
function cards(target, orders, selectedIDs) {
  target.replaceChildren(
    ...orders.map((o) => {
      const c = text(
        "div",
        "",
        "card" + (selectedIDs.has(o.id) ? " selected" : ""),
      );
      c.append(
        text("strong", o.id + " · " + o.name),
        text("p", o.load + " 容量单位 / ¥" + money(o.income)),
      );
      return c;
    }),
  );
}
$("selectRun").onclick = async () => {
  const token = version;
  stop();
  result = null;
  $("toRoute").hidden = true;
  document.dispatchEvent(new Event("selectioninvalid"));
  try {
    const data = { ...current(), params: selectParams() };
    controller = new AbortController();
    $("selectRun").disabled = true;
    $("selectStatus").textContent = "正在生成接单建议…";
    $("selectStatus").className = "";
    const r = await api("/api/dispatch/select", data, controller.signal);
    if (token !== version) return;
    result = r;
    $("selectStatus").textContent = !r.selected.length
      ? "暂无可选订单"
      : r.verifiedOptimal
        ? "已达到当前容量模型的最优收益"
        : "当前推荐方案（尚未达到容量模型最优收益）";
    stats($("selectStats"), [
      ["预计配送收入", "¥" + money(r.income), "未扣除骑行成本"],
      [
        "推荐接单",
        r.selected.length + " 笔",
        "容量 " + r.load + " / " + data.capacity,
      ],
      ["最优收益对照", "¥" + money(r.optimalIncome), "动态规划仅用于评价"],
    ]);
    cards($("selectOrders"), r.selected, new Set(r.selected.map((o) => o.id)));
    $("selectExcluded").replaceChildren(
      ...r.excluded.map((o) => text("p", o.id + "：" + o.reason, "muted")),
    );
    chart(
      $("selectCurve"),
      [
        {
          values: r.evolution.generations.map((g) => g.bestSoFar / 100),
          color: "#24624e",
        },
        {
          values: r.evolution.generations.map((g) => g.average / 100),
          color: "#bc9142",
        },
      ],
      "收益 / 元 · 绿：历史最优，金：种群平均",
    );
    stop = playback($("selectPlayback"), r.evolution.generations, (g) => {
      const selected = r.eligible.filter((o, i) => g.genes[i] === 1),
        load = selected.reduce((s, o) => s + o.load, 0);
      $("selectFrame").textContent =
        "编码 " +
        g.genes.join("") +
        " · 占用 " +
        load +
        " / " +
        data.capacity +
        (load > data.capacity ? " · 本代不可行" : "");
      cards($("selectGenes"), r.eligible, new Set(selected.map((o) => o.id)));
    });
    if (!r.evolution.generations.length) {
      $("selectGenes").replaceChildren();
      $("selectFrame").textContent = "候选订单不足两笔，直接判定。";
    }
    $("toRoute").hidden = !r.selected.length;
    notify("接单建议已生成。");
  } catch (e) {
    if (e.name !== "AbortError") {
      $("selectStatus").textContent = e.message;
      notify(e.message, true);
    }
  } finally {
    $("selectRun").disabled = false;
    controller = null;
  }
};
$("toRoute").onclick = () => {
  if (result) {
    document.dispatchEvent(
      new CustomEvent("routeorders", { detail: result.selected }),
    );
    location.hash = "route";
  }
};
