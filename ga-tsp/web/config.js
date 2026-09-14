import {text} from './dispatch.js';
export const defaultTSP=()=>({population:100,generations:200,crossoverRate:0.9,mutationRate:0.02,elitism:2,seed:7,selection:'tournament',crossover:'ox',mutation:'inversion',encoding:'permutation',initialization:'random',localSearch:false});
export const defaultBag=()=>({population:100,generations:200,crossoverRate:0.9,mutationRate:0.03,elitism:2,seed:7,selection:'tournament',crossover:'onepoint',mutation:'bitflip',initialization:'random',greedyRepair:true});
export function configPanel(parent,kind,onchange=()=>{}){
 const details=document.createElement('details');details.append(text('summary','高级算法设置'));
 const form=text('div','','config');details.append(form);parent.append(details);
 const initial=kind==='tsp'?defaultTSP():defaultBag(),inputs={};
 function field(key,label,options){
  const node=text('label',label),input=document.createElement(options?'select':'input');
  input.dataset.param=key;input.setAttribute('aria-label',label);
  if(options){for(const [value,name] of options){const o=text('option',name);o.value=value;input.append(o);}}
  else{input.type='number';input.step=['crossoverRate','mutationRate'].includes(key)?'0.01':'1';}
  input.value=initial[key];input.onchange=()=>{if(key==='encoding')sync();onchange();};node.append(input);form.append(node);inputs[key]=input;
 }
 for(const [key,label]of [['population','种群规模'],['generations','进化代数'],['crossoverRate','交叉概率'],['mutationRate','变异概率'],['elitism','精英保留'],['seed','随机种子']])field(key,label);
 field('initialization','初始化',kind==='tsp'?[['random','随机'],['mixed','50% 最近邻 + 50% 随机']]:[['random','独立随机'],['stratified','分层均匀']]);
 field('selection','选择方法',kind==='tsp'?[['tournament','锦标赛'],['roulette','轮盘赌']]:[['tournament','锦标赛'],['rank','排序轮盘赌']]);
 if(kind==='tsp')field('encoding','编码方法',[['permutation','排列编码'],['random-key','随机键编码']]);
 field('crossover','交叉方法',kind==='tsp'?[['ox','OX'],['pmx','PMX']]:[['onepoint','单点交叉'],['uniform','均匀交叉']]);
 field('mutation','变异方法',kind==='tsp'?[['inversion','逆转'],['swap','交换'],['insert','插入']]:[['bitflip','位翻转'],['swap','交换']]);
 const flag=kind==='tsp'?'localSearch':'greedyRepair',label=text('label',kind==='tsp'?'启用 2-opt':'启用超容量修复'),checkbox=document.createElement('input');checkbox.type='checkbox';checkbox.checked=initial[flag];checkbox.onchange=onchange;label.append(checkbox);form.append(label);inputs[flag]=checkbox;
 function options(key,values){inputs[key].replaceChildren(...values.map(([v,n])=>{const o=text('option',n);o.value=v;return o;}));}
 function sync(){const keys=inputs.encoding.value==='random-key';options('crossover',keys?[['uniform','均匀交叉']]:[['ox','OX'],['pmx','PMX']]);options('mutation',keys?[['reset','随机重置']]:[['inversion','逆转'],['swap','交换'],['insert','插入']]);}
 details.append(text('p',kind==='tsp'?'随机键按键值排序，同值按地点编号排序。切换编码会切换适配算子，效果差异不能全部归因于编码；随机键变异概率作用于每个基因，排列变异概率作用于每个个体。':'二进制编码；变异概率作用于每个基因。关闭修复时不可行个体适应度为零，业务结果仍会重新校验容量。','muted'));
 return ()=>{
  const params={};for(const [key,input]of Object.entries(inputs))params[key]=input.type==='checkbox'?input.checked:input.type==='number'?Number(input.value):input.value;
  if(!Number.isSafeInteger(params.seed))throw Error('随机种子须为安全整数');
  for(const key of ['population','generations','elitism'])if(!Number.isInteger(params[key]))throw Error('种群、代数和精英数量须为整数');
  for(const key of ['crossoverRate','mutationRate'])if(!Number.isFinite(params[key])||params[key]<0||params[key]>1)throw Error('概率须在 0～1 之间');
  return params;
 };
}

