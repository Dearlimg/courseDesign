'use strict';
const $ = id => document.getElementById(id);
const kind = new URLSearchParams(location.search).get('problem') === 'cec' ? 'cec' : 'knapsack';
const isCEC = kind === 'cec';
const colors = ['#238268', '#bc8746', '#8075ad', '#5e92a6', '#ba7383', '#7d9750', '#a28a70', '#4c6268'];
const state = { instance: null, landscape: null, result: null, scan: null, gen: 0, timer: null, busy: false, snapshot: null };
const fmt = v => !Number.isFinite(v) ? '—' : (Math.abs(v) > 0 && Math.abs(v) < 0.001 ? v.toExponential(3) : v.toLocaleString('zh-CN', { maximumFractionDigits: 3 }));
async function api(url, body) {
  const response = await fetch(url, body ? { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) } : {});
  const data = await response.json();
  if (!response.ok) throw new Error(data.error || '请求失败，请稍后重试');
  return data;
}
function status(text, error = false) { $('status').textContent = text; $('status').classList.toggle('error', error); }
function busy(value) {
  state.busy = value;
  document.querySelectorAll('#config input,#config select,#config button,#config textarea,#scan,#scanParam,#scanValues').forEach(el => { el.disabled = value; });
  $('status').classList.toggle('busy', value);
}
function empty(id, text) {
  const canvas = $(id), ctx = canvas.getContext('2d');
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  ctx.fillStyle = '#8b9e8d'; ctx.textAlign = 'center'; ctx.font = '16px "Microsoft YaHei", sans-serif';
  ctx.fillText(text, canvas.width / 2, canvas.height / 2); ctx.textAlign = 'left';
}
function stop() { clearInterval(state.timer); state.timer = null; $('play').textContent = '播放'; }
function reset() {
  stop(); state.result = null; state.scan = null; state.gen = 0; state.snapshot = null;
  ['play', 'first', 'step', 'last', 'timeline', 'export'].forEach(id => { $(id).disabled = true; });
  $('timeline').value = 0; $('timeline').max = 0;
  ['metricGen', 'metricBest', 'metricAvg', 'metricAux'].forEach(id => { $(id).textContent = '—'; });
  $('runTag').textContent = '等待开始'; $('scanStatus').textContent = '尚未运行';
  $('summary').innerHTML = '<tr><th>求解状态</th><td>等待实验运行</td></tr>';
  $('solution').textContent = '尚未求解'; $('scanLegend').replaceChildren();
  $('scanRows').innerHTML = '<tr><td colspan="5" class="empty-cell">运行对比实验后，这里将展示各组结果</td></tr>';
  empty('curve', '开始进化后，查看最优解与平均值的变化');
  empty('scanCurve', '调整参数，探索不同的收敛表现');
}
function options(id, values) {
  $(id).replaceChildren(...values.map(([value, label]) => new Option(label, value)));
}
function setup() {
  document.querySelector(`[data-problem="${kind}"]`).setAttribute('aria-current', 'page');
  $('bagConfig').hidden = isCEC; $('bagView').hidden = isCEC;
  $('cecConfig').hidden = !isCEC; $('cecView').hidden = !isCEC;
  $('capacity').required = !isCEC; $('itemCount').required = !isCEC; $('instanceSeed').required = !isCEC;
  options('crossover', isCEC ? [['arithmetic', '算术交叉'], ['blend', '混合交叉']] : [['onepoint', '单点交叉'], ['uniform', '均匀交叉']]);
  options('mutation', isCEC ? [['gaussian', '高斯变异'], ['reset', '随机重置']] : [['bitflip', '位翻转变异'], ['swap', '交换变异']]);
  if (isCEC) {
    document.title = 'CEC 测试 · 进化实验室';
    $('title').innerHTML = '穿越函数地形，<span>寻找全局最优。</span>';
    $('eyebrow').textContent = '03 / 连续函数优化';
    $('subtitle').textContent = '在单峰与多峰之间，观察种群如何探索、聚集与收敛。';
    $('goal').textContent = '最小化目标函数值'; $('encoding').textContent = '实数编码 · CEC 2005 基准子集';
    $('visualTitle').textContent = '函数地形与种群轨迹'; $('bestLabel').textContent = '当代最优值'; $('auxLabel').textContent = '最优误差';
    $('mutationRate').value = '0.1';
    $('curveNote').textContent = '纵轴为目标函数值（越小越好）；使用原始值而非误差对数。实线为当代最优与种群平均，虚线为历史最优（单调累积进度）。所有曲线基于全部维度计算。';
    $('method').textContent = '实数编码直接表示各维坐标。初始种群可采用独立随机或分层均匀抽样；锦标赛与排序轮盘赌决定亲本；算术或混合交叉产生后代，高斯或随机重置变异增加探索能力。越界坐标截断回合法范围，精英保留优秀个体。变异概率作用于每个基因。';
    $('reference').innerHTML = '采用 CEC 2005 的 F1、F2、F9 原始移位定义及偏置；二维用于教学，提供 10、30、50 维实验。这里不是完整竞赛测试套件。定义参考 Suganthan 等（2005）《实参数优化问题定义与评价准则》，<a href="https://github.com/thieu1995/opfunu/tree/master/opfunu/cec_based/data_2005" target="_blank" rel="noopener">移位数据来源</a>。';
  } else {
    document.title = '0/1 背包 · 进化实验室';
    $('method').textContent = '每件物品对应一个二进制基因：1 表示选入，0 表示舍弃。交叉与变异后，按价值密度从低到高移除物品直到满足容量约束。适应度为可行方案的总价值。动态规划独立计算精确最优值，仅用作评价基准，不参与种群进化。变异概率作用于每个基因。';
    $('reference').textContent = '可编辑物品重量、价值和背包容量；最多 100 件物品，容量上限 10000。固定实例种子生成相同物品，固定算法种子复现实验。零概率和零精英数均按输入值执行。';
  }
  reset();
}
async function loadInstance() {
  if (state.busy) return;
  busy(true); status('正在加载实例…');
  try {
    let instance, landscape;
    if (isCEC) {
      const data = await api(`/api/cec/landscape?function=${$('function').value}&dimension=${$('dimension').value}`);
      instance = data.instance; landscape = data.grid;
    } else {
      instance = await api(`/api/knapsack/instance?n=${$('itemCount').value}&seed=${$('instanceSeed').value}`);
    }
    reset(); state.instance = instance; state.landscape = landscape;
    if (isCEC) {
      const b = instance.benchmark;
      $('functionInfo').textContent = `${b.description} 搜索范围 [${b.lower}, ${b.upper}]，理论最优 ${b.optimum}。${b.formula}`;
      $('projectionNote').textContent = instance.dimension === 2 ? '二维真实函数地形。点的位置为种群前两维，连线为最优个体的移动轨迹。' : `${instance.dimension} 维实验：热力图为第 3～${instance.dimension} 维固定在理论最优点的二维切片；种群点仅为前两维投影，其完整函数值由全部维度决定。`;
      drawLandscape();
    } else {
      $('capacity').value = instance.capacity;
      $('itemInput').value = instance.items.map(item => `${item.weight},${item.value}`).join('\n');
      drawBag();
    }
    status('实例已就绪，可以开始进化。');
  } catch (error) { status(error.message, true); }
  finally { busy(false); }
}
function applyItems() {
  if (state.busy) return;
  try {
    const items = $('itemInput').value.trim().split(/\n/).filter(line => line.trim()).map((line, i) => {
      const values = line.trim().split(/[,，\s]+/).map(Number);
      if (values.length !== 2 || values.some(n => !Number.isInteger(n) || n < 1)) throw new Error(`第 ${i + 1} 行须为两个正整数：重量,价值`);
      if (values[0] > 10000 || values[1] > 100000) throw new Error('重量上限 10000，价值上限 100000');
      return { weight: values[0], value: values[1] };
    });
    if (items.length < 2 || items.length > 100) throw new Error('物品数量须在 2～100 之间');
    const capacity = Number($('capacity').value);
    if (!Number.isInteger(capacity) || capacity < 1 || capacity > 10000) throw new Error('容量须为 1～10000 的整数');
    reset(); state.instance = { items, capacity }; $('itemCount').value = items.length;
    drawBag(); status('物品数据已应用。');
  } catch (error) { status(error.message, true); }
}
function request() {
  const params = {};
  ['population', 'generations', 'crossoverRate', 'mutationRate', 'elitism', 'seed'].forEach(id => { params[id] = Number($(id).value); });
  ['selection', 'initialization', 'crossover', 'mutation'].forEach(id => { params[id] = $(id).value; });
  params.greedyRepair = $('greedyRepair').checked;
  if (Object.values(params).some(v => typeof v === 'number' && !Number.isFinite(v))) throw new Error('请填写有效参数');
  if (!state.instance) throw new Error('请先加载问题实例');
  if (isCEC) {
    if (state.instance.benchmark.id !== $('function').value || state.instance.dimension !== Number($('dimension').value)) throw new Error('函数切换未成功，请重新选择后加载');
    return { problem: kind, function: state.instance.benchmark.id, dimension: state.instance.dimension, params };
  }
  if (!Number.isInteger(Number($('capacity').value))) throw new Error('背包容量须为整数');
  return { problem: kind, knapsack: { items: state.instance.items, capacity: Number($('capacity').value) }, params };
}
async function solve(event) {
  event.preventDefault(); if (state.busy) return;
  try {
    if (!$('config').reportValidity()) return;
    const payload = request();
    stop(); busy(true); status('正在进化，请稍候…');
    const result = await api('/api/experiments/solve', payload);
    state.result = result; state.snapshot = payload;
    if (!isCEC) state.instance.capacity = payload.knapsack.capacity;
    $('timeline').max = result.generations.length - 1;
    ['play', 'first', 'step', 'last', 'timeline', 'export'].forEach(id => { $(id).disabled = false; });
    summary(); setGen(0); play();
    status(`进化完成，共 ${result.generations.length} 代，评估 ${result.evaluations} 次，用时 ${fmt(result.elapsedMs)} 毫秒。`);
  } catch (error) { status(error.message, true); }
  finally { busy(false); }
}
function drawBag(genes = []) {
  if (!state.instance) return;
  let weight = 0;
  const nodes = state.instance.items.map((item, i) => {
    const selected = genes[i] === 1;
    if (selected) weight += item.weight;
    const card = document.createElement('div'); card.className = `item-card${selected ? ' selected' : ''}`;
    card.setAttribute('aria-label', `物品 ${i + 1}，重量 ${item.weight}，价值 ${item.value}，${selected ? '已选' : '未选'}`);
    card.innerHTML = `<b>物品 ${String(i + 1).padStart(2, '0')}</b><strong>${item.value}</strong><small>重量 ${item.weight} · 价值 ${item.value}</small>`;
    return card;
  });
  $('items').replaceChildren(...nodes);
  $('capacityLabel').textContent = `${weight} / ${state.instance.capacity}`;
  $('capacityFill').style.width = `${Math.min(100, weight / state.instance.capacity * 100)}%`;
  $('metricAux').textContent = `${fmt(weight / state.instance.capacity * 100)}%`;
}
let terrainCanvas = null, terrainKey = '';
function drawLandscape(generation) {
  if (!state.landscape) return;
  const canvas = $('landscape'), ctx = canvas.getContext('2d'), b = state.instance.benchmark;
  const pad = { x: 52, y: 22, w: canvas.width - 78, h: canvas.height - 65 };
  const grid = state.landscape, key = `${b.id}-${state.instance.dimension}`;
  if (key !== terrainKey) {
    terrainKey = key; terrainCanvas = document.createElement('canvas'); terrainCanvas.width = grid.length; terrainCanvas.height = grid.length;
    const tc = terrainCanvas.getContext('2d'), max = Math.max(...grid.flat().map(v => Math.log1p(Math.max(0, v - b.optimum))));
    grid.forEach((row, y) => row.forEach((v, x) => {
      const t = Math.log1p(Math.max(0, v - b.optimum)) / (max || 1);
      tc.fillStyle = `rgb(${Math.round(235 - 170 * t)},${Math.round(245 - 126 * t)},${Math.round(232 - 138 * t)})`; tc.fillRect(x, y, 1, 1);
    }));
  }
  ctx.clearRect(0, 0, canvas.width, canvas.height); ctx.drawImage(terrainCanvas, pad.x, pad.y, pad.w, pad.h);
  const point = genes => [pad.x + (genes[0] - b.lower) / (b.upper - b.lower) * pad.w, pad.y + (b.upper - genes[1]) / (b.upper - b.lower) * pad.h];
  ctx.font = '12px "Microsoft YaHei",sans-serif'; ctx.fillStyle = '#6c806f';
  for (let i = 0; i <= 4; i++) {
    const v = b.lower + i / 4 * (b.upper - b.lower);
    ctx.fillText(fmt(v), pad.x + pad.w * i / 4 - 8, pad.y + pad.h + 20);
    ctx.fillText(fmt(v), 4, pad.y + pad.h * (1 - i / 4) + 4);
  }
  ctx.fillText('第一维', canvas.width - 63, canvas.height - 5); ctx.fillText('第二维', 4, 14);
  if (generation) {
    ctx.strokeStyle = '#f0cf89'; ctx.lineWidth = 1.6; ctx.beginPath();
    state.result.generations.slice(0, state.gen + 1).forEach((g, i) => { const [x, y] = point(g.genes); i ? ctx.lineTo(x, y) : ctx.moveTo(x, y); }); ctx.stroke();
    generation.samples.forEach(genes => {
      const [x, y] = point(genes); ctx.beginPath(); ctx.arc(x, y, 4, 0, Math.PI * 2); ctx.fillStyle = '#e1f8ee'; ctx.fill(); ctx.strokeStyle = '#246c58'; ctx.lineWidth = 1; ctx.stroke();
    });
    const [x, y] = point(generation.genes); ctx.beginPath(); ctx.arc(x, y, 6, 0, Math.PI * 2); ctx.fillStyle = '#e5b759'; ctx.fill(); ctx.strokeStyle = '#fff8de'; ctx.stroke();
  }
  const [ox, oy] = point(state.instance.shift); ctx.strokeStyle = '#fff'; ctx.lineWidth = 2; ctx.beginPath(); ctx.moveTo(ox - 7, oy); ctx.lineTo(ox + 7, oy); ctx.moveTo(ox, oy - 7); ctx.lineTo(ox, oy + 7); ctx.stroke();
}
function chart(id, series, cursor = null) {
  const canvas = $(id), ctx = canvas.getContext('2d'); ctx.clearRect(0, 0, canvas.width, canvas.height);
  const all = series.flatMap(s => s.values), count = Math.max(...series.map(s => s.values.length));
  if (!all.length) return;
  let lo = Math.min(...all), hi = Math.max(...all); const margin = (hi - lo) * .1 || Math.max(1, Math.abs(hi) * .01); lo -= margin; hi += margin;
  const left = 90, top = 25, width = canvas.width - 115, height = canvas.height - 68;
  const x = i => left + i / Math.max(1, count - 1) * width, y = v => top + (hi - v) / (hi - lo) * height;
  ctx.font = '12px "Microsoft YaHei",sans-serif'; ctx.fillStyle = '#809180'; ctx.lineWidth = 1;
  for (let i = 0; i < 5; i++) {
    const yy = top + height * i / 4, value = hi - (hi - lo) * i / 4;
    ctx.strokeStyle = '#e5ebe4'; ctx.beginPath(); ctx.moveTo(left, yy); ctx.lineTo(left + width, yy); ctx.stroke();
    ctx.fillText(Math.abs(value) > 99999 ? value.toExponential(1) : fmt(value), 4, yy + 4);
    ctx.fillText(String(Math.round((count - 1) * i / 4)), left + width * i / 4 - 6, top + height + 23);
  }
  series.forEach(s => { ctx.strokeStyle = s.color; ctx.lineWidth = 2.3; ctx.beginPath(); s.values.forEach((v, i) => { i ? ctx.lineTo(x(i), y(v)) : ctx.moveTo(x(i), y(v)); }); ctx.stroke(); });
  if (cursor !== null) { ctx.setLineDash([4, 4]); ctx.strokeStyle = '#83988a'; ctx.lineWidth = 1; ctx.beginPath(); ctx.moveTo(x(cursor), top); ctx.lineTo(x(cursor), top + height); ctx.stroke(); ctx.setLineDash([]); }
  ctx.fillText('进化代数', canvas.width - 75, canvas.height - 5);
}
// drawCurve 专门用于「02 / 收敛分析」：实线画当代最优与平均（反映 GA 探索过程），
// 虚线画历史最优作为参考（数学上单调，仅显示进度），并用 cursor 标记当前回放代次。
function drawCurve(id, best, avg, soFar, cursor = null) {
  const canvas = $(id), ctx = canvas.getContext('2d'); ctx.clearRect(0, 0, canvas.width, canvas.height);
  const all = [...best, ...avg, ...soFar], count = best.length;
  if (!all.length) return;
  let lo = Math.min(...all), hi = Math.max(...all); const margin = (hi - lo) * .1 || Math.max(1, Math.abs(hi) * .01); lo -= margin; hi += margin;
  const left = 90, top = 25, width = canvas.width - 115, height = canvas.height - 68;
  const x = i => left + i / Math.max(1, count - 1) * width, y = v => top + (hi - v) / (hi - lo) * height;
  ctx.font = '12px "Microsoft YaHei",sans-serif'; ctx.fillStyle = '#809180'; ctx.lineWidth = 1;
  for (let i = 0; i < 5; i++) {
    const yy = top + height * i / 4, value = hi - (hi - lo) * i / 4;
    ctx.strokeStyle = '#e5ebe4'; ctx.beginPath(); ctx.moveTo(left, yy); ctx.lineTo(left + width, yy); ctx.stroke();
    ctx.fillText(Math.abs(value) > 99999 ? value.toExponential(1) : fmt(value), 4, yy + 4);
    ctx.fillText(String(Math.round((count - 1) * i / 4)), left + width * i / 4 - 6, top + height + 23);
  }
  // 历史最优（虚线参考，单调累积进度）
  ctx.save(); ctx.setLineDash([6, 5]); ctx.strokeStyle = colors[3]; ctx.lineWidth = 1.8;
  ctx.beginPath(); soFar.forEach((v, i) => { i ? ctx.lineTo(x(i), y(v)) : ctx.moveTo(x(i), y(v)); }); ctx.stroke(); ctx.restore();
  // 当代最优（实线，主曲线）
  ctx.strokeStyle = colors[0]; ctx.lineWidth = 2.3; ctx.beginPath();
  best.forEach((v, i) => { i ? ctx.lineTo(x(i), y(v)) : ctx.moveTo(x(i), y(v)); }); ctx.stroke();
  // 种群平均（实线，次曲线）
  ctx.strokeStyle = colors[1]; ctx.lineWidth = 2.3; ctx.beginPath();
  avg.forEach((v, i) => { i ? ctx.lineTo(x(i), y(v)) : ctx.moveTo(x(i), y(v)); }); ctx.stroke();
  if (cursor !== null) { ctx.setLineDash([4, 4]); ctx.strokeStyle = '#83988a'; ctx.lineWidth = 1; ctx.beginPath(); ctx.moveTo(x(cursor), top); ctx.lineTo(x(cursor), top + height); ctx.stroke(); ctx.setLineDash([]); }
  ctx.fillText('进化代数', canvas.width - 75, canvas.height - 5);
}
function setGen(gen) {
  if (!state.result) return;
  state.gen = Math.max(0, Math.min(gen, state.result.generations.length - 1));
  const g = state.result.generations[state.gen];
  $('timeline').value = state.gen; $('metricGen').textContent = `${state.gen} / ${state.result.generations.length - 1}`;
  $('metricBest').textContent = fmt(g.best); $('metricAvg').textContent = fmt(g.average);
  $('runTag').textContent = `第 ${state.gen} 代`;
  if (isCEC) { $('metricAux').textContent = fmt(Math.abs(g.best - state.result.optimal)); drawLandscape(g); } else drawBag(g.genes);
  // 主曲线展示当代最优（能看到探索-波动-收敛过程），叠加历史最优作为虚线参考
  const bestCur = state.result.generations.map(g => g.best);
  const avgCur = state.result.generations.map(g => g.average);
  const soFarCur = state.result.generations.map(g => g.bestSoFar);
  drawCurve('curve', bestCur, avgCur, soFarCur, state.gen);
}
function play() {
  if (!state.result) return;
  stop(); if (state.gen === state.result.generations.length - 1) setGen(0);
  $('play').textContent = '暂停';
  state.timer = setInterval(() => { if (state.gen >= state.result.generations.length - 1) { stop(); return; } setGen(state.gen + 1); }, Number($('speed').value));
}
function summary() {
  const r = state.result;
  const rows = [['历史最优值', fmt(r.best)], ['理论最优值', fmt(r.optimal)], ['距最优差值', fmt(Math.abs(r.best - r.optimal))], ['最后改善代数', `第 ${r.convergedGen} 代`], ['目标函数评估次数', fmt(r.evaluations)], ['求解耗时', `${fmt(r.elapsedMs)} 毫秒`]];
  $('summary').replaceChildren(...rows.map(([label, value]) => { const tr = document.createElement('tr'); const th = document.createElement('th'), td = document.createElement('td'); th.textContent = label; td.textContent = value; tr.append(th, td); return tr; }));
  $('solution').textContent = isCEC ? r.genes.map((v, i) => `第 ${i + 1} 维 = ${v.toPrecision(8)}`).join('； ') : `选中物品：${r.genes.flatMap((v, i) => v === 1 ? [i + 1] : []).join('、') || '无'}。二进制编码：${r.genes.join('')}`;
}
async function scan() {
  if (state.busy || !$('config').reportValidity()) return;
  try {
    const payload = request();
    const tokens = $('scanValues').value.trim().split(/[,，\s]+/);
    const values = tokens.map(Number);
    if (!tokens[0] || values.length > 8 || values.some(v => !Number.isFinite(v))) throw new Error('请填写 1～8 个有效的对比取值');
    busy(true); stop(); $('scanStatus').textContent = '正在运行…'; status('正在运行参数对比实验…');
    const data = await api('/api/experiments/scan', { ...payload, scanParam: $('scanParam').value, values });
    state.scan = { ...data, request: payload, param: $('scanParam').value };
    chart('scanCurve', data.results.map((r, i) => ({ values: r.curve, color: colors[i] })));
    $('scanLegend').replaceChildren(...data.results.map((r, i) => { const s = document.createElement('span'); s.style.color = colors[i]; s.textContent = `${$('scanParam').selectedOptions[0].text} = ${r.value}`; return s; }));
    const bestError = Math.min(...data.results.map(r => r.error));
    $('scanRows').replaceChildren(...data.results.map(r => { const tr = document.createElement('tr'); if (r.error === bestError) tr.className = 'best-row'; [r.value, r.best, r.error, r.convergedGen, r.elapsedMs].forEach(v => { const td = document.createElement('td'); td.textContent = fmt(v); tr.append(td); }); return tr; }));
    $('scanStatus').textContent = `${data.results.length} 组实验完成`; status('参数对比已完成，最佳结果已高亮。');
  } catch (error) { $('scanStatus').textContent = '运行失败'; status(error.message, true); }
  finally { busy(false); }
}
function exportData() {
  if (!state.result) return;
  const rows = [['代数', '当代最优', '种群平均', '历史最优', '独特个体占比', '最优基因']];
  state.result.generations.forEach(g => rows.push([g.gen, g.best, g.average, g.bestSoFar, g.diversity, g.genes.join(';')]));
  rows.push([], ['问题类型', kind], ['请求参数', JSON.stringify(state.snapshot)], ['理论最优', state.result.optimal], ['求解耗时（毫秒）', state.result.elapsedMs]);
  if (state.scan) { rows.push([], ['对比实验参数', JSON.stringify(state.scan.request), state.scan.param], ['参数值', '最优值', '误差', '最后改善代数', '耗时（毫秒）']); state.scan.results.forEach(r => rows.push([r.value, r.best, r.error, r.convergedGen, r.elapsedMs])); }
  const csv = '\uFEFF' + rows.map(row => row.map(v => '"' + String(v).replaceAll('"', '""') + '"').join(',')).join('\r\n');
  const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }));
  const a = document.createElement('a'); a.href = url; a.download = `${isCEC ? 'CEC测试' : '01背包'}-实验数据.csv`; a.click(); setTimeout(() => URL.revokeObjectURL(url), 1000);
}
setup();
$('config').addEventListener('submit', solve);
$('generate').addEventListener('click', loadInstance);
$('applyItems').addEventListener('click', applyItems);
['function', 'dimension'].forEach(id => $(id).addEventListener('change', loadInstance));
$('capacity').addEventListener('change', () => { if (!state.instance || state.busy || !$('capacity').checkValidity()) return; reset(); state.instance.capacity = Number($('capacity').value); drawBag(); });
$('play').addEventListener('click', () => state.timer ? stop() : play());
$('first').addEventListener('click', () => { stop(); setGen(0); });
$('step').addEventListener('click', () => { stop(); setGen(state.gen + 1); });
$('last').addEventListener('click', () => { stop(); setGen(state.result.generations.length - 1); });
$('timeline').addEventListener('input', e => { stop(); setGen(Number(e.target.value)); });
$('speed').addEventListener('change', () => { if (state.timer) play(); });
$('scan').addEventListener('click', scan);
$('scanParam').addEventListener('change', () => { $('scanValues').value = { mutationRate: '0.01, 0.03, 0.08, 0.15', crossoverRate: '0.5, 0.7, 0.9, 1', population: '40, 80, 120, 200', elitism: '0, 1, 2, 5' }[$('scanParam').value]; });
$('export').addEventListener('click', exportData);
window.addEventListener('pagehide', stop);
loadInstance();
