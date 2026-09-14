import {$,current,api,notify,text} from './dispatch.js';
import {routeInput} from './route.js';
import {configPanel} from './config.js';
import {chart} from './charts.js';
import {csv,analysisRows,download} from './export.js';
const host=$('analysisContent');
host.innerHTML='<div class="config"><label>实验问题<select id="analysisProblem"><option value="tsp">配送路线 TSP</option><option value="knapsack">智能接单 0/1 背包</option></select></label><label>固定随机种子（最多 10 个）<input id="analysisSeeds" value="7,17,27,37,47" style="width:240px"></label><label>对照模板<select id="analysisPreset"><option value="mutation">变异概率 0.02 / 0.08</option><option value="initialization">初始化方法</option><option value="encoding">TSP 编码方法</option><option value="custom">自定义两组配置</option></select></label></div><p class="muted">路线实验使用路线页当前地点；背包实验使用工作台全部候选订单。每组使用同一组种子，2-opt 关闭。组间保持其他条件一致。</p><div id="analysisConfigs" class="split"></div><div class="actions"><button id="compareRun" class="primary">运行重复实验</button><button id="compareCancel" disabled>取消计算</button><button id="compareCSV" disabled>导出实验 CSV</button><button id="comparePNG" disabled>导出曲线 PNG</button></div><p id="compareStatus" role="status"></p><canvas id="compareChart" width="1000" height="360" aria-label="重复实验平均收敛曲线"></canvas><p id="compareLegend" class="muted"></p><div class="table-wrap"><table><thead><tr><th>配置组</th><th>最佳值</th><th>平均值</th><th>样本标准差</th><th>平均耗时 ms</th></tr></thead><tbody id="compareRows"></tbody></table></div><details><summary>每次运行的结果</summary><div class="table-wrap"><table><thead><tr><th>配置组</th><th>种子</th><th>最终值</th><th>耗时 ms</th></tr></thead><tbody id="compareRuns"></tbody></table></div></details>';
let getters=[],controller=null,result=null,revision=0;
function invalidate(){revision++;controller?.abort();result=null;$('compareCSV').disabled=true;$('comparePNG').disabled=true;$('compareStatus').textContent='输入或配置已变化，请重新运行实验。';}
function build(){
 const kind=$('analysisProblem').value,preset=$('analysisPreset').value;
 $('analysisConfigs').replaceChildren();getters=[];
 for(let i=0;i<2;i++){
  const panel=text('div'),name=document.createElement('input');name.value='配置 '+(i===0?'A':'B');name.setAttribute('aria-label','配置组 '+(i===0?'A':'B')+' 名称');name.oninput=invalidate;panel.append(name);
  const read=configPanel(panel,kind,invalidate);panel.querySelector('details').open=true;
  const set=(key,value)=>{const node=panel.querySelector('[data-param="'+key+'"]');if(node){node.value=value;node.dispatchEvent(new Event('change'));}};
  if(preset==='mutation')set('mutationRate',i===0?0.02:0.08);
  if(preset==='initialization'&&i===1)set('initialization',kind==='tsp'?'mixed':'stratified');
  if(preset==='encoding'&&kind==='tsp'&&i===1)set('encoding','random-key');
  if(kind==='tsp'){const flag=panel.querySelector('input[type="checkbox"]');flag.checked=false;flag.disabled=true;}
  getters.push(()=>({name:name.value.trim(),[kind==='tsp'?'tsp':'knapsack']:read()}));$('analysisConfigs').append(panel);
 }
 invalidate();
}
$('analysisProblem').onchange=()=>{if($('analysisProblem').value!=='tsp'&&$('analysisPreset').value==='encoding')$('analysisPreset').value='mutation';$('analysisPreset').querySelector('option[value="encoding"]').disabled=$('analysisProblem').value!=='tsp';build();};
$('analysisPreset').onchange=build;$('analysisSeeds').oninput=invalidate;
document.addEventListener('orderschanged',invalidate);document.addEventListener('routeinvalid',invalidate);
$('compareCancel').onclick=()=>{controller?.abort();};
$('compareRun').onclick=async()=>{
 const token=revision;result=null;$('compareCSV').disabled=true;$('comparePNG').disabled=true;
 try{
  const problem=$('analysisProblem').value,seeds=$('analysisSeeds').value.split(/[,，\s]+/).filter(Boolean).map(Number);
  if(!seeds.length||seeds.some(s=>!Number.isSafeInteger(s)||s<=0))throw Error('请填写不重复的正整数随机种子');
  const groups=getters.map(read=>read()),request={problem,seeds,groups};
  if(problem==='tsp')request.route=routeInput();else request.selection=current();
  controller=new AbortController();$('compareRun').disabled=true;$('compareCancel').disabled=false;$('compareStatus').textContent='正在执行 '+groups.length+' 组 × '+seeds.length+' 次实验…';
  const r=await api('/api/analysis/compare',request,controller.signal);if(token!==revision)return;result=r;
  const divisor=r.unit==='cents'?100:1,fmt=value=>(value/divisor).toFixed(3),colors=['#24624e','#bc9142'];
  chart($('compareChart'),r.groups.map((g,i)=>({values:g.meanCurve.map(v=>v/divisor),color:colors[i]})),r.unit==='cents'?'平均历史最优收益 / 元（越大越好）':'平均历史最优距离（越小越好）');
  $('compareRows').replaceChildren(...r.groups.map(g=>{const row=document.createElement('tr');for(const v of [g.name,fmt(g.best),fmt(g.mean),fmt(g.stdDev),g.meanElapsedMs.toFixed(2)])row.append(text('td',v));return row;}));
  $('compareRuns').replaceChildren(...r.groups.flatMap(g=>g.runs.map(run=>{const row=document.createElement('tr');for(const v of [g.name,run.seed,fmt(run.value),run.elapsedMs.toFixed(2)])row.append(text('td',v));return row;})));
  $('compareLegend').textContent=r.groups.map((g,i)=>(i===0?'绿色':'金色')+'：'+g.name).join(' · ')+'。曲线为相同代数的重复运行均值；CSV 金额单位为分。';
  $('compareStatus').textContent='已完成 '+r.groups.reduce((s,g)=>s+g.runs.length,0)+' 次运行。样本标准差按 n−1 计算。';
  $('compareCSV').disabled=false;$('comparePNG').disabled=false;notify('重复实验已完成。');
 }catch(e){if(token===revision){$('compareStatus').textContent=e.name==='AbortError'?'实验已取消，没有生成新结果。':e.message;if(e.name!=='AbortError')notify(e.message,true);}}
 finally{controller=null;$('compareRun').disabled=false;$('compareCancel').disabled=true;}
};
$('compareCSV').onclick=()=>{if(result)download(csv(analysisRows(result)),'text/csv;charset=utf-8','骑迹-重复实验.csv');};
$('comparePNG').onclick=()=>{
 if(!result)return;const canvas=document.createElement('canvas');canvas.width=1000;canvas.height=445;
 const c=canvas.getContext('2d');c.fillStyle='#fafbf6';c.fillRect(0,0,1000,445);c.drawImage($('compareChart'),0,55);c.font='16px sans-serif';c.fillStyle='#19382f';c.fillText('骑迹 · '+(result.input.problem==='tsp'?'配送路线':'智能接单')+'重复实验',20,28);c.font='12px sans-serif';c.fillText($('compareLegend').textContent,20,430);canvas.toBlob(blob=>{if(blob)download(blob,'image/png','骑迹-平均收敛曲线.png');},'image/png');
};
build();$('compareStatus').textContent='选择配置后运行实验，结果将包含每次运行的原始数据。';

