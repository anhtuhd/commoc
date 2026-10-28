import {InputError,customerErrors} from './input-errors.ts';
import {validateDelivery} from './delivery.ts';
import type { ShopState,Order } from './types.ts';
export function quote(s:ShopState,b:any){
 if(!s.shipping.enabled||b.region!=='hanoi')throw Error('Khu vực này cần liên hệ để báo phí giao hàng.');
 if(!Array.isArray(b.items)||!b.items.length||b.items.length>30)throw Error('Giỏ hàng chưa hợp lệ.');
 const counts=new Map<string,number>();for(const i of b.items){if(typeof i.id!=='string'||!Number.isInteger(i.quantity)||i.quantity<1||i.quantity>99)throw Error('Số lượng chưa hợp lệ.');counts.set(i.id,(counts.get(i.id)||0)+i.quantity)}
 const items=[...counts].map(([id,quantity])=>{const p=s.products.find(p=>p.id===id&&p.active);if(!p||p.mode==='quote')throw Error('Sản phẩm cần báo giá hoặc không còn mở bán.');if(quantity>p.stock||quantity>99)throw Error(`${p.name}: không đủ số lượng trong kho.`);return {id,name:p.name,unit:p.unit,price:p.price,quantity}});
 const subtotal=items.reduce((n,p)=>n+p.price*p.quantity,0);const shipping=subtotal>=s.shipping.freeFrom?0:s.shipping.fee;let discount=0;const coupon=String(b.coupon||'').trim().toUpperCase();
 if(coupon){const c=s.coupons.find(c=>c.code===coupon);if(!c||!c.active||new Date(c.endsAt+'T23:59:59+07:00').getTime()<Date.now()||c.used>=c.limit||subtotal<c.minimum)throw Error('Mã ưu đãi không hợp lệ, đã hết hạn hoặc chưa đạt giá trị tối thiểu.');discount=Math.min(Math.floor(subtotal*c.percent/100),c.maxDiscount,subtotal)}
 return {items,subtotal,shipping,discount,total:subtotal+shipping-discount,coupon};
}
export function placeOrder(s:ShopState,b:any):Order{
 const c=b.customer,errors=customerErrors(c);if(Object.keys(errors).length)throw new InputError('Vui lòng kiểm tra thông tin nhận hàng.',errors);
 if(!['cod','bank'].includes(b.payment))throw Error('Phương thức thanh toán không hợp lệ.');
 if(typeof b.key!=='string'||b.key.length<16||b.key.length>100)throw Error('Mã yêu cầu không hợp lệ.');
 const fingerprint=JSON.stringify({items:b.items,customer:c,payment:b.payment,coupon:b.coupon||'',delivery:b.delivery});const existing=s.orders.find(o=>o.key===b.key);if(existing){if(existing.fingerprint!==fingerprint)throw Error('Yêu cầu đã thay đổi. Vui lòng tải lại trang thanh toán.');return existing}
 if(b.payment==='bank'&&(!s.brand.bankAccount||!s.brand.bankName||!s.brand.bankHolder))throw Error('Cửa hàng chưa bật chuyển khoản. Vui lòng chọn COD.');
 const delivery=validateDelivery(s,b.delivery);
 const q=quote(s,{...b,region:c.region});for(const i of q.items)s.products.find(p=>p.id===i.id)!.stock-=i.quantity;if(q.coupon)s.coupons.find(c=>c.code===q.coupon)!.used++;
 let paymentCode:string|undefined;if(b.payment==='bank'){do{paymentCode='MV'+crypto.randomUUID().replaceAll('-','').slice(0,12).toUpperCase()}while(s.orders.some(o=>o.paymentCode===paymentCode))}
 const id=crypto.randomUUID();const order:Order={paymentCode,...q,delivery,paymentExpiresAt:b.payment==='bank'?new Date(Date.now()+(s.shipping.paymentMinutes??60)*60000).toISOString():undefined,id,number:'MV-'+id.slice(0,8).toUpperCase(),key:b.key,fingerprint,createdAt:new Date().toISOString(),customer:{name:c.name.trim(),phone:c.phone,address:c.address.trim(),region:c.region,note:String(c.note||'')},payment:b.payment,status:'new',paymentStatus:'unpaid',note:'',brand:{name:s.brand.name,phone:s.brand.phone,bankName:s.brand.bankName,bankAccount:s.brand.bankAccount,bankHolder:s.brand.bankHolder,bankBin:s.brand.bankBin},utm:String(b.utm||'').slice(0,200)};s.orders.unshift(order);return order;
}
const transitions:Record<string,string[]>={new:['confirmed','cancelled'],confirmed:['preparing','cancelled'],preparing:['shipping','cancelled'],shipping:['completed'],completed:[],cancelled:[]};
export function changeOrder(s:ShopState,id:string,status:string,paymentStatus:string,note:string){const o=s.orders.find(o=>o.id===id);if(!o)throw Error('Không tìm thấy đơn.');if(status!==o.status&&!transitions[o.status]?.includes(status))throw Error('Không thể chuyển trạng thái này.');if(o.sepayTransactionId&&paymentStatus==='unpaid'&&o.paymentStatus!=='unpaid')throw Error('Thanh toán SePay đã đối soát không thể đặt lại chưa thanh toán.');if(!['unpaid','paid','refunded'].includes(paymentStatus))throw Error('Trạng thái thanh toán không hợp lệ.');if(paymentStatus==='refunded'&&o.paymentStatus!=='paid'&&o.paymentStatus!=='refunded')throw Error('Chỉ hoàn tiền đơn đã thanh toán.');if(o.payment==='bank'&&paymentStatus==='unpaid'&&['preparing','shipping','completed'].includes(status))throw Error('Đối soát thanh toán trước khi chuẩn bị đơn chuyển khoản.');if(o.status==='cancelled'&&paymentStatus!==o.paymentStatus)throw Error('Đơn đã hủy: đối soát riêng, không ghi nhận thanh toán cho đơn đã trả tồn kho.');if(status==='cancelled'&&o.status!=='cancelled'){for(const i of o.items){const p=s.products.find(p=>p.id===i.id);if(p)p.stock+=i.quantity}if(o.coupon){const c=s.coupons.find(c=>c.code===o.coupon);if(c)c.used=Math.max(0,c.used-1)}}o.status=status;o.paymentStatus=paymentStatus;o.note=note.slice(0,1000);return o}

// Only pending bank payments reserve stock until this deadline. COD is unaffected.
export function expireOrders(s:ShopState,now=Date.now()){
 let count=0;
 for(const o of s.orders){
  if(o.payment==='bank'&&o.paymentStatus==='unpaid'&&['new','confirmed'].includes(o.status)&&o.paymentExpiresAt&&Date.parse(o.paymentExpiresAt)<=now){
   changeOrder(s,o.id,'cancelled','unpaid','Tự hủy do quá hạn thanh toán. Nếu đã chuyển tiền, liên hệ cửa hàng để đối soát.');count++;
  }
 }
 return count;
}
