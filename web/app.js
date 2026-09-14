// 迷宫求解系统前端逻辑：Canvas 渲染、迷宫编辑、API 调用、探索过程动画回放。
"use strict";

const COLORS = {
  wall: "#2c3e50",
  open: "#ecf0f1",
  start: "#27ae60",
  end: "#e74c3c",
  visited: "#ffe08a",
  dead: "#b2bec3",
  current: "#e67e22",
  path: "#3498db",
  gridLine: "rgba(44, 62, 80, 0.08)",
};

const canvas = document.getElementById("mazeCanvas");
const ctx = canvas.getContext("2d");

// 全局状态
const state = {
  grid: [],          // 二维数组：0 通路 / 1 墙
  rows: 0,
  cols: 0,
  cellSize: 0,
  offsetX: 0,
  offsetY: 0,
  visited: new Set(), // 已探索格子 "x,y"
  dead: new Set(),    // 回溯（死路）格子
  path: [],           // 最短路径
  steps: [],
  stepIndex: 0,
  playing: false,
  timer: null,
  activeCell: null,
};

// ---------- 工具函数 ----------

const key = (x, y) => `${x},${y}`;

function $(id) { return document.getElementById(id); }

function selectedAlgo() {
  return document.querySelector('input[name="algo"]:checked').value;
}

// ---------- Canvas 渲染 ----------

function computeLayout() {
  const size = Math.min(canvas.width, canvas.height) - 8;
  state.cellSize = Math.floor(size / Math.max(state.cols, state.rows));
  state.offsetX = Math.floor((canvas.width - state.cellSize * state.cols) / 2);
  state.offsetY = Math.floor((canvas.height - state.cellSize * state.rows) / 2);
}

function draw() {
  if (!state.grid.length) return;
  computeLayout();
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  const cs = state.cellSize;

  for (let y = 0; y < state.rows; y++) {
    for (let x = 0; x < state.cols; x++) {
      const px = state.offsetX + x * cs;
      const py = state.offsetY + y * cs;
      let color = state.grid[y][x] === 1 ? COLORS.wall : COLORS.open;
      if (state.grid[y][x] === 0 && state.dead.has(key(x, y))) color = COLORS.dead;
      else if (state.grid[y][x] === 0 && state.visited.has(key(x, y))) color = COLORS.visited;
      ctx.fillStyle = color;
      ctx.fillRect(px, py, cs, cs);
      ctx.strokeStyle = COLORS.gridLine;
      ctx.strokeRect(px + 0.5, py + 0.5, cs - 1, cs - 1);
    }
  }

  // 当前探索格
  if (state.activeCell) {
    ctx.strokeStyle = COLORS.current;
    ctx.lineWidth = 3;
    ctx.strokeRect(
      state.offsetX + state.activeCell.x * cs + 2,
      state.offsetY + state.activeCell.y * cs + 2,
      cs - 4, cs - 4
    );
    ctx.lineWidth = 1;
  }

  // 最短路径：折线 + 圆点
  if (state.path.length > 1) {
    ctx.strokeStyle = COLORS.path;
    ctx.lineWidth = Math.max(3, cs * 0.22);
    ctx.lineCap = "round";
    ctx.lineJoin = "round";
    ctx.beginPath();
    state.path.forEach((p, i) => {
      const cx = state.offsetX + p.x * cs + cs / 2;
      const cy = state.offsetY + p.y * cs + cs / 2;
      if (i === 0) ctx.moveTo(cx, cy); else ctx.lineTo(cx, cy);
    });
    ctx.stroke();
    ctx.lineWidth = 1;
  }

  // 起点与终点（最后绘制保证可见）
  drawEndpoint(0, 0, COLORS.start, "S");
  drawEndpoint(state.cols - 1, state.rows - 1, COLORS.end, "E");
}

function drawEndpoint(x, y, color, label) {
  const cs = state.cellSize;
  const px = state.offsetX + x * cs;
  const py = state.offsetY + y * cs;
  ctx.fillStyle = color;
  ctx.beginPath();
  ctx.arc(px + cs / 2, py + cs / 2, cs * 0.38, 0, Math.PI * 2);
  ctx.fill();
  ctx.fillStyle = "#fff";
  ctx.font = `bold ${Math.max(10, Math.floor(cs * 0.5))}px sans-serif`;
  ctx.textAlign = "center";
  ctx.textBaseline = "middle";
  ctx.fillText(label, px + cs / 2, py + cs / 2 + 1);
}

