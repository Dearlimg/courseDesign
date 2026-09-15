// All geometry comes from the same versioned dataset used by the route solver.
const NS = 'http://www.w3.org/2000/svg';
const svg = (tag, attrs = {}, text = '') => {
  const element = document.createElementNS(NS, tag);
  for (const [key, value] of Object.entries(attrs)) element.setAttribute(key, String(value));
  element.textContent = text;
  return element;
};

export class CampusMap {
  constructor(element, data, onPick) {
    this.element = element;
    this.map = data;
    this.onPick = onPick;
    this.batch = undefined;
    this.result = undefined;
    this.activePlace = undefined;
    this.activeLeg = undefined;
    this.onlyOrders = false;
    this.view = { x: 0, y: 0, width: data.width, height: data.height };
    this.bindGestures();
    this.render();
  }

  update(batch, result) {
    this.batch = batch;
    this.result = result;
    this.activeLeg = undefined;
    this.render();
  }

  fit() {
    this.view = { x: 0, y: 0, width: this.map.width, height: this.map.height };
    this.applyView();
  }

  zoom(factor) {
    const width = Math.max(this.map.width / 5, Math.min(this.map.width, this.view.width / factor));
    const height = width * this.map.height / this.map.width;
    this.view = { x: this.view.x + (this.view.width-width)/2, y: this.view.y + (this.view.height-height)/2, width, height };
    this.applyView();
  }

  applyView() {
    const v = this.view;
    v.x = Math.max(0, Math.min(this.map.width-v.width, v.x));
    v.y = Math.max(0, Math.min(this.map.height-v.height, v.y));
    this.element.setAttribute('viewBox', `${v.x} ${v.y} ${v.width} ${v.height}`);
    this.element.classList.toggle('map-zoomed',this.map.width/v.width>1.05);
    document.getElementById('mapZoomLevel').textContent = `${Math.round(this.map.width/v.width*100)}%`;
  }

  focusPlace(id, zoom = true) {
    const place = this.map.places[id];
    if (!place) return;
    this.activePlace = id;
    this.activeLeg = undefined;
    if (zoom) {
      this.view.width = this.map.width / 2;
      this.view.height = this.map.height / 2;
      this.view.x = place.x - this.view.width / 2;
      this.view.y = place.y - this.view.height / 2;
    }
    this.render();
    this.onPick(place);
  }

  focusLeg(index) {
    this.activeLeg = index;
    this.render();
  }

  bindGestures() {
    let drag;
    this.element.addEventListener('pointerdown', event => {
      if (event.button !== 0 || event.target.closest('[data-place]')) return;
      const ctm = this.element.getScreenCTM();
      if (!ctm) return;
      drag = {x:event.clientX,y:event.clientY,view:{...this.view},scale:ctm.a};
      this.element.setPointerCapture(event.pointerId);
      this.element.classList.add('dragging');
    });
    this.element.addEventListener('pointermove', event => {
      if (!drag) return;
      this.view.x = drag.view.x-(event.clientX-drag.x)/drag.scale;
      this.view.y = drag.view.y-(event.clientY-drag.y)/drag.scale;
      this.applyView();
    });
    const end = () => { drag=undefined; this.element.classList.remove('dragging'); };
    this.element.addEventListener('pointerup', end);
    this.element.addEventListener('pointercancel', end);
    this.element.addEventListener('wheel', event => {
      if (!event.ctrlKey) return;
      event.preventDefault();
      this.zoom(event.deltaY<0?1.2:1/1.2);
    }, {passive:false});
    this.element.addEventListener('keydown', event => {
      if (event.target !== this.element) return;
      if (event.key==='+' || event.key==='=') this.zoom(1.25);
      else if (event.key==='-') this.zoom(0.8);
      else if (event.key==='Home') this.fit();
      else if (event.key.startsWith('Arrow')) {
        const step = this.view.width/8;
        this.view.x += event.key==='ArrowLeft'?-step:event.key==='ArrowRight'?step:0;
        this.view.y += event.key==='ArrowUp'?-step:event.key==='ArrowDown'?step:0;
        this.applyView();
      } else return;
      event.preventDefault();
    });
  }

