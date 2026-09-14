/* 遗传算法求解 TSP —— 前端可视化与交互 */
'use strict';

const $ = (id) => document.getElementById(id);

const COLORS = {
  best: '#38bdf8',
  avg: '#fbbf24',
  optimal: '#22c55e',
  diversity: '#a855f7',
  grid: 'rgba(148, 163, 184, 0.14)',
  axis: 'rgba(148, 163, 184, 0.45)',
  text: '#94a3b8',
  city: '#e2e8f0',
  optimalTour: 'rgba(34, 197, 94, 0.45)',
  cloud: 'rgba(125, 211, 252, 0.13)',
  newEdge: '#4ade80',
  oldEdge: 'rgba(248, 113, 113, 0.6)',
  scanPalette: ['#38bdf8', '#f472b6', '#fbbf24', '#22c55e', '#a855f7', '#f97316', '#14b8a6', '#e2e8f0'],
};

const state = {
  instance: null,
  result: null,
  gen: 0,
  playTimer: null,
  speedMs: 40,
  scan: null,
  busy: false,
  prevBestTour: null,
  flashTimer: null,
};

/* ---------------- 通用工具 ---------------- */

async function api(path, options = {}) {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  const text = await res.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch (_) { data = null; }
  if (!res.ok) throw new Error((data && data.error) || `请求失败（HTTP ${res.status}）`);
  return data;
}

const fmt = (v, digits = 2) => (v == null || Number.isNaN(v) ? '—' : Number(v).toFixed(digits));

function setStatus(msg, isError = false, busy = false) {
  const el = $('status');
  el.textContent = msg;
  el.classList.toggle('error', isError);
  el.classList.toggle('busy', busy);
}

function drawEmpty(canvas, message) {
  const ctx = canvas.getContext('2d');
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  ctx.fillStyle = COLORS.text;
  ctx.font = '15px "Segoe UI", "Microsoft YaHei", sans-serif';
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  ctx.fillText(message, canvas.width / 2, canvas.height / 2);
  ctx.textAlign = 'start';
  ctx.textBaseline = 'alphabetic';
}

/* 无向边键 / 边集合 */
function edgeKey(a, b) { return a < b ? a + '-' + b : b + '-' + a; }
function edgeSet(tour) {
  const s = new Set();
  const n = tour.length;
  for (let i = 0; i < n; i++) s.add(edgeKey(tour[i], tour[(i + 1) % n]));
  return s;
}

/* ---------------- 实例加载 ---------------- */

function syncInstanceUI() {
  const isRandom = $('instSelect').value === 'random';
  document.querySelectorAll('.random-only').forEach((el) => {
    el.style.display = isRandom ? 'flex' : 'none';
  });
  $('loadBuiltinBtn').style.display = isRandom ? 'none' : 'inline-block';
}

async function loadInstance() {
  try {
    setStatus('正在加载实例……');
    if ($('instSelect').value === 'att48') {
      state.instance = await api('/api/instances/att48');
    } else {
      const n = parseInt($('randN').value, 10) || 30;
      const seed = parseInt($('randSeed').value, 10) || 42;
      state.instance = await api('/api/instance/random', {
        method: 'POST',
        body: JSON.stringify({ n, seed }),
      });
    }
    resetRun();
    const inst = state.instance;
    $('instInfo').textContent =
      `${inst.name} · ${inst.cities.length} 城 · ` +
      (inst.edgeType === 'att' ? 'ATT 伪欧氏距离' : '欧氏距离') +
      (inst.optimal > 0 ? ` · 已知最优 ${inst.optimal}` : ' · 最优解未知');
    $('solveBtn').disabled = false;
    $('scanBtn').disabled = false;
    setStatus('实例已就绪，点击「开始进化」');
    drawMap(null, false);
    drawEmpty($('convCanvas'), '尚未求解');
    drawEmpty($('divCanvas'), '尚未求解');
  } catch (e) {
    setStatus(e.message, true);
  }
}

function resetRun() {
  stopPlay();
  state.result = null;
  state.gen = 0;
  state.prevBestTour = null;
  $('genSlider').max = 0;
  $('genSlider').value = 0;
  $('genSlider').disabled = true;
  ['playBtn', 'stepBtn', 'resetBtn', 'jumpBestBtn', 'jumpEndBtn'].forEach((id) => { $(id).disabled = true; });
  $('genLabel').textContent = '第 0 / 0 代';
  $('mapInfo').textContent = '未求解';
  ['hudGen', 'hudBest', 'hudAvg', 'hudDiv', 'hudGap', 'hudImprove'].forEach((id) => { $(id).textContent = '—'; });
  clearStats();
}

