export const $ = (id) => document.getElementById(id);
export const money = (cents) => (cents / 100).toFixed(2);
export const storageKey = (kind) =>
  "qiji.user." + document.documentElement.dataset.userId + "." + kind + ".v1";
export const sample = () => ({
  capacity: 10,
  orders: [
    { id: "A", name: "1 号宿舍", x: 20, y: 25, load: 4, income: 1200 },
    { id: "B", name: "2 号宿舍", x: 65, y: 20, load: 3, income: 1000 },
    { id: "C", name: "图书馆", x: 80, y: 70, load: 2, income: 700 },
    { id: "D", name: "实验楼", x: 30, y: 85, load: 5, income: 1400 },
    { id: "E", name: "教学楼", x: 50, y: 50, load: 3, income: 800 },
  ],
});
export function notify(message, error = false) {
  $("notice").textContent = message;
  $("notice").classList.toggle("error", error);
}
export function validate(data) {
  if (
    !Number.isInteger(data.capacity) ||
    data.capacity < 1 ||
    data.capacity > 10000
  )
    throw Error("餐箱容量须为 1～10000 的整数");
  if (!Array.isArray(data.orders) || data.orders.length > 100)
    throw Error("最多支持 100 笔订单");
  const ids = new Set();
  for (const o of data.orders) {
    if (
      typeof o.id !== "string" ||
      !o.id.trim() ||
      o.id.length > 60 ||
      ids.has(o.id.trim())
    )
      throw Error("订单编号不能为空、重复或超过 60 字符");
    ids.add(o.id.trim());
    if (typeof o.name !== "string" || !o.name.trim() || o.name.length > 100)
      throw Error("请填写不超过 100 字符的送达点");
    if (![o.x, o.y].every((v) => Number.isFinite(v) && v >= 0 && v <= 1000))
      throw Error("坐标须为 0～1000 的有限数");
    if (!Number.isInteger(o.load) || o.load < 1 || o.load > 10000)
      throw Error("容量占用须为 1～10000 的整数");
    if (!Number.isInteger(o.income) || o.income < 1 || o.income > 100000)
      throw Error("配送收入须为 0.01～1000.00 元，最多两位小数");
  }
  return data;
}
export let state = sample();
try {
  const saved = localStorage.getItem(storageKey("orders"));
  if (saved) state = validate(JSON.parse(saved));
} catch {
  notify("本机保存的数据无法读取，已载入园区示例。", true);
}
export function current() {
  const orders = [...$("orderRows").children].map((row) => {
    const v = (key) =>
      row.querySelector('[data-key="' + key + '"]').value.trim();
    const amount = v("income");
    if (!/^\d+(\.\d{1,2})?$/.test(amount))
      throw Error("配送收入最多保留两位小数");
    return {
      id: v("id"),
      name: v("name"),
      x: v("x") === "" ? NaN : Number(v("x")),
      y: v("y") === "" ? NaN : Number(v("y")),
      load: Number(v("load")),
      income: Math.round(Number(amount) * 100),
    };
  });
  return validate({ capacity: Number($("capacity").value), orders });
}
function persist() {
  document.dispatchEvent(new Event("orderschanged"));
  try {
    state = current();
    localStorage.setItem(storageKey("orders"), JSON.stringify(state));
    summary();
    notify("订单数据已保存。");
  } catch (e) {
    notify(e.message, true);
  }
}
function summary() {
  $("orderSummary").textContent =
    state.orders.length +
    " 笔订单 · 配送收入合计 ¥" +
    money(state.orders.reduce((s, o) => s + o.income, 0));
}
export function renderOrders() {
  $("orderRows").replaceChildren();
  for (const order of state.orders) addRow(order);
  $("capacity").value = state.capacity;
  summary();
}
function addRow(order) {
  const row = document.createElement("tr");
  for (const key of ["id", "name", "x", "y", "load", "income"]) {
    const td = document.createElement("td"),
      input = document.createElement("input");
    input.dataset.key = key;
    input.value = key === "income" ? money(order[key]) : order[key];
    input.setAttribute(
      "aria-label",
      order.id +
        " " +
        {
          id: "订单编号",
          name: "送达点",
          x: "X 坐标",
          y: "Y 坐标",
          load: "容量占用",
          income: "配送收入",
        }[key],
    );
    input.type = ["id", "name"].includes(key) ? "text" : "number";
    if (input.type === "number") input.step = key === "load" ? "1" : "0.01";
    input.addEventListener("input", persist);
    td.append(input);
    row.append(td);
  }
  const td = document.createElement("td"),
    button = document.createElement("button");
  button.textContent = "删除";
  button.onclick = () => {
    row.remove();
    persist();
  };
  td.append(button);
  row.append(td);
  $("orderRows").append(row);
}
$("add").onclick = () => {
  if ($("orderRows").children.length >= 100)
    return notify("最多支持 100 笔订单", true);
  addRow({
    id: "O" + Date.now(),
    name: "新送达点",
    x: 50,
    y: 50,
    load: 1,
    income: 500,
  });
  persist();
};
$("reset").onclick = () => {
  state = sample();
  renderOrders();
  persist();
};
$("capacity").addEventListener("input", persist);
renderOrders();
export async function api(url, payload, signal) {
  const response = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
    signal,
  });
  const result = await response.json();
  if (response.status === 401) {
    location.replace("/auth.html");
    throw Error("登录已失效");
  }
  if (!response.ok) throw Error(result.error || "请求失败");
  return result;
}
export function text(tag, value, className = "") {
  const node = document.createElement(tag);
  node.textContent = value;
  node.className = className;
  return node;
}
export function stats(target, items) {
  target.replaceChildren(
    ...items.map(([label, value, note]) => {
      const box = text("div", "", "stat");
      box.append(
        text("small", label),
        text("strong", value),
        text("small", note || ""),
      );
      return box;
    }),
  );
}
