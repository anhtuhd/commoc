import type {Order} from './types.ts';
export function bankQr(order:Pick<Order,'payment'|'paymentStatus'|'status'|'paymentExpiresAt'|'brand'|'total'|'number'|'paymentCode'>,now=Date.now()):string|null{
 if(order.payment!=='bank'||order.paymentStatus!=='unpaid'||!['new','confirmed'].includes(order.status)||!order.paymentExpiresAt||Date.parse(order.paymentExpiresAt)<=now)return null;
 const b=order.brand;if(!/^\d{6}$/.test(b?.bankBin||'')||!/^\d{4,30}$/.test(b?.bankAccount||'')||!Number.isSafeInteger(order.total)||order.total<=0)return null;
 return `https://img.vietqr.io/image/${b.bankBin}-${b.bankAccount}-compact2.png?`+new URLSearchParams({amount:String(order.total),addInfo:order.paymentCode||order.number,accountName:b.bankHolder});
}