/* ---------------- 求解 ---------------- */

function readParams() {
  return {
    population: parseInt($('pPopulation').value, 10) || 0,
    generations: parseInt($('pGenerations').value, 10) || 0,
    crossoverRate: parseFloat($('pCrossover').value),
    mutationRate: parseFloat($('pMutation').value),
    elitism: parseInt($('pElitism').value, 10) || 0,
    selection: $('pSelection').value,
    crossover: $('pCrossoverOp').value,
    mutation: $('pMutationOp').value,
    localSearch: $('pLocalSearch').checked,
    seed: parseInt($('pSeed').value, 10) || 0,
  };
}

async function solve() {
  if (!state.instance || state.busy) return;
  state.busy = true;
  $('solveBtn').disabled = true;
  setStatus('正在进化，请稍候……（代数越多耗时越长）', false, true);
  try {
    const inst = state.instance;
    const res = await api('/api/solve', {
      method: 'POST',
      body: JSON.stringify({
        instance: { name: inst.name, edgeType: inst.edgeType, cities: inst.cities },
        params: readParams(),
      }),
    });
    state.result = res;
    const last = res.generations.length - 1;
    $('genSlider').max = last;
    $('genSlider').disabled = false;
    ['playBtn', 'stepBtn', 'resetBtn', 'jumpBestBtn', 'jumpEndBtn'].forEach((id) => { $(id).disabled = false; });
    setGen(0);
    updateStats();
    setStatus(`进化完成：${res.generations.length} 代，用时 ${fmt(res.elapsedMs, 0)} ms`);
    startPlay();
  } catch (e) {
    setStatus(e.message, true);
  } finally {
    state.busy = false;
    $('solveBtn').disabled = false;
  }
}

/* ---------------- 回放控制 ---------------- */

function setGen(i) {
  if (!state.result) return;
  const gens = state.result.generations;
  const last = gens.length - 1;
  state.gen = Math.max(0, Math.min(i, last));
  $('genSlider').value = state.gen;
  const gen = gens[state.gen];
  const prev = state.gen > 0 ? gens[state.gen - 1] : null;
  state.prevBestTour = prev ? prev.bestTour : null;
  const improved = prev ? gen.best < prev.best : false;

  // HUD
  $('hudGen').textContent = `${state.gen}/${last}`;
  $('hudBest').textContent = fmt(gen.best);
  $('hudAvg').textContent = fmt(gen.avg);
  $('hudDiv').textContent = fmt(gen.diversity * 100, 0) + '%';
  const opt = state.instance.optimal || 0;
  $('hudGap').textContent = opt > 0 ? fmt(((gen.best - opt) / opt) * 100) + '%' : '—';
  const initBest = gens[0].best;
  $('hudImprove').textContent = fmt(((initBest - gen.best) / initBest) * 100) + '%';

  $('genLabel').textContent = `第 ${state.gen} / ${last} 代`;
  $('mapInfo').textContent = `第 ${state.gen} 代 · 最短 ${fmt(gen.best)}`;
  $('sCurrent').textContent = fmt(gen.best);

  drawMap(gen, improved);
  drawConvergence(state.gen);
  drawDiversity(state.gen);

  if (improved) triggerFlash(gen.best);
}

function startPlay() {
  stopPlay();
  if (!state.result) return;
  const last = state.result.generations.length - 1;
  if (state.gen >= last) setGen(0);
  $('playBtn').textContent = '暂停';
  state.playTimer = setInterval(() => {
    const end = state.result.generations.length - 1;
    if (state.gen >= end) { stopPlay(); return; }
    setGen(state.gen + 1);
  }, state.speedMs);
}

function stopPlay() {
  if (state.playTimer) clearInterval(state.playTimer);
  state.playTimer = null;
  $('playBtn').textContent = '播放';
}

