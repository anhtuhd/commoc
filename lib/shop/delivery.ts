import type {ShopState} from './types.ts';
export function deliveryOptions(s:ShopState,now=Date.now()){
 const c=s.shipping;if(!c.scheduleEnabled)return [];
 const result:{date:string;slot:string;remaining:number}[]=[];
 const today=new Date(now+7*3600000).toISOString().slice(0,10);
 for(let day=0;day<(c.maxDays??7);day++){
  const date=new Date(Date.parse(today+'T00:00:00Z')+day*86400000).toISOString().slice(0,10);
  for(const slot of c.slots??[]){
   if(Date.parse(date+'T'+slot.start+':00+07:00')<now+(c.leadHours??4)*3600000)continue;
   const used=s.orders.filter(o=>o.status!=='cancelled'&&o.delivery?.date===date&&o.delivery.slot===slot.label).length;
   if(used<slot.capacity)result.push({date,slot:slot.label,remaining:slot.capacity-used});
  }
 }
 return result;
}
export function validateDelivery(s:ShopState,input:unknown,now=Date.now()){
 if(!s.shipping.scheduleEnabled)return undefined;
 const selected=deliveryOptions(s,now).find(x=>typeof input==='object'&&input!==null&&'date' in input&&'slot' in input&&x.date===input.date&&x.slot===input.slot);
 if(!selected)throw Error('Khung giao hàng đã đầy hoặc hết giờ đặt. Vui lòng chọn lại.');
 return {date:selected.date,slot:selected.slot};
}
