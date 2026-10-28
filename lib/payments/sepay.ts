import {createHmac,timingSafeEqual} from 'node:crypto';
import {z} from 'zod';
import type {Order,ShopState} from '../shop/types.ts';

export type PaymentSettings={enabled:boolean;accountNumber:string;gateway:string;bankBin:string};
export const settingsSchema=z.object({enabled:z.boolean(),accountNumber:z.string().regex(/^\d{4,30}$/),gateway:z.string().trim().min(2).max(80).regex(/^[A-Za-z0-9 ]+$/),bankBin:z.string().regex(/^\d{6}$/)});
const transactionSchema=z.object({
 id:z.number().int().positive().max(Number.MAX_SAFE_INTEGER),gateway:z.string().min(1).max(80),
 transactionDate:z.string().regex(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/),
 accountNumber:z.string().min(1).max(50),subAccount:z.string().max(100).nullish().transform(v=>v||''),
 code:z.string().max(100).nullish().transform(v=>v||''),content:z.string().max(4000),
 transferType:z.enum(['in','out']),transferAmount:z.number().int().positive().max(Number.MAX_SAFE_INTEGER),
 referenceCode:z.string().max(200).nullish().transform(v=>v||''),
});
export type Transaction=z.infer<typeof transactionSchema>;
export function parseTransaction(value:unknown):Transaction {
 const t=transactionSchema.parse(value),iso=t.transactionDate.replace(' ','T');
 const millis=Date.parse(iso+'+07:00');
 if(!Number.isFinite(millis)||new Date(millis+7*3600000).toISOString().slice(0,19)!==iso)throw Error('Invalid transaction date');
 return t;
}
export function verifySignature(raw:Uint8Array,timestamp:string|null,signature:string|null,secret:string,now=Date.now()){
 if(!secret||!timestamp||!/^\d{10}$/.test(timestamp)||!signature||!/^sha256=[a-fA-F0-9]{64}$/.test(signature)||Math.abs(now/1000-Number(timestamp))>300)return false;
 const expected=createHmac('sha256',secret).update(timestamp+'.').update(raw).digest();
 return timingSafeEqual(expected,Buffer.from(signature.slice(7),'hex'));
}
export const gatewayKey=(s:string)=>s.replace(/ /g,'').toUpperCase();
// Only whole alphanumeric tokens count. Do not silently strip punctuation from arbitrary bank memos.
function codes(content:string){return [...new Set(content.toUpperCase().match(/(?<![A-Z0-9])MV(?:[A-Z0-9]{12}|-?[A-F0-9]{8})(?![A-Z0-9])/g)||[])];}
export function matchingOrder(s:ShopState,t:Transaction):Order|undefined {
 const found=codes(t.content),code=t.code.toUpperCase();
 if(found.length!==1||(code&&code!==found[0]))return;
 const orders=s.orders.filter(o=>o.paymentCode?o.paymentCode===found[0]:o.number===found[0]||o.number.replace('-','')===found[0]);
 return orders.length===1?orders[0]:undefined;
}
export function eligibility(s:ShopState,t:Transaction,o:Order,now:number):string|null {
 const c=s.payments;
 if(!c||t.accountNumber!==c.accountNumber||gatewayKey(t.gateway)!==gatewayKey(c.gateway)||t.subAccount)return 'Tài khoản/ngân hàng nhận hoặc tài khoản ảo chưa được cấu hình';
 if(o.brand.bankAccount!==t.accountNumber||o.brand.bankBin!==c.bankBin)return 'Tài khoản nhận không khớp thông tin ngân hàng của đơn';
 if(o.payment!=='bank'||o.paymentStatus!=='unpaid')return 'Đơn không chờ thanh toán chuyển khoản';
 if(!['new','confirmed'].includes(o.status)||!o.paymentExpiresAt||Date.parse(o.paymentExpiresAt)<=now)return 'Đơn đã hủy, hết hạn hoặc không còn chờ thanh toán';
 const at=Date.parse(t.transactionDate.replace(' ','T')+'+07:00');
 if(at<Math.floor(Date.parse(o.createdAt)/1000)*1000||at>now+300000||at>Date.parse(o.paymentExpiresAt))return 'Thời gian giao dịch không phù hợp với đơn';
 if(t.transferAmount!==o.total)return 'Số tiền nhận không bằng tổng đơn';
 return null;
}
export function matchOrder(s:ShopState,t:Transaction,now=Date.now()):{status:'matched'|'needs_review'|'ignored';reason:string;order?:Order} {
 if(t.transferType!=='in')return {status:'ignored',reason:'Giao dịch tiền ra'};
 const order=matchingOrder(s,t);
 if(!order)return {status:'needs_review',reason:'Không có mã đơn duy nhất khớp nội dung giao dịch'};
 const reason=eligibility(s,t,order,now);
 return {status:reason?'needs_review':'matched',reason:reason||'Khớp mã, tài khoản, thời hạn và tổng tiền',order};
}