function triggerFlash(best) {
  const wrap = $('mapWrap');
  wrap.classList.remove('flash');
  void wrap.offsetWidth; // 强制重排以重启动画
  wrap.classList.add('flash');
  const badge = $('newBest');
  badge.textContent = `新最优 ${fmt(best)}`;
  badge.classList.add('show');
  clearTimeout(state.flashTimer);
  state.flashTimer = setTimeout(() => {
    wrap.classList.remove('flash');
    badge.classList.remove('show');
  }, 750);
}

/* ---------------- 地图绘制 ---------------- */

function mapTransform(canvas) {
  const cities = state.instance.cities;
  const xs = cities.map((c) => c.x);
  const ys = cities.map((c) => c.y);
  const minX = Math.min(...xs), maxX = Math.max(...xs);
  const minY = Math.min(...ys), maxY = Math.max(...ys);
  const pad = 46;
  const w = canvas.width - pad * 2;
  const h = canvas.height - pad * 2;
  const spanX = Math.max(maxX - minX, 1e-6);
  const spanY = Math.max(maxY - minY, 1e-6);
  const scale = Math.min(w / spanX, h / spanY);
  const offX = pad + (w - spanX * scale) / 2;
  const offY = pad + (h - spanY * scale) / 2;
  return (c) => [
    offX + (c.x - minX) * scale,
    canvas.height - (offY + (c.y - minY) * scale),
  ];
}

function strokeTour(ctx, project, cities, tour, color, width, dashed = false) {
  ctx.save();
  ctx.strokeStyle = color;
  ctx.lineWidth = width;
  ctx.lineJoin = 'round';
  ctx.setLineDash(dashed ? [7, 6] : []);
  ctx.beginPath();
  tour.forEach((idx, i) => {
    const [x, y] = project(cities[idx]);
    if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
  });
  const [sx, sy] = project(cities[tour[0]]);
  ctx.lineTo(sx, sy);
  ctx.stroke();
  ctx.restore();
}

/* 按 edge 谓语逐段着色绘制环游 */
function strokeTourEdges(ctx, project, cities, tour, keep) {
  const n = tour.length;
  ctx.beginPath();
  let drawing = false;
  for (let i = 0; i < n; i++) {
    const a = tour[i], b = tour[(i + 1) % n];
    if (keep(a, b)) {
      const [x, y] = project(cities[a]);
      if (!drawing) { ctx.moveTo(x, y); drawing = true; }
      else ctx.lineTo(x, y);
      const [x2, y2] = project(cities[b]);
      ctx.lineTo(x2, y2);
    } else {
      drawing = false;
    }
  }
  ctx.stroke();
}

