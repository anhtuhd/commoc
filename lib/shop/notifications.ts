import type {ShopState} from './types.ts';
export type NotificationConfig={enabled:boolean;phone:string;enabledAt?:string;command?:'connect'|'disconnect';requestId?:string};
export function notificationMessage(o:{number:string;brand:{name:string};customer:{name:string;phone:string;address:string;note:string};items:{name:string;quantity:number}[];total:number;payment:string;delivery?:{date:string;slot:string}}){return [`Đơn mới ${o.number} · ${o.brand.name}`,`${o.customer.name} · ${o.customer.phone}`,o.customer.address,...o.items.map(i=>`${i.quantity} × ${i.name}`),`Tổng: ${o.total.toLocaleString('vi-VN')}đ · ${o.payment==='bank'?'Chuyển khoản':'COD'}`,o.delivery?`Giao: ${o.delivery.date} ${o.delivery.slot}`:'',o.customer.note?`Ghi chú: ${o.customer.note}`:''].filter(Boolean).join('\n').slice(0,1900)}
export function pendingNotifications(s:ShopState,now=Date.now()){
 const config=s.notifications;if(!config?.enabled||!config.phone||!config.enabledAt)return [];
 return s.orders.filter(o=>{const log=s.notificationLog?.[o.id];return o.createdAt>=config.enabledAt!&&o.status!=='cancelled'&&!log?.sent&&(log?.attempts??0)<6&&(log?.retryAt??0)<=now}).slice(0,5);
}
export function recordNotification(s:ShopState,id:string,sent:boolean,now=Date.now()){
 s.notificationLog??={};const attempts=(s.notificationLog[id]?.attempts??0)+1;
 s.notificationLog[id]={sent,attempts,at:new Date(now).toISOString(),retryAt:now+Math.min(3600000,60000*2**Math.min(attempts-1,6))};
}
