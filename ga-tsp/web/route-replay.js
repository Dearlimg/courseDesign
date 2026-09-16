const $ = id => document.getElementById(id);
const svg = (tag, attrs, text = '') => {
  const el = document.createElementNS('http://www.w3.org/2000/svg', tag);
  for (const [key, value] of Object.entries(attrs)) el.setAttribute(key, value);
  el.textContent = text; return el;
};
export class RouteReplay {
  constructor(onFrame) {
    this.onFrame = onFrame; this.frames = []; this.index = 0;
    $('evoPlay').onclick = () => this.timer ? this.pause() : this.play();
    $('evoStep').onclick = () => { this.pause(); this.show(this.index + 1); };
    $('evoReset').onclick = () => { this.pause(); this.show(0); };
    $('evoEnd').onclick = () => { this.pause(); this.show(this.frames.length - 1); };
    $('evoSeek').oninput = () => { this.pause(); this.show(Number($('evoSeek').value)); };
    $('evoSpeed').onchange = () => { if (this.timer) { this.pause(); this.play(); } };
    document.addEventListener('visibilitychange', () => { if (document.hidden) this.pause(); });
  }
  clear() { this.pause(); this.frames = []; $('evolution').hidden = true; }
  load(plan) {
    this.clear(); this.frames = plan.routeFrames || [];
    if (!this.frames.length) return;
    $('evolution').hidden = false;
    $('evoSeek').max = this.frames.length - 1;
    this.show(this.frames.length - 1);
  }
  pause() { clearInterval(this.timer); this.timer = null; $('evoPlay').textContent = '播放'; }
  play() {
    if (this.frames.length < 2) return;
    if (this.index === this.frames.length - 1) this.show(0);
    $('evoPlay').textContent = '暂停';
    this.timer = setInterval(() => {
      this.show(this.index + 1);
      if (this.index === this.frames.length - 1) this.pause();
    }, Number($('evoSpeed').value));
  }
  show(index) {
    if (!this.frames.length) return;
    this.index = Math.max(0, Math.min(index, this.frames.length - 1));
    const frame = this.frames[this.index], first = this.frames[0];
    const previous = this.frames[Math.max(0, this.index - 1)];
    $('evoSeek').value = this.index;
    const improvement = first.distanceMeters ? (1 - frame.distanceMeters / first.distanceMeters) * 100 : 0;
    $('evoStats').replaceChildren(...[
      ['当前代数', String(frame.gen)], ['历史最佳路程', frame.distanceMeters.toFixed(1) + ' 米'],
      ['路线得分', frame.fitness.toFixed(4)], ['较初始缩短', improvement.toFixed(2) + '%'],
    ].map(([label, value]) => {
      const box = document.createElement('div'); box.className = 'stat';
      const small = document.createElement('small'); small.textContent = label;
      const strong = document.createElement('strong'); strong.textContent = value;
      box.append(small, strong); return box;
    }));
    const change = previous.distanceMeters - frame.distanceMeters;
    $('evoStatus').textContent = frame.phase + ' · ' +
      (change > 0.000001 ? '发现更短路线，减少 ' + change.toFixed(1) + ' 米' : '保留当前最佳路线') +
      (this.index === this.frames.length - 1 ? ' · 已到最终结果' : ' · 地图与停靠顺序展示当前回放帧');
    this.chart('evoDistance', 'distanceMeters', '#b58313');
    this.chart('evoFitness', 'fitness', '#248879');
    this.onFrame(frame);
  }
  chart(id, key, color) {
    const el = $(id), values = this.frames.map(f => f[key]);
    const low = Math.min(...values), high = Math.max(...values), range = high - low || 1;
    const x = i => 66 + 410 * i / Math.max(1, values.length - 1);
    const y = v => 125 - 95 * (v - low) / range;
    el.replaceChildren(svg('path', {d:'M66 20V130H476',fill:'none',stroke:'#ddd'}));
    for (const [value, yy] of [[high,30],[low,125]]) el.append(svg('text',{x:4,y:yy,'font-size':11,fill:'#777'},value.toFixed(key === 'fitness' ? 3 : 0)));
    el.append(svg('polyline',{points:values.slice(0,this.index+1).map((v,i)=>x(i)+','+y(v)).join(' '),fill:'none',stroke:color,'stroke-width':2.5}));
    el.append(svg('circle',{cx:x(this.index),cy:y(values[this.index]),r:4,fill:color}));
    el.append(svg('text',{x:66,y:154,'font-size':11},'初始种群'),svg('text',{x:424,y:154,'font-size':11},'最终择优'));
  }
}