function drawMap(gen, improved) {
  const canvas = $('mapCanvas');
  const ctx = canvas.getContext('2d');
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  if (!state.instance) { drawEmpty(canvas, '请先加载实例'); return; }
  const cities = state.instance.cities;
  const project = mapTransform(canvas);
  const tour = gen ? gen.bestTour : null;

  // 已知最优回路（参考基准）
  if (state.instance.optimalTour && state.instance.optimalTour.length) {
    strokeTour(ctx, project, cities, state.instance.optimalTour, COLORS.optimalTour, 2, true);
  }

  // 种群云：抽样个体（淡色），直观显示种群分布与收敛趋势
  if (gen && gen.sampleTours && gen.sampleTours.length) {
    ctx.save();
    ctx.strokeStyle = COLORS.cloud;
    ctx.lineWidth = 1;
    ctx.lineJoin = 'round';
    gen.sampleTours.forEach((t) => {
      if (!t || !t.length) return;
      ctx.beginPath();
      t.forEach((idx, i) => {
        const [x, y] = project(cities[idx]);
        if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
      });
      const [sx, sy] = project(cities[t[0]]);
      ctx.lineTo(sx, sy);
      ctx.stroke();
    });
    ctx.restore();
  }

  // 进化差异：上一代最优 → 当代最优 的边变化
  if (state.prevBestTour && tour && state.prevBestTour.length === tour.length) {
    const prevEdges = edgeSet(state.prevBestTour);
    const curEdges = edgeSet(tour);
    // 消失的边（红色虚线，沿上一代回路绘制）
    ctx.save();
    ctx.strokeStyle = COLORS.oldEdge;
    ctx.lineWidth = 1.6;
    ctx.setLineDash([5, 4]);
    strokeTourEdges(ctx, project, cities, state.prevBestTour,
      (a, b) => !curEdges.has(edgeKey(a, b)));
    ctx.restore();
    // 新增的边（绿色发光，沿当代回路绘制）
    ctx.save();
    ctx.shadowColor = 'rgba(74, 222, 128, 0.9)';
    ctx.shadowBlur = 10;
    ctx.strokeStyle = COLORS.newEdge;
    ctx.lineWidth = 2.8;
    strokeTourEdges(ctx, project, cities, tour,
      (a, b) => !prevEdges.has(edgeKey(a, b)));
    ctx.restore();
  }

  // 当代最优回路（青色加粗）
  if (tour && tour.length) {
    ctx.save();
    ctx.shadowColor = 'rgba(56, 189, 248, 0.9)';
    ctx.shadowBlur = 14;
    strokeTour(ctx, project, cities, tour, COLORS.best, 2.6);
    ctx.restore();
  }

  // 城市点
  const startIdx = tour && tour.length ? tour[0] : 0;
  cities.forEach((c, i) => {
    const [x, y] = project(c);
    ctx.beginPath();
    ctx.arc(x, y, i === startIdx ? 6.5 : 4, 0, Math.PI * 2);
    ctx.fillStyle = i === startIdx ? '#22d3ee' : '#0b1020';
    ctx.fill();
    ctx.lineWidth = 1.6;
    ctx.strokeStyle = i === startIdx ? '#a5f3fc' : COLORS.city;
    ctx.stroke();
  });

  // 起点标注
  const [sx, sy] = project(cities[startIdx]);
  ctx.fillStyle = '#a5f3fc';
  ctx.font = '12px Consolas, monospace';
  ctx.fillText('起点', sx + 10, sy - 8);

  // 左上信息
  ctx.font = '12px "Segoe UI", "Microsoft YaHei", sans-serif';
  ctx.fillStyle = COLORS.text;
  ctx.fillText(`城市数 ${cities.length}` + (improved ? ' · 本代找到新最优' : ''), 14, 22);
}

/* ---------------- 折线图通用绘制 ---------------- */

function chartFrame(ctx, canvas, pad, yMin, yMax, xCount, xLabel) {
  const W = canvas.width - pad.l - pad.r;
  const H = canvas.height - pad.t - pad.b;
  ctx.strokeStyle = COLORS.grid;
  ctx.lineWidth = 1;
  ctx.font = '11px Consolas, monospace';
  ctx.fillStyle = COLORS.text;

  for (let i = 0; i <= 4; i++) {
    const y = pad.t + (H * i) / 4;
    const val = yMax - ((yMax - yMin) * i) / 4;
    ctx.beginPath();
    ctx.moveTo(pad.l, y);
    ctx.lineTo(pad.l + W, y);
    ctx.stroke();
    ctx.fillText(val.toFixed(0), 8, y + 4);
  }
  const ticks = Math.min(6, Math.max(2, xCount));
  for (let i = 0; i < ticks; i++) {
    const ratio = ticks === 1 ? 0 : i / (ticks - 1);
    const x = pad.l + W * ratio;
    ctx.beginPath();
    ctx.moveTo(x, pad.t);
    ctx.lineTo(x, pad.t + H);
    ctx.stroke();
    ctx.fillText(String(Math.round(ratio * (xCount - 1))), x - 6, pad.t + H + 18);
  }
  ctx.fillStyle = COLORS.axis;
  ctx.fillText(xLabel, pad.l + W - 60, pad.t + H + 30);
  return {
    W, H,
    x: (i, total) => pad.l + (total <= 1 ? W / 2 : (i / (total - 1)) * W),
    y: (v) => pad.t + H - ((v - yMin) / (yMax - yMin || 1)) * H,
    top: () => pad.t,
    bottom: () => pad.t + H,
  };
}

function strokeSeries(ctx, points, color, width) {
  ctx.save();
  ctx.strokeStyle = color;
  ctx.lineWidth = width;
  ctx.lineJoin = 'round';
  ctx.beginPath();
  points.forEach(([x, y], i) => (i === 0 ? ctx.moveTo(x, y) : ctx.lineTo(x, y)));
  ctx.stroke();
  ctx.restore();
}

function drawPlayhead(ctx, frame, x) {
  ctx.save();
  ctx.strokeStyle = 'rgba(226, 232, 240, 0.55)';
  ctx.setLineDash([4, 4]);
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.moveTo(x, frame.top());
  ctx.lineTo(x, frame.bottom());
  ctx.stroke();
  ctx.restore();
}