  render() {
    const root = this.element;
    root.replaceChildren(svg('rect',{width:this.map.width,height:this.map.height,class:'map-ground'}));
    const defs = svg('defs');
    const arrow = svg('marker',{id:'campus-arrow',viewBox:'0 0 10 10',refX:8,refY:5,markerWidth:4,markerHeight:4,orient:'auto'});
    arrow.append(svg('path',{d:'M0 0L10 5L0 10Z',fill:'#bd7611'}));defs.append(arrow);root.append(defs);
    for (const area of this.map.areas) root.append(svg('path',{d:area.path,class:`map-area area-${area.kind}`}));
    // Pitch details are visual only; no routes can cross these playing surfaces.
    for (const box of [{x:735,y:1090,w:96,h:120},{x:250,y:1160,w:72,h:75}]) {
      root.append(svg('rect',{x:box.x,y:box.y,width:box.w,height:box.h,class:'pitch-mark'}));
      root.append(svg('line',{x1:box.x,y1:box.y+box.h/2,x2:box.x+box.w,y2:box.y+box.h/2,class:'pitch-mark'}));
      root.append(svg('circle',{cx:box.x+box.w/2,cy:box.y+box.h/2,r:box.w/7,class:'pitch-mark'}));
    }
    for (const cls of ['road-outline','road-surface']) {
      for (const road of this.map.roads) {
        const a=this.map.nodes[road.from],b=this.map.nodes[road.to];
        root.append(svg('line',{x1:a.x,y1:a.y,x2:b.x,y2:b.y,class:cls}));
      }
    }
    for (const label of this.map.labels) root.append(svg('text',{
      x:label.x,y:label.y,class:`area-label label-${label.kind}`,'text-anchor':'middle',
      transform:`rotate(${label.rotate||0} ${label.x} ${label.y})`,
    },label.text));
    if (this.result) {
      const legs=this.result.legs || [];
      const ordered = legs.map((leg,index)=>({leg,index}));
      ordered.sort((a,b)=>Number(a.index===this.activeLeg)-Number(b.index===this.activeLeg));
      ordered.forEach(({leg,index})=>{
        const points=leg.roadPath.map(id=>`${this.map.nodes[id].x},${this.map.nodes[id].y}`).join(' ');
        root.append(svg('polyline',{points,class:`delivery-route ${this.activeLeg===index?'route-focused':''}`,
          'marker-mid':'url(#campus-arrow)','marker-end':'url(#campus-arrow)'}));
      });
    }
    const counts=new Map(),selected=new Set(this.result?.stops || []);
    for (const order of this.batch?.orders || []) counts.set(order.destinationId,(counts.get(order.destinationId)||0)+1);
    for (const place of this.map.places) {
      if (this.onlyOrders && !counts.has(place.id) && place.id!==0 && this.activePlace!==place.id) continue;
      const active=this.activePlace===place.id;
      const chosen=selected.has(place.id);
      const group=svg('g',{class:`map-poi ${chosen?'is-selected':''} ${active?'is-focused':''} poi-${place.category}`,
        role:'button',tabindex:0,'data-place':place.id,'aria-label':`${place.name}，${counts.get(place.id)||0} 单，定位查看`});
      group.append(svg('title',{},`${place.name} · 距取餐点约 ${Math.round(this.map.depotMeters[place.id])} 米`));
      if (active) group.append(svg('circle',{cx:place.x,cy:place.y,r:27,class:'poi-halo'}));
      group.append(svg('circle',{cx:place.x,cy:place.y,r:16,class:'poi-dot'}));
      const orderIndex=this.result?.stops.indexOf(place.id) ?? -1;
      const glyph=place.id===0?'取':chosen?String(orderIndex):place.category==='gate'?'门':'·';
      group.append(svg('text',{x:place.x,y:place.y+6,class:'poi-glyph','text-anchor':'middle'},glyph));
      group.append(svg('text',{x:place.labelX,y:place.labelY,class:'poi-label','text-anchor':'middle'},place.name));
      if (counts.has(place.id)) {
        group.append(svg('rect',{x:place.x+25,y:place.y-12,width:42,height:25,rx:9,class:'poi-badge'}));
        group.append(svg('text',{x:place.x+46,y:place.y+6,class:'poi-count','text-anchor':'middle'},`${counts.get(place.id)}单`));
      }
      group.addEventListener('click',()=>this.focusPlace(place.id,false));
      group.addEventListener('keydown',event=>{if(event.key==='Enter'||event.key===' '){event.preventDefault();this.focusPlace(place.id,false);}});
      root.append(group);
    }
    this.applyView();
  }
}
