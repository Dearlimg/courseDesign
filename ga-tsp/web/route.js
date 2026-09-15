import { $, current, api, notify, text, stats } from "./dispatch.js";
import { playback } from "./charts.js";
import { configPanel } from "./config.js";
const host = $("routeContent");
host.className = "";
host.innerHTML =
  '<div class="config"><label>取餐点名称<input id="depotName" value="园区取餐点"></label><label>X 坐标<input id="depotX" type="number" value="10"></label><label>Y 坐标<input id="depotY" type="number" value="10"></label></div><div class="split"><div><label>送达点：每行 地点名称,X,Y<textarea id="routePoints" rows="7" placeholder="图书馆,80,70"></textarea></label><div class="toolbar"><button id="allToRoute">带入全部候选订单</button><button id="routeRun" class="primary">规划配送路线</button></div><p id="routeSource" class="muted">独立编辑的送达点</p><p id="routeStatus" role="status"></p><ol id="routeList"></ol></div><canvas id="routeMap" width="620" height="440" aria-label="配送路线示意图"></canvas></div><div id="routeStats" class="stats"></div><details><summary>查看路线演化过程</summary><p class="muted">回放显示原始 GA 搜索结果；业务采用路线同时与输入顺序基准比较，始终保留更短方案。</p><div id="routePlayback" class="playback"></div><canvas id="routeEvolution" width="900" height="400" aria-label="遗传路线进化图"></canvas></details>';
let version = 0,
  source = null,
  stop = () => {},
  controller = null;
const routeParams = configPanel(host, "tsp", () => stale());
function save() {
  try {
    localStorage.setItem(
      "qiji.route.v1",
      JSON.stringify({
        name: $("depotName").value,
        x: $("depotX").value,
        y: $("depotY").value,
        points: $("routePoints").value,
      }),
    );
  } catch {
    notify("浏览器无法保存路线输入。", true);
  }
}
try {
  const s = JSON.parse(localStorage.getItem("qiji.route.v1"));
  if (s && typeof s.points === "string") {
    $("depotName").value = s.name;
    $("depotX").value = s.x;
    $("depotY").value = s.y;
    $("routePoints").value = s.points;
  }
} catch {}
function stale() {
  version++;
  controller?.abort();
  stop();
  $("routeStatus").textContent = "输入已变化，原路线已过期，请重新规划。";
  $("routeStatus").className = "stale";
  document.dispatchEvent(new Event("routeinvalid"));
}
for (const id of ["depotName", "depotX", "depotY", "routePoints"])
  $(id).addEventListener("input", () => {
    if (id === "routePoints") {
      source = null;
      $("routeSource").textContent = "独立编辑的送达点";
    }
    save();
    stale();
  });