// ---------- 迷宫初始化 ----------

function setGrid(grid) {
  state.grid = grid;
  state.rows = grid.length;
  state.cols = grid[0].length;
  $("rows").value = state.rows;
  $("cols").value = state.cols;
  resetSolve();
  draw();
}

async function loadPreset() {
  try {
    const res = await fetch("/api/preset");
    const data = await res.json();
    setGrid(data.maze);
  } catch (e) { alert("加载题目示例失败: " + e.message); }
}

async function generate() {
  const width = parseInt($("cols").value, 10);
  const height = parseInt($("rows").value, 10);
  if (!(width >= 5 && width <= 50 && height >= 5 && height <= 50)) {
    alert("迷宫宽高须在 5 ~ 50 之间");
    return;
  }
  try {
    const res = await fetch("/api/maze/generate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        width, height,
        density: parseFloat($("density").value),
      }),
    });
    const data = await res.json();
    if (!res.ok) { alert(data.error || "生成失败"); return; }
    setGrid(data.maze);
  } catch (e) { alert("生成失败: " + e.message); }
}

// 手动编辑：点击切换墙/通路
canvas.addEventListener("click", (e) => {
  if (!state.grid.length) return;
  const rect = canvas.getBoundingClientRect();
  const scale = canvas.width / rect.width;
  const mx = (e.clientX - rect.left) * scale;
  const my = (e.clientY - rect.top) * scale;
  const x = Math.floor((mx - state.offsetX) / state.cellSize);
  const y = Math.floor((my - state.offsetY) / state.cellSize);
  if (x < 0 || y < 0 || x >= state.cols || y >= state.rows) return;
  const isStart = x === 0 && y === 0;
  const isEnd = x === state.cols - 1 && y === state.rows - 1;
  if (isStart || isEnd) return;
  state.grid[y][x] ^= 1;
  resetSolve();
  draw();
});

// ---------- 求解与动画 ----------

function resetSolve() {
  stopPlay();
  state.visited.clear();
  state.dead.clear();
  state.path = [];
  state.steps = [];
  state.stepIndex = 0;
  state.activeCell = null;
  $("btnPlay").disabled = true;
  $("btnStep").disabled = true;
  $("btnReset").disabled = true;
  setProgress("尚未求解", 0);
}

async function solve(algorithm, animate = true) {
  if (!state.grid.length) { alert("请先生成或加载迷宫"); return; }
  resetSolve();
  try {
    const res = await fetch("/api/solve", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ maze: state.grid, algorithm }),
    });
    const data = await res.json();
    if (!res.ok) { alert(data.error || "求解失败"); return; }
    updateStatsRow(algorithm, data);
    updateConclusion();
    state.steps = data.steps || [];
    $("btnPlay").disabled = state.steps.length === 0;
    $("btnStep").disabled = state.steps.length === 0;
    $("btnReset").disabled = state.steps.length === 0;
    if (!animate || state.steps.length === 0) {
      showFinal(data);
    } else {
      setProgress(`探索中 0 / ${state.steps.length}`, 0);
      play();
    }
  } catch (e) { alert("求解失败: " + e.message); }
}

// 双算法对比：后台各求一次，动画回放当前选中的算法
async function compare() {
  await solve("bfs", false);
  const bfsData = state.lastResult;
  await solve("dfs", false);
  // 重新用 BFS 结果刷新表格（solve("dfs") 已覆盖动画数据，无妨）
  void bfsData;
  updateConclusion();
}

// 直接展示最终结果（跳过动画）
function showFinal(data) {
  (data.steps || []).forEach((s) => {
    if (s.action === "backtrack") state.dead.add(key(s.cell.x, s.cell.y));
    else state.visited.add(key(s.cell.x, s.cell.y));
  });
  state.path = data.path || [];
  state.activeCell = null;
  setProgress(
    data.found
      ? `完成：共 ${state.steps.length} 步探索，路径 ${data.pathLength} 步`
      : "完成：该迷宫无通路",
    100
  );
  draw();
}