/* ---------------- 收敛曲线 ---------------- */

function drawConvergence(upto) {
  const canvas = $('convCanvas');
  if (!state.result) { drawEmpty(canvas, '尚未求解'); return; }
  const gens = state.result.generations;
  const ctx = canvas.getContext('2d');
  ctx.clearRect(0, 0, canvas.width, canvas.height);

  let yMin = Infinity, yMax = -Infinity;
  gens.forEach((g) => {
    yMin = Math.min(yMin, g.best);
    yMax = Math.max(yMax, g.avg);
  });
  const optimal = state.instance.optimal || 0;
  if (optimal > 0) yMin = Math.min(yMin, optimal);
  const margin = (yMax - yMin) * 0.08 || 1;
  yMin -= margin;
  yMax += margin;

  const frame = chartFrame(ctx, canvas, { l: 64, r: 20, t: 22, b: 40 }, yMin, yMax, gens.length, '代数');
  const n = Math.min(upto, gens.length - 1);

  // 已知最优水平线
  if (optimal > 0) {
    ctx.save();
    ctx.setLineDash([7, 6]);
    ctx.strokeStyle = COLORS.optimal;
    ctx.lineWidth = 1.6;
    ctx.beginPath();
    ctx.moveTo(frame.x(0, gens.length), frame.y(optimal));
    ctx.lineTo(frame.x(gens.length - 1, gens.length), frame.y(optimal));
    ctx.stroke();
    ctx.restore();
  }

  const allBest = gens.map((g, i) => [frame.x(i, gens.length), frame.y(g.best)]);
  const allAvg = gens.map((g, i) => [frame.x(i, gens.length), frame.y(g.avg)]);

  // 全程淡色轨迹（提供整体走势背景）
  ctx.save();
  ctx.globalAlpha = 0.16;
  strokeSeries(ctx, allAvg, COLORS.avg, 1.4);
  strokeSeries(ctx, allBest, COLORS.best, 1.8);
  ctx.restore();

  // 已走过部分（高亮）
  strokeSeries(ctx, allAvg.slice(0, n + 1), COLORS.avg, 1.6);
  strokeSeries(ctx, allBest.slice(0, n + 1), COLORS.best, 2.4);

  // 播放头
  const px = frame.x(n, gens.length);
  drawPlayhead(ctx, frame, px);

  // 当前代标记
  const [cx, cy] = allBest[n];
  ctx.beginPath();
  ctx.arc(cx, cy, 4.5, 0, Math.PI * 2);
  ctx.fillStyle = COLORS.best;
  ctx.fill();
  ctx.fillStyle = '#e0f2fe';
  ctx.font = '12px Consolas, monospace';
  ctx.fillText(fmt(gens[n].best), Math.min(cx + 8, canvas.width - 90), cy - 8);
}

/* ---------------- 多样性曲线 ---------------- */

function drawDiversity(upto) {
  const canvas = $('divCanvas');
  if (!state.result) { drawEmpty(canvas, '尚未求解'); return; }
  const gens = state.result.generations;
  const ctx = canvas.getContext('2d');
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  const frame = chartFrame(ctx, canvas, { l: 52, r: 20, t: 20, b: 38 }, 0, 1.05, gens.length, '代数');
  const n = Math.min(upto, gens.length - 1);

  const all = gens.map((g, i) => [frame.x(i, gens.length), frame.y(g.diversity)]);

  // 全程淡色
  ctx.save();
  ctx.globalAlpha = 0.16;
  strokeSeries(ctx, all, COLORS.diversity, 2);
  ctx.restore();

  const pts = all.slice(0, n + 1);
  // 面积填充
  ctx.save();
  const grad = ctx.createLinearGradient(0, 0, 0, canvas.height);
  grad.addColorStop(0, 'rgba(168, 85, 247, 0.35)');
  grad.addColorStop(1, 'rgba(168, 85, 247, 0.02)');
  ctx.fillStyle = grad;
  ctx.beginPath();
  ctx.moveTo(pts[0][0], frame.y(0));
  pts.forEach(([x, y]) => ctx.lineTo(x, y));
  ctx.lineTo(pts[pts.length - 1][0], frame.y(0));
  ctx.closePath();
  ctx.fill();
  ctx.restore();

  strokeSeries(ctx, pts, COLORS.diversity, 2);
  drawPlayhead(ctx, frame, frame.x(n, gens.length));
}