function accept(orders) {
  source = orders.map((o) => ({ ...o }));
  $("routePoints").value = orders
    .map((o) => o.name + "," + o.x + "," + o.y)
    .join("\n");
  $("routeSource").textContent =
    "来自 " + orders.length + " 笔订单；相同坐标会合并停靠点。";
  save();
  stale();
}
document.addEventListener("routeorders", (e) => accept(e.detail));
$("allToRoute").onclick = () => {
  try {
    accept(current().orders);
  } catch (e) {
    notify(e.message, true);
  }
};
document.addEventListener("orderschanged", () => {
  if (source) {
    source = null;
    stale();
    $("routePoints").value = "";
    save();
    $("routeSource").textContent = "关联订单已修改，请重新带入。";
  }
});
export function routeInput() {
  const coord = (value) => (value.trim() === "" ? NaN : Number(value));
  const depot = {
    name: $("depotName").value.trim(),
    x: coord($("depotX").value),
    y: coord($("depotY").value),
  };
  const lines = $("routePoints")
    .value.trim()
    .split("\n")
    .filter((s) => s.trim());
  const points = source
    ? source.map((o) => ({ name: o.name, x: o.x, y: o.y, orderIds: [o.id] }))
    : lines.map((line, i) => {
        const parts = line.split(/[,，]/).map((s) => s.trim());
        if (parts.length !== 3)
          throw Error("第 " + (i + 1) + " 行须为 地点名称,X,Y");
        return {
          name: parts[0],
          x: coord(parts[1]),
          y: coord(parts[2]),
          orderIds: source ? [source[i].id] : [],
        };
      });
  if (!points.length) throw Error("请先输入或带入送达点");
  for (const p of [depot, ...points])
    if (
      !p.name ||
      ![p.x, p.y].every((v) => Number.isFinite(v) && v >= 0 && v <= 1000)
    )
      throw Error("地点须有名称，坐标须为 0～1000");
  return { depot, points };
}
export function drawRoute(canvas, stops, tour) {
  const c = canvas.getContext("2d"),
    w = canvas.width,
    h = canvas.height;
  c.fillStyle = "#fafbf6";
  c.fillRect(0, 0, w, h);
  if (!stops.length) return;
  const xs = stops.map((p) => p.x),
    ys = stops.map((p) => p.y),
    minX = Math.min(...xs),
    minY = Math.min(...ys),
    span = Math.max(Math.max(...xs) - minX, Math.max(...ys) - minY, 1);
  const scale = Math.min(w - 120, h - 100) / span;
  const xy = (p) => [60 + (p.x - minX) * scale, h - 50 - (p.y - minY) * scale];
  c.strokeStyle = "#e1e8d9";
  c.lineWidth = 1;
  for (let x = 30; x < w; x += 40) {
    c.beginPath();
    c.moveTo(x, 20);
    c.lineTo(x, h - 20);
    c.stroke();
  }
  c.strokeStyle = "#24624e";
  c.lineWidth = 2.5;
  c.beginPath();
  [...tour, tour[0]].forEach((id, i) => {
    if (id === undefined) return;
    const [x, y] = xy(stops[id]);
    if (i === 0) c.moveTo(x, y);
    else c.lineTo(x, y);
  });
  c.stroke();
  c.font = "12px sans-serif";
  stops.forEach((p, i) => {
    const [x, y] = xy(p);
    c.beginPath();
    c.fillStyle = i === 0 ? "#d1a145" : "#24624e";
    c.arc(x, y, 7, 0, Math.PI * 2);
    c.fill();
    c.fillStyle = "#234c3c";
    const label =
      (i === 0 ? "取餐点" : String(tour.indexOf(i))) + " · " + p.name;
    c.fillText(
      label,
      Math.min(x + 12, w - c.measureText(label).width - 8),
      y - 10,
    );
  });
}
$("routeRun").onclick = async () => {
  const token = version;
  stop();
  try {
    const request = { ...routeInput(), params: routeParams() };
    controller = new AbortController();
    $("routeRun").disabled = true;
    $("routeStatus").textContent = "正在优化配送顺序…";
    $("routeStatus").className = "";
    const r = await api("/api/dispatch/route", request, controller.signal);
    if (token !== version) return;
    $("routeStatus").textContent =
      r.method === "direct"
        ? "送达点较少，已直接计算闭合路线。"
        : r.usedBaseline
          ? "本次未找到更短路线，采用输入顺序基准。"
          : "已生成推荐配送路线。";
    stats($("routeStats"), [
      ["本趟路程", r.distance.toFixed(2), "平面距离单位，包含返站"],
      [
        "输入顺序基准",
        r.baselineDistance.toFixed(2),
        "同一批地点、同一距离模型",
      ],
      [
        "路程改善",
        r.baselineDistance ? (r.improvement * 100).toFixed(1) + "%" : "不适用",
        "不代表实际配送时间改善",
      ],
    ]);
    $("routeList").replaceChildren(
      ...[...r.tour, 0].map((id, i) =>
        text(
          "li",
          (i === r.tour.length ? "返站：" : "") +
            r.stops[id].name +
            (r.stops[id].orderIds?.length
              ? " · 订单 " + r.stops[id].orderIds.join("、")
              : ""),
        ),
      ),
    );
    drawRoute($("routeMap"), r.stops, r.tour);
    stop = playback($("routePlayback"), r.evolution.generations, (g) =>
      drawRoute($("routeEvolution"), r.stops, g.bestTour),
    );
    if (!r.evolution.generations.length)
      drawRoute($("routeEvolution"), r.stops, r.tour);
    notify("配送路线已生成。");
  } catch (e) {
    if (e.name !== "AbortError") {
      $("routeStatus").textContent = e.message;
      notify(e.message, true);
    }
  } finally {
    $("routeRun").disabled = false;
    controller = null;
  }
};
