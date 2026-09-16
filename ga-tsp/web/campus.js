import { RouteReplay } from './route-replay.js';
import { csv, download } from './export.js';
import { CampusMap } from './campus-map.js';
let atlas, replay, focusedPlace = 0;

const $ = id => document.getElementById(id);
const money = value => (value / 100).toFixed(2);
const node = (tag, value, className = '') => {
  const el = document.createElement(tag); el.textContent = value; el.className = className; return el;
};
const svgNode = (tag, attrs = {}, value = '') => {
  const el = document.createElementNS('http://www.w3.org/2000/svg', tag);
  for (const [key, v] of Object.entries(attrs)) el.setAttribute(key, String(v));
  el.textContent = value; return el;
};
let map, batch, result, comparison, storageKey, busy = false, controller;
const settings = ['mode','capacityGrams','maxMinutes','speedKph','costKm','costMinute','seed','population','generations'];
const tell = (message, error = false) => { $('notice').textContent = message; $('notice').classList.toggle('error', error); };

async function request(url, body, signal) {
  const response = await fetch(url, {
    method: body === undefined ? 'GET' : 'POST', credentials: 'same-origin', signal,
    headers: body === undefined ? {} : {'Content-Type':'application/json'},
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (response.status === 401) { location.replace('/auth.html'); throw Error('登录已失效'); }
  const text = await response.text();
  let data = {};
  try { data = text ? JSON.parse(text) : {}; }
  catch { throw Error(`接口 ${url} 未返回 JSON（HTTP ${response.status}），请确认后端已重新编译启动。`); }
  if (!response.ok) throw Error(data.error || '服务暂不可用');
  return data;
}
function updateButtons() {
  $('savePlan').disabled = busy || !result;
  $('exportPlan').disabled = busy || !result;
  $('exportComparison').disabled = busy || !comparison;
}
async function job(message, action) {
  if (busy) return;
  replay?.pause();
  busy = true; controller = new AbortController();
  document.querySelectorAll('main button, main input, main select').forEach(el => { el.disabled = true; });
  $('cancel').hidden = false; $('cancel').disabled = false;
  tell(message);
  try { await action(controller.signal); }
  catch (e) { tell(e.name === 'AbortError' ? '已取消操作。' : e.message, e.name !== 'AbortError'); }
  finally {
    busy = false;
    document.querySelectorAll('main button, main input, main select').forEach(el => { el.disabled = false; });
    $('cancel').hidden = true; updateButtons();
  }
}
function persist() {
  try {
    localStorage.setItem(storageKey, JSON.stringify({batch, settings:Object.fromEntries(settings.map(id => [id,$(id).value]))}));
  } catch { tell('本机存储不可用；可使用服务端保存或导出。',true); }
}
function invalidate() {
  replay?.clear();
  if (result || comparison) { $('planStatus').textContent = '输入已变化，旧结果已失效，请重新规划。'; $('planStatus').className = 'stale'; }
  result = undefined; comparison = undefined;
  for (const id of ['metrics','selectedOrders','excludedOrders','comparisonRows','comparisonRuns','curve']) $(id).replaceChildren();
  $('exactResult').textContent = ''; $('curveLabel').textContent = '';
  $('stopList').replaceChildren(node('li','生成方案后显示'));
  resetCrate();
  updateButtons(); drawMap(); persist();
}
function input() {
  if (!batch) throw Error('请先生成或载入订单。');
  return {
    batch,mode:$('mode').value,capacityGrams:Number($('capacityGrams').value),maxMinutes:Number($('maxMinutes').value),
    speedKph:Number($('speedKph').value),costCentsPerKm:Math.round(Number($('costKm').value)*100),
    costCentsPerMinute:Math.round(Number($('costMinute').value)*100),seed:Number($('seed').value),
    population:Number($('population').value),generations:Number($('generations').value),
  };
}
function applyInput(req) {
  batch = structuredClone(req.batch);
  for (const id of ['mode','capacityGrams','maxMinutes','speedKph','seed','population','generations']) $(id).value = req[id];
  $('costKm').value = money(req.costCentsPerKm); $('costMinute').value = money(req.costCentsPerMinute);
  modeHelp(); renderOrders(); persist();
}
function modeHelp() {
  $('modeHelp').textContent = {
    business:'目标：配送收入 − 道路里程成本 − 配送时间成本；同时满足载重与时长上限。允许不出车。',
    knapsack:'仅按载重选择收入最高的组合，用动态规划验证；配送成本与时长不参与选单，结果仍会展示这些指标。',
    route:'访问所有订单的送达点并返站；合并同址停靠，以道路距离优化顺序。载重与时长只作提示。',
  }[$('mode').value];
}
function batchSummary() {
  $('batchInfo').textContent = batch ? `${batch.orders.length} 笔订单 · 配送收入 ¥${money(batch.orders.reduce((s,o) => s+o.deliveryFeeCents,0))} · 数据种子 ${batch.seed} · ${batch.mapVersion}` : '暂无订单';
}
// 餐品样例：仅用于演示展示，按订单编号稳定派生（不参与计算，也不写入后端）
const MENU = [
  {name:'珍珠奶茶',icon:'🧋',spec:'中杯 · 少冰三分糖',tone:'#c2843f',soft:'#f7e6d1',tags:['饮品','现制']},
  {name:'香辣鸡腿堡套餐',icon:'🍔',spec:'双层 · 加辣',tone:'#d4821f',soft:'#fbe6c8',tags:['主食','热食']},
  {name:'芋泥波波茶',icon:'🧋',spec:'大杯 · 常温',tone:'#a97bb0',soft:'#f0e4f2',tags:['饮品','现制']},
  {name:'脆皮炸鸡桶',icon:'🍗',spec:'六块装 · 微辣',tone:'#c9682f',soft:'#f9dfcd',tags:['主食','热食']},
  {name:'牛肉拉面',icon:'🍜',spec:'大碗 · 免葱',tone:'#b4834a',soft:'#f5e7d4',tags:['主食','汤面']},
  {name:'冰美式咖啡',icon:'☕',spec:'大杯 · 少冰',tone:'#8a6a4f',soft:'#eee3d8',tags:['饮品','提神']},
  {name:'大份薯条',icon:'🍟',spec:'大份 · 番茄酱',tone:'#d3a027',soft:'#faecc6',tags:['小食','热食']},
  {name:'奥尔良鸡排饭',icon:'🍱',spec:'标准份 · 加饭',tone:'#b06b3a',soft:'#f6e2d2',tags:['主食','套餐']},
  {name:'杨枝甘露',icon:'🥭',spec:'中杯 · 少糖',tone:'#dfa62a',soft:'#fbeec8',tags:['饮品','现制']},
  {name:'现烤蛋挞',icon:'🥧',spec:'两只装',tone:'#c89238',soft:'#f9ead0',tags:['甜点','现烤']},
  {name:'照烧鸡肉卷',icon:'🌯',spec:'标准份 · 不辣',tone:'#bd8b3f',soft:'#f7ebd5',tags:['主食','便携']},
  {name:'冰可乐',icon:'🥤',spec:'大杯 · 加冰',tone:'#9c5a5a',soft:'#f4dede',tags:['饮品','冰饮']},
];
const hash = text => { let h = 2166136261; for (let i=0;i<text.length;i++) { h ^= text.charCodeAt(i); h = Math.imul(h,16777619); } return h >>> 0; };
const menuFor = order => MENU[hash(String(order.id)) % MENU.length];
const portion = grams => grams <= 400 ? '轻食份' : grams <= 900 ? '标准份' : '加量装';
const placeName = order => map?.places?.[order.destinationId]?.name || '未知地点';
// 单笔配送距离：取餐点沿道路到该送达点的米数（后端 depotMeters，不可达为 -1）
const meters = value => value === undefined || value < 0 ? '—' : value >= 1000 ? `${(value / 1000).toFixed(2)} km` : `${Math.round(value)} m`;
const distanceOf = order => (map?.depotMeters || [])[order.destinationId];
function renderOrders() {
  const container = $('campusOrders'); container.replaceChildren();
  const orders = batch?.orders || [];
  if (!orders.length) {
    container.append(node('p','还没有订单：点击「随机生成订单」生成 20～50 笔候选，或用「＋ 新增订单」手工添加，也可从历史记录载入。','empty'));
    batchSummary(); drawMap(); return;
  }
  const labels = {id:'订单编号',destinationId:'送达位置',weightGrams:'重量 / 克',deliveryFeeCents:'配送收入 / 元',serviceSeconds:'交付 / 秒'};
  for (const order of orders) {
    const dish = menuFor(order);
    const card = document.createElement('article');
    card.className = 'order-card'; card.dataset.orderId = order.id;
    card.style.setProperty('--tone', dish.tone); card.style.setProperty('--soft', dish.soft);
    const top = document.createElement('div'); top.className = 'top';
    const title = document.createElement('div'); title.className = 'title';
    const name = node('div', dish.name, 'name');
    const spec = node('div', `${portion(order.weightGrams)} · ${dish.spec}`, 'spec');
    const tags = document.createElement('div'); tags.className = 'tags';
    for (const tag of dish.tags) tags.append(node('span', tag));
    const idTag = node('span', `#${order.id}`);
    tags.append(idTag);
    title.append(name, spec, tags);
    const price = document.createElement('div'); price.className = 'price';
    const feeText = document.createTextNode(`¥${money(order.deliveryFeeCents)}`);
    price.append(feeText, node('small','配送费'));
    top.append(node('span', dish.icon, 'thumb'), title, price);
    const stats = document.createElement('div'); stats.className = 'stats';
    const metric = (label, value) => { const box = node('div','','stat'); const v = node('b', value, 'v'); box.append(node('span', label, 'k'), v); return [box, v]; };
    const [weightBox, weightValue] = metric('标准重量', `${order.weightGrams} g`);
    const [feeBox, feeValue] = metric('价值收益', `¥${money(order.deliveryFeeCents)}`);
    const [distanceBox, distanceValue] = metric('距取餐点约', meters(distanceOf(order)));
    stats.append(weightBox, feeBox, distanceBox);
    const foot = document.createElement('div'); foot.className = 'foot';
    const place = node('button', `送达 ${placeName(order)} ↗`, 'place');
    place.title = '在校园地图中定位';
    place.onclick = () => { atlas.focusPlace(order.destinationId); $('route').scrollIntoView({behavior:'smooth'}); };
    const remove = node('button','删除','remove');
    remove.onclick = () => { batch.orders = batch.orders.filter(o => o !== order); renderOrders(); invalidate(); };
    foot.append(place, remove);
    const fields = document.createElement('div'); fields.className = 'fields';
    for (const key of ['id','destinationId','weightGrams','deliveryFeeCents','serviceSeconds']) {
      const label = document.createElement('label');
      if (key === 'id' || key === 'destinationId') label.className = 'wide';
      label.append(node('span',labels[key]));
      const el = document.createElement(key === 'destinationId' ? 'select' : 'input');
      el.setAttribute('aria-label', `${order.id} ${labels[key]}`);
      if (key === 'destinationId') {
        for (const place of map.places) { const option = node('option',place.name); option.value=place.id; option.selected = place.id === order.destinationId; el.append(option); }
      } else {
        el.type = key === 'id' ? 'text' : 'number'; el.required = true;
        if (key === 'id') el.maxLength = 60;
        else {
          el.min = key === 'serviceSeconds' ? '0' : key === 'deliveryFeeCents' ? '0.01' : '1';
          el.step = key === 'deliveryFeeCents' ? '0.01' : '1';
          el.max = key === 'serviceSeconds' ? '600' : key === 'deliveryFeeCents' ? '1000' : '10000';
          el.value = key === 'deliveryFeeCents' ? money(order[key]) : order[key];
        }
        if (key === 'id') el.value = order.id;
      }
      el.addEventListener('input', () => {
        if (key === 'destinationId') order.destinationId = Number(el.value);
        else order[key] = key === 'id' ? el.value.trim() : key === 'deliveryFeeCents' ? Math.round(Number(el.value)*100) : Number(el.value);
        if (key === 'id') { card.dataset.orderId = order.id; idTag.textContent = `#${order.id}`; }
        if (key === 'weightGrams') { weightValue.textContent = `${order.weightGrams} g`; spec.textContent = `${portion(order.weightGrams)} · ${dish.spec}`; }
        if (key === 'deliveryFeeCents') { feeValue.textContent = `¥${money(order.deliveryFeeCents)}`; feeText.nodeValue = `¥${money(order.deliveryFeeCents)}`; }
        if (key === 'destinationId') { place.textContent = `送达 ${placeName(order)}`; distanceValue.textContent = meters(distanceOf(order)); }
        invalidate(); batchSummary();
      });
      label.append(el); fields.append(label);
    }
    const tune = document.createElement('details'); tune.className = 'tune';
    tune.append(node('summary','调整参数'), fields);
    card.append(top, stats, foot, tune);
    container.append(card);
  }
  batchSummary(); drawMap();
}
function drawMap() {
  atlas?.update(batch,result);
  if (map) showPlace(map.places[focusedPlace]);
}
function showPlace(place) {
  focusedPlace = place.id;
  $('mapPlace').value = place.id;
  $('mapPlaceName').textContent = place.name;
  const orders = (batch?.orders || []).filter(order => order.destinationId === place.id);
  const chosen = (result?.selected || []).filter(order => order.destinationId === place.id);
  $('mapPlaceInfo').textContent = '距取餐点约 ' + Math.round(map.depotMeters[place.id]) + ' 米 · ' + orders.length + ' 笔候选订单 · ' + chosen.length + ' 笔已选。' + (orders.length ? ' 订单：' + orders.map(order => order.id).join('、') : '');
}
function setupMap() {
  for (const place of map.places) {
    const option = node('option',place.name); option.value=place.id; $('mapPlace').append(option);
  }
  atlas = new CampusMap($('campusMap'),map,showPlace);
  replay = new RouteReplay(frame => {
    atlas.update(batch, {...result, stops:frame.stops, legs:frame.legs});
    renderStops({...result, stops:frame.stops, legs:frame.legs});
  });
  $('mapZoomIn').onclick=()=>atlas.zoom(1.35);
  $('mapZoomOut').onclick=()=>atlas.zoom(1/1.35);
  $('mapFit').onclick=()=>atlas.fit();
  $('mapLocate').onclick=()=>atlas.focusPlace(Number($('mapPlace').value));
  $('mapOnlyOrders').onchange=()=>{atlas.onlyOrders=$('mapOnlyOrders').checked;atlas.render();};
  let savedMapView;
  const toggleExpanded = () => {
    const expanded=$('mapShell').classList.toggle('map-expanded');
    if (expanded) { savedMapView={...atlas.view}; atlas.zoom(2.5); }
    else if (savedMapView) { atlas.view=savedMapView; atlas.applyView(); }
    $('mapExpand').textContent=expanded?'收起大图':'展开大图';
    $('mapExpand').setAttribute('aria-expanded',String(expanded));
  };
  $('mapExpand').onclick=toggleExpanded;
  document.addEventListener('keydown',event=>{if(event.key==='Escape' && $('mapShell').classList.contains('map-expanded'))toggleExpanded();});
  $('mapAddOrder').onclick=()=>{
    batch ||= {mapVersion:map.version,seed:0,scenario:'manual',orders:[]};
    if(batch.orders.length>=50)return tell('最多支持 50 笔校园订单。',true);
    batch.orders.push({id:'M'+Date.now(),destinationId:focusedPlace,weightGrams:500,deliveryFeeCents:600,serviceSeconds:60});
    invalidate();renderOrders();tell('已添加到 '+map.places[focusedPlace].name+' 的订单。');
  };
}
// ===== 配送箱：自动选单后，选中的卡片依次飞入箱中 =====
const crateBox = () => $('crate');
const calmMotion = () => window.matchMedia('(prefers-reduced-motion: reduce)').matches;
function crateStamp(done, total) {
  const badge = $('crateBadge');
  badge.hidden = false; badge.textContent = String(done);
  $('crateCount').textContent = `已装袋 ${done} 单 / 共 ${total} 单`;
  if (!calmMotion()) badge.animate([{ transform: 'scale(1)' }, { transform: 'scale(1.32)' }, { transform: 'scale(1)' }], { duration: 240, easing: 'ease-out' });
}
function resetCrate() {
  $('crateBadge').hidden = true; $('crateBadge').textContent = '0';
  $('crateItems').replaceChildren(); $('crateItems').hidden = true;
  $('crateToggle').setAttribute('aria-expanded', 'false');
  $('crateCount').textContent = '空箱待命';
  crateBox().dataset.state = 'empty';
  document.querySelectorAll('#campusOrders .order-card.packed').forEach(card => card.classList.remove('packed'));
}
async function packCrate(selected) {
  const cards = new Map([...document.querySelectorAll('#campusOrders .order-card')].map(card => [card.dataset.orderId, card]));
  const target = $('crateToggle'), list = $('crateItems');
  crateBox().dataset.state = 'packing';
  let done = 0;
  const fly = async (order, delay) => {
    const card = cards.get(order.id), dish = menuFor(order);
    if (card && !calmMotion()) {
      const from = card.getBoundingClientRect(), to = target.getBoundingClientRect();
      const visible = from.bottom > 0 && from.top < innerHeight && from.right > 0 && from.left < innerWidth;
      if (visible) {
        const ghost = card.cloneNode(true);
        ghost.querySelectorAll('details').forEach(el => el.removeAttribute('open'));
        ghost.classList.add('flying');
        Object.assign(ghost.style, { left: `${from.left}px`, top: `${from.top}px`, width: `${from.width}px`, height: `${from.height}px` });
        document.body.append(ghost);
        const dx = to.left + to.width / 2 - (from.left + from.width / 2);
        const dy = to.top + to.height / 2 - (from.top + from.height / 2);
        await ghost.animate([
          { transform: 'translate(0,0) scale(1) rotate(0deg)', opacity: 1, offset: 0 },
          { transform: `translate(${dx * .55}px, ${dy * .55 - 34}px) scale(.62) rotate(-3deg)`, opacity: .92, offset: .5 },
          { transform: `translate(${dx}px, ${dy}px) scale(.06) rotate(2deg)`, opacity: 0, offset: 1 },
        ], { duration: 440, delay, easing: 'cubic-bezier(.42,.05,.32,1)', fill: 'both' }).finished.catch(() => {});
        ghost.remove();
      }
    }
    card?.classList.add('packed');
    const chip = document.createElement('div'); chip.className = 'chip';
    chip.append(node('span', dish.icon), node('b', order.id), node('small', `¥${money(order.deliveryFeeCents)}`));
    list.append(chip);
    done += 1; crateStamp(done, selected.length);
  };
  for (let i = 0; i < selected.length; i += 10) {
    const group = selected.slice(i, i + 10);
    group.forEach((order, index) => fly(order, index * 35));
    await new Promise(resolve => setTimeout(resolve, group.length * 35 + 440));
  }
  crateBox().dataset.state = done ? 'packed' : 'empty';
  $('crateCount').textContent = done ? `已装袋 ${done} 单 · 点击查看` : '空箱待命';
  if (done && !calmMotion()) target.animate([{ transform: 'translateY(0)' }, { transform: 'translateY(-5px)' }, { transform: 'translateY(0)' }], { duration: 300, easing: 'ease-out' });
}
function renderResult(plan) {
  result = plan; $('planStatus').className = '';
  const title = plan.input.mode === 'knapsack' ? (plan.verifiedOptimal ? '已达到标准背包最优收入。' : `背包 DP 最优收入 ¥${money(plan.optimalIncomeCents)}；本次 GA 尚未达到。`) : plan.input.mode === 'route' ? '已规划全部送达点的道路路线。' : '当前搜索得到的最佳可行方案。';
  $('planStatus').textContent = `${title} ${plan.metrics.feasible?'满足载重和时长约束。':'超出载重或时长，仅作实验对照。'} 用时 ${plan.elapsedMs.toFixed(0)} ms。`;
  const m=plan.metrics;
  $('metrics').replaceChildren(...[
    ['配送收入',`¥${money(m.incomeCents)}`,'所选订单的骑手配送费'],['预计成本',`¥${money(m.costCents)}`,'里程成本 + 时间成本'],
    ['预计净收益',`¥${money(m.netCents)}`,`${plan.selected.length} 笔订单`],['配送里程',`${(m.distanceMeters/1000).toFixed(2)} km`,'沿校园路网估算，含返站'],
    ['预计耗时',`${m.minutes.toFixed(1)} 分钟`,`上限 ${plan.input.maxMinutes} 分钟，含交付`],['餐箱载重',`${m.weightGrams} g`,`上限 ${plan.input.capacityGrams} g`],
  ].map(([label,value,note]) => { const el=node('div','','stat'); el.append(node('small',label),node('strong',value),node('small',note)); return el; }));
  $('selectedOrders').replaceChildren(...plan.selected.map(o => node('span',`${menuFor(o).icon} ${o.id} · ${map.places[o.destinationId].name}`)));
  if (!plan.selected.length) $('selectedOrders').append(node('p','当前方案不配送任何订单。'));
  $('excludedOrders').replaceChildren(...plan.excluded.map(o => node('div',`${menuFor(o).icon} ${o.id}：${o.reason}`)));
  resetCrate();
  if (plan.selected.length) packCrate(plan.selected);
  drawMap(); drawCurve(plan); updateButtons();
  renderStops(plan);
  replay.load(plan);
}
function renderStops(plan) {
  const stops=plan.stops.length > 1 ? [...plan.stops,0] : [0];
  $('stopList').replaceChildren(...stops.map((id,index) => {
    const orders=plan.selected.filter(o => o.destinationId===id).map(o=>o.id).join('、');
    const item=node('li','');
    const button=node('button',map.places[id].name+(index===stops.length-1 && index>0?' · 返站':''));
    button.onclick=()=>{atlas.focusPlace(id,false);if(index>0)atlas.focusLeg(index-1);};
    item.append(button);
    if(index>0 && plan.legs?.[index-1])item.append(node('small','本段约 '+Math.round(plan.legs[index-1].distanceMeters)+' 米'));
    if(orders && !(index===stops.length-1 && index>0))item.append(node('small',orders));
    return item;
  }));
}
function drawCurve(plan) {
  const values=plan.curve,svg=$('curve'); svg.replaceChildren();
  $('curveLabel').textContent=plan.input.mode==='route' ? '最近邻 + 2-opt 与最终路线的距离（米）；保留较短路线。' : `${plan.input.mode==='business'?'净收益':'配送收入'}历史最佳曲线（分），最后一点包含候选集路线优化。`;
  if (!values.length) return;
  const low=Math.min(...values),high=Math.max(...values),range=high-low || 1;
  svg.append(svgNode('polyline',{points:values.map((v,i)=>`${60+i*710/Math.max(1,values.length-1)},${175-(v-low)/range*140}`).join(' '),fill:'none',stroke:'#b28b16','stroke-width':3}));
  svg.append(svgNode('text',{x:10,y:30},high.toFixed(0)),svgNode('text',{x:10,y:180},low.toFixed(0)),svgNode('text',{x:60,y:208},'开始'),svgNode('text',{x:720,y:208},'最终'));
}
function renderComparison(data) {
  comparison=data;
  $('comparisonRows').replaceChildren(...data.groups.map(group => {
    const row=document.createElement('tr');
    for (const value of [group.name,money(group.bestNetCents),money(group.meanNetCents),money(group.stdDevNetCents),`${group.feasibleRuns}/${group.runs.length}`]) row.append(node('td',value));
    return row;
  }));
  $('comparisonRuns').replaceChildren(...data.groups.flatMap(group=>group.runs.map(run=>node('p',`${group.name} · 种子 ${run.seed}：收入 ¥${money(run.metrics.incomeCents)}，成本 ¥${money(run.metrics.costCents)}，净收益 ¥${money(run.metrics.netCents)}，${(run.metrics.distanceMeters/1000).toFixed(2)} km，${run.metrics.minutes.toFixed(1)} 分钟，${run.metrics.feasible?'可行':'超出约束'}，${run.elapsedMs.toFixed(0)} ms`))));
  $('exactResult').textContent=data.exactNetCents === undefined ? '超过十单，未计算业务精确最优值；以上为近似算法的重复实验结果。' : `小规模精确业务最优净收益：¥${money(data.exactNetCents)}。`;
  updateButtons();
}
async function refreshHistory(signal) {
  const rows=await request(`/api/campus/history?kind=${$('historyKind').value}`,undefined,signal);
  $('historyList').replaceChildren(...rows.map(row=>{ const option=node('option',`#${row.id} · ${new Date(row.createdAt).toLocaleString()}`);option.value=row.id;return option; }));
  if (!rows.length) {const option=node('option','暂无保存记录');option.value='';$('historyList').append(option);}
}
function bind() {
  for (const id of settings) $(id).addEventListener('input',()=>{modeHelp();invalidate();});
  $('cancel').onclick=()=>controller?.abort();
  $('generate').onclick=()=>job('正在生成校园订单…',async signal=>{
    batch=await request('/api/campus/batches/generate',{count:Number($('count').value),seed:Number($('dataSeed').value),scenario:$('scenario').value},signal);
    invalidate();renderOrders();tell(`已生成 ${batch.orders.length} 单。数据种子 ${batch.seed} 可用于复现。`);
  });
  $('addOrder').onclick=()=>{
    batch ||= {mapVersion:map.version,seed:0,scenario:'manual',orders:[]};
    if (batch.orders.length>=50) return tell('最多支持 50 笔校园订单。',true);
    batch.orders.push({id:'M'+Date.now(),destinationId:1,weightGrams:500,deliveryFeeCents:600,serviceSeconds:60});
    invalidate();renderOrders();
  };
  $('solve').onclick=()=>job('正在评估订单组合与道路路线…',async signal=>{
    const data=await request('/api/campus/plan',input(),signal);renderResult(data);tell('配送方案已生成。');
  });
  $('saveBatch').onclick=()=>job('正在保存订单批次…',async signal=>{
    if (!batch) throw Error('请先生成订单');
    const saved=await request('/api/campus/batches/save',batch,signal);tell(`批次已保存为 #${saved.id}。`);
  });
  $('savePlan').onclick=()=>job('正在复算并保存方案快照…',async signal=>{
    if (!result) throw Error('请先规划');
    const saved=await request('/api/campus/plans/save',result.input,signal);renderResult(saved.payload);tell(`方案已保存为 #${saved.id}。`);
  });
  $('refreshHistory').onclick=()=>job('正在读取历史记录…',async signal=>{await refreshHistory(signal);tell('历史记录已更新。');});
  $('historyKind').onchange=()=>{$('historyList').replaceChildren();};
  $('crateToggle').onclick=()=>{ const items=$('crateItems'); items.hidden=!items.hidden; $('crateToggle').setAttribute('aria-expanded',String(!items.hidden)); };
  $('loadHistory').onclick=()=>job('正在载入快照…',async signal=>{
    const id=$('historyList').value;if (!/^\d+$/.test(id)) throw Error('请先刷新并选择记录');
    const row=await request(`/api/campus/history/${id}`,undefined,signal);
    const savedBatch=row.kind==='plan'?row.payload.input.batch:row.payload;
    if (savedBatch.mapVersion!==map.version) throw Error('历史记录的地图版本与当前地图不一致');
    invalidate();
    if (row.kind==='plan') { applyInput(row.payload.input);renderResult(row.payload); }
    else { batch=row.payload;renderOrders();persist(); }
    tell(`已载入 #${row.id}。`);
  });
  $('compare').onclick=()=>job('正在运行三种策略的五次对照实验…',async signal=>{
    const data=await request('/api/campus/compare',{plan:input(),seeds:[7,17,27,37,47]},signal);
    renderComparison(data);tell('对照实验已完成。');
  });
  $('exportPlan').onclick=()=>{if(result) download(JSON.stringify(result,null,2),'application/json','campus-plan.json');};
  $('exportComparison').onclick=()=>{
    if(!comparison)return;
    const rows=[['校园配送对照实验'],['输入快照',JSON.stringify(comparison.input)],['统计说明','全部运行均计入统计；可行性单独列出；金额为分'],['策略','种子','收入/分','成本/分','净收益/分','距离/米','时长/分钟','载重/克','可行','耗时/ms','曲线单位','曲线']];
    for(const g of comparison.groups)for(const r of g.runs)rows.push([g.name,r.seed,r.metrics.incomeCents,r.metrics.costCents,r.metrics.netCents,r.metrics.distanceMeters,r.metrics.minutes,r.metrics.weightGrams,r.metrics.feasible,r.elapsedMs,r.curveUnit,JSON.stringify(r.curve)]);
    download(csv(rows),'text/csv;charset=utf-8','campus-comparison.csv');
  };
  $('logout').onclick=()=>job('正在退出…',async signal=>{await request('/api/auth/logout',{},signal);location.replace('/auth.html');});
}
async function start() {
  try {
    const {user}=await request('/api/auth/me');$('accountName').textContent=user.username;
    storageKey=`simple_tuan.user.${user.id}.campus.v1`;
    map=await request('/api/campus/map');$('mapNotice').textContent=map.notice;
    if (!Array.isArray(map.nodes) || !map.width || !map.places.every(place=>Number.isInteger(place.nodeId))) {
      throw Error('校园地图需要新版后端，请重新编译并启动 Go 服务。');
    }
    setupMap();
    try {
      const saved=JSON.parse(localStorage.getItem(storageKey));
      if(saved?.batch?.mapVersion===map.version && Array.isArray(saved.batch.orders) && saved.batch.orders.length<=50) {
        batch=saved.batch;
        for(const id of settings)if(saved.settings?.[id]!==undefined)$(id).value=saved.settings[id];
      } else if(saved?.batch) { tell('地图已升级为长安校区西区，旧版地点编号不能直接沿用，请重新生成订单。'); }
    } catch { tell('本机数据读取失败，请重新生成订单。',true); }
    bind();modeHelp();renderOrders();updateButtons();
    document.body.dataset.auth='ready';$('authLoading').hidden=true;
  } catch(e) { $('authLoading').textContent=e.message || '工作台加载失败，请刷新重试。'; }
}
start();