/* ---------------- 统计表 ---------------- */

function clearStats() {
  ['sCurrent', 'sBest', 'sOptimal', 'sConv', 'sImprove', 'sTime', 'sParams']
    .forEach((id) => { $(id).textContent = '—'; });
}

function updateStats() {
  const res = state.result;
  if (!res) { clearStats(); return; }
  const i = Math.min(state.gen, res.generations.length - 1);
  $('sCurrent').textContent = fmt(res.generations[i].best);
  $('sBest').textContent = fmt(res.bestDistance);
  const optimal = state.instance.optimal || 0;
  $('sOptimal').textContent = optimal > 0
    ? `${fmt(optimal)}　Gap ${fmt(((res.bestDistance - optimal) / optimal) * 100)}%`
    : '未知（随机实例）';
  $('sConv').textContent = `第 ${res.convergedGen} 代（共 ${res.generations.length} 代）`;
  const init = res.generations[0].best;
  $('sImprove').textContent = `${fmt(init)} → ${fmt(res.bestDistance)}　改善 ${fmt(((init - res.bestDistance) / init) * 100)}%`;
  $('sTime').textContent = `${fmt(res.elapsedMs, 1)} ms`;
  const p = res.params;
  $('sParams').textContent =
    `种群 ${p.population} · 代数 ${p.generations} · 交叉率 ${p.crossoverRate} · 变异率 ${p.mutationRate} · ` +
    `精英 ${p.elitism} · ${p.selection}/${p.crossover}/${p.mutation} · ` +
    `2-opt 局部搜索 ${p.localSearch ? '开' : '关'} · 种子 ${p.seed}`;
}

/* ---------------- 参数对比实验 ---------------- */

async function runScan() {
  if (!state.instance || state.busy) return;
  const values = $('scanValues').value
    .split(/[,，\s]+/)
    .map((s) => parseFloat(s))
    .filter((v) => !Number.isNaN(v));
  if (!values.length) { setStatus('请填写有效的扫描取值', true); return; }

  state.busy = true;
  $('scanBtn').disabled = true;
  $('scanInfo').textContent = '运行中……';
  try {
    const inst = state.instance;
    const data = await api('/api/scan', {
      method: 'POST',
      body: JSON.stringify({
        instance: { name: inst.name, edgeType: inst.edgeType, cities: inst.cities },
        params: readParams(),
        param: $('scanParam').value,
        values,
      }),
    });
    state.scan = data.results;
    drawScan();
    renderScanTable();
    $('scanInfo').textContent = `${data.results.length} 组参数对比完成`;
  } catch (e) {
    $('scanInfo').textContent = '运行失败';
    setStatus(e.message, true);
  } finally {
    state.busy = false;
    $('scanBtn').disabled = false;
  }
}

function drawScan() {
  const canvas = $('scanCanvas');
  const ctx = canvas.getContext('2d');
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  if (!state.scan || !state.scan.length) { drawEmpty(canvas, '尚未运行对比实验'); return; }

  let yMin = Infinity, yMax = -Infinity, maxLen = 0;
  state.scan.forEach((r) => {
    yMin = Math.min(yMin, Math.min(...r.bestPerGen));
    yMax = Math.max(yMax, Math.max(...r.bestPerGen));
    maxLen = Math.max(maxLen, r.bestPerGen.length);
  });
  const optimal = state.instance.optimal || 0;
  if (optimal > 0) yMin = Math.min(yMin, optimal);
  const margin = (yMax - yMin) * 0.08 || 1;
  yMin -= margin;
  yMax += margin;

  const frame = chartFrame(ctx, canvas, { l: 64, r: 20, t: 22, b: 40 }, yMin, yMax, maxLen, '代数');

  if (optimal > 0) {
    ctx.save();
    ctx.setLineDash([7, 6]);
    ctx.strokeStyle = COLORS.optimal;
    ctx.lineWidth = 1.6;
    ctx.beginPath();
    ctx.moveTo(frame.x(0, maxLen), frame.y(optimal));
    ctx.lineTo(frame.x(maxLen - 1, maxLen), frame.y(optimal));
    ctx.stroke();
    ctx.restore();
  }

  state.scan.forEach((r, k) => {
    const color = COLORS.scanPalette[k % COLORS.scanPalette.length];
    const pts = r.bestPerGen.map((v, i) => [frame.x(i, Math.max(r.bestPerGen.length, 1)), frame.y(v)]);
    strokeSeries(ctx, pts, color, 2);
    ctx.fillStyle = color;
    ctx.font = '12px Consolas, monospace';
    const lx = 78 + (k % 4) * 250;
    const ly = 16 + Math.floor(k / 4) * 18;
    ctx.fillText(`${r.label} → ${fmt(r.bestDistance)}`, lx, ly);
  });
}

