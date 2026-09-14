export function chart(canvas,series,label='目标值'){
 const ctx=canvas.getContext('2d'),w=canvas.width,h=canvas.height;
 ctx.clearRect(0,0,w,h);ctx.fillStyle='#fafbf6';ctx.fillRect(0,0,w,h);
 const values=series.flatMap(s=>s.values).filter(Number.isFinite);
 if(!values.length){ctx.fillStyle='#6e8075';ctx.fillText('运行后显示曲线',25,30);return;}
 const min=Math.min(...values),max=Math.max(...values),span=max-min||1;
 ctx.font='12px sans-serif';ctx.fillStyle='#6e8075';ctx.fillText(label,16,20);
 for(let j=0;j<5;j++){
  const y=36+j*(h-76)/4;ctx.strokeStyle='#e2e8dc';ctx.beginPath();ctx.moveTo(65,y);ctx.lineTo(w-20,y);ctx.stroke();
  ctx.fillText((max-j*span/4).toFixed(1),8,y+4);
 }
 for(const s of series){ctx.strokeStyle=s.color;ctx.lineWidth=2;ctx.beginPath();s.values.forEach((v,i)=>{const x=65+i*(w-85)/Math.max(1,s.values.length-1),y=36+(max-v)/span*(h-76);if(i===0)ctx.moveTo(x,y);else ctx.lineTo(x,y);});ctx.stroke();}
 ctx.fillStyle='#6e8075';ctx.fillText('第 1 代',65,h-12);ctx.fillText('代数 →',w-85,h-12);
}
export function playback(container,frames,draw){
 container.replaceChildren();let index=0,timer=null;
 const button=document.createElement('button'),range=document.createElement('input'),label=document.createElement('span');
 const step=document.createElement('button'),first=document.createElement('button'),last=document.createElement('button'),speed=document.createElement('select');
 button.textContent='播放';first.textContent='首代';last.textContent='末代';step.textContent='下一代';
 speed.setAttribute('aria-label','回放速度');
 for(const [v,n]of [[500,'慢速'],[160,'正常'],[40,'快速']]){const o=document.createElement('option');o.value=v;o.textContent=n;speed.append(o);}speed.value='160';
 range.type='range';range.min=0;range.max=Math.max(0,frames.length-1);range.value=0;range.setAttribute('aria-label','回放代数');
 const show=()=>{range.value=index;label.textContent=(index+1)+' / '+frames.length+' 代';if(frames.length)draw(frames[index],index);};
 const stop=()=>{clearInterval(timer);timer=null;button.textContent='播放';};
 const play=()=>{stop();button.textContent='暂停';timer=setInterval(()=>{if(index>=frames.length-1){stop();return;}index++;show();},Number(speed.value));};
 button.onclick=()=>{if(timer)stop();else{if(index===frames.length-1)index=0;show();play();}};
 range.oninput=()=>{stop();index=Number(range.value);show();};
 step.onclick=()=>{stop();index=Math.min(index+1,frames.length-1);show();};
 first.onclick=()=>{stop();index=0;show();};last.onclick=()=>{stop();index=frames.length-1;show();};
 speed.onchange=()=>{if(timer)play();};
 for(const node of [button,first,step,last,range,speed])node.disabled=!frames.length;
 container.append(button,first,step,last,range,speed,label);if(frames.length)show();else label.textContent='直接判定，无进化回放';
 return stop;
}