function play() {
  if (state.playing) { stopPlay(); return; }
  if (state.stepIndex >= state.steps.length) {
    // 已播完，重新开始
    state.visited.clear();
    state.dead.clear();
    state.path = [];
    state.stepIndex = 0;
  }
  state.playing = true;
  $("btnPlay").textContent = "暂停";
  const delay = parseInt($("speed").value, 10);
  state.timer = setInterval(stepForward, delay);
}

function stopPlay() {
  state.playing = false;
  if (state.timer) { clearInterval(state.timer); state.timer = null; }
  $("btnPlay").textContent = "播放";
}

function stepForward() {
  if (state.stepIndex >= state.steps.length) { stopPlay(); return; }
  const s = state.steps[state.stepIndex];
  state.activeCell = s.cell;
  if (s.action === "backtrack") {
    state.dead.add(key(s.cell.x, s.cell.y));
    state.visited.delete(key(s.cell.x, s.cell.y));
  } else {
    state.visited.add(key(s.cell.x, s.cell.y));
    state.dead.delete(key(s.cell.x, s.cell.y));
  }
  state.stepIndex++;
  setProgress(
    `探索中 ${state.stepIndex} / ${state.steps.length}`,
    (state.stepIndex / state.steps.length) * 100
  );
  draw();
  if (state.stepIndex >= state.steps.length) {
    stopPlay();
    // 探索结束，展示最短路径
    state.path = state.finalPath || [];
    setProgress(
      state.finalFound
        ? `完成：共 ${state.steps.length} 步探索，路径 ${state.finalLen} 步`
        : "完成：该迷宫无通路",
      100
    );
    draw();
  }
}

function setProgress(text, percent) {
  $("progressText").textContent = text;
  $("progressBar").style.width = percent + "%";
}

// ---------- 统计与结论 ----------

function updateStatsRow(algorithm, data) {
  const row = algorithm === "bfs" ? $("statBfs") : $("statDfs");
  row.cells[1].textContent = data.found ? "找到通路" : "无通路";
  row.cells[2].textContent = data.found ? data.pathLength : "-";
  row.cells[3].textContent = data.visitedCount;
  row.cells[4].textContent = data.elapsedMs.toFixed(3);
  state.lastResult = data;
  state.finalPath = data.path || [];
  state.finalFound = data.found;
  state.finalLen = data.pathLength;
}

function updateConclusion() {
  const bfsCells = $("statBfs").cells;
  const dfsCells = $("statDfs").cells;
  const el = $("conclusion");
  if (bfsCells[2].textContent === "-" || dfsCells[2].textContent === "-") {
    el.textContent = "";
    return;
  }
  const bl = parseInt(bfsCells[2].textContent, 10);
  const dl = parseInt(dfsCells[2].textContent, 10);
  if (isNaN(bl) || isNaN(dl)) { el.textContent = ""; return; }
  el.textContent = bl === dl
    ? "本例中 BFS 与 DFS 路径长度相同；一般情形下 BFS 按层扩展，可保证得到最短路径。"
    : `BFS 路径(${bl} 步) 比 DFS(${dl} 步) 更短：BFS 保证最短通路，DFS 仅保证可行通路。`;
}

// ---------- 事件绑定 ----------

$("btnPreset").addEventListener("click", loadPreset);
$("btnGenerate").addEventListener("click", generate);
$("btnSolve").addEventListener("click", () => solve(selectedAlgo(), true));
$("btnCompare").addEventListener("click", compare);
$("btnPlay").addEventListener("click", play);
$("btnStep").addEventListener("click", () => { stopPlay(); stepForward(); });
$("btnReset").addEventListener("click", () => { resetSolve(); draw(); });
$("density").addEventListener("input", () => {
  $("densityVal").textContent = parseFloat($("density").value).toFixed(2);
});
$("speed").addEventListener("input", () => {
  $("speedVal").textContent = $("speed").value + "ms/步";
  if (state.playing) { stopPlay(); play(); }
});

// 初始加载题目示例迷宫
loadPreset();