function renderScanTable() {
  const body = $('scanTable').querySelector('tbody');
  body.innerHTML = '';
  if (!state.scan) return;
  const bestVal = Math.min(...state.scan.map((r) => r.bestDistance));
  const optimal = state.instance.optimal || 0;
  state.scan.forEach((r) => {
    const tr = document.createElement('tr');
    if (r.bestDistance === bestVal) tr.className = 'best-row';
    const gap = optimal > 0 ? `${fmt(((r.bestDistance - optimal) / optimal) * 100)}%` : '—';
    tr.innerHTML =
      `<td>${r.label}</td><td>${fmt(r.bestDistance)}</td>` +
      `<td>${r.convergedGen}</td><td>${fmt(r.elapsedMs, 0)}</td><td>${gap}</td>`;
    body.appendChild(tr);
  });
}

/* ---------------- 事件绑定 ---------------- */

function bindEvents() {
  $('instSelect').addEventListener('change', syncInstanceUI);
  $('loadBuiltinBtn').addEventListener('click', loadInstance);
  $('loadBtn').addEventListener('click', loadInstance);
  $('solveBtn').addEventListener('click', solve);
  $('scanBtn').addEventListener('click', runScan);

  $('playBtn').addEventListener('click', () => {
    if (state.playTimer) stopPlay(); else startPlay();
  });
  $('stepBtn').addEventListener('click', () => { stopPlay(); setGen(state.gen + 1); });
  $('resetBtn').addEventListener('click', () => { stopPlay(); setGen(0); });
  $('jumpBestBtn').addEventListener('click', () => {
    stopPlay();
    if (state.result) setGen(state.result.convergedGen);
  });
  $('jumpEndBtn').addEventListener('click', () => {
    stopPlay();
    if (state.result) setGen(state.result.generations.length - 1);
  });
  $('genSlider').addEventListener('input', (e) => {
    stopPlay();
    setGen(parseInt(e.target.value, 10));
  });
  $('speedSlider').addEventListener('input', (e) => {
    const v = parseInt(e.target.value, 10);
    state.speedMs = Math.round(1000 / v);
    $('speedLabel').textContent = `${state.speedMs}ms/代`;
    if (state.playTimer) startPlay();
  });

  // 键盘快捷键
  document.addEventListener('keydown', (e) => {
    const tag = e.target.tagName;
    if (tag === 'INPUT' || tag === 'SELECT' || tag === 'TEXTAREA') return;
    if (!state.result) return;
    switch (e.key) {
      case ' ':
        e.preventDefault();
        state.playTimer ? stopPlay() : startPlay();
        break;
      case 'ArrowRight':
        e.preventDefault();
        stopPlay();
        setGen(state.gen + 1);
        break;
      case 'ArrowLeft':
        e.preventDefault();
        stopPlay();
        setGen(state.gen - 1);
        break;
      case 'Home':
        e.preventDefault();
        stopPlay();
        setGen(0);
        break;
      case 'End':
        e.preventDefault();
        stopPlay();
        setGen(state.result.generations.length - 1);
        break;
      case 'c':
      case 'C':
        e.preventDefault();
        stopPlay();
        setGen(state.result.convergedGen);
        break;
    }
  });
}

syncInstanceUI();
bindEvents();
drawEmpty($('mapCanvas'), '请先加载实例');
drawEmpty($('convCanvas'), '尚未求解');
drawEmpty($('divCanvas'), '尚未求解');
drawEmpty($('scanCanvas'), '尚未运行对比实验');
$('speedLabel').textContent = `${state.speedMs}ms/代`;
loadInstance();
