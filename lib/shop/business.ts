import {z} from 'zod';
import type {Brand,BusinessDetails,PublicData} from './types.ts';
export const refundLabels={FullRefund:'Hoàn tiền toàn bộ',ExchangeRefund:'Đổi sản phẩm cùng loại',StoreCreditRefund:'Hoàn bằng tín dụng mua hàng tại cửa hàng'};
export const emptyBusiness:BusinessDetails={addressConfirmed:false,street:'',locality:'',region:'',postalCode:'',country:'',shippingConfirmed:false,handlingMin:'',handlingMax:'',transitMin:'',transitMax:'',returnsConfirmed:false,returnCategory:'',returnDays:'',returnMethod:'',returnFees:'',refundType:''};
const text=z.string().trim().max(200);
const days=z.string().trim().refine(v=>v===''||/^\d{1,3}$/.test(v),'Số ngày cần là số nguyên từ 0 đến 999.');
export const businessSchema=z.object({addressConfirmed:z.boolean(),street:text,locality:text,region:text,postalCode:text,country:z.enum(['','VN']),shippingConfirmed:z.boolean(),handlingMin:days,handlingMax:days,transitMin:days,transitMax:days,returnsConfirmed:z.boolean(),returnCategory:z.enum(['','none','finite','unlimited']),returnDays:days,returnMethod:z.enum(['','mail','store']),refundType:z.enum(['','FullRefund','ExchangeRefund','StoreCreditRefund']).default(''),returnFees:z.enum(['','free','customer'])}).superRefine((b,c)=>{
 const fail=(message:string)=>c.addIssue({code:'custom',message});
 if(b.addressConfirmed&&![b.street,b.locality,b.region,b.postalCode,b.country].every(Boolean))fail('Địa chỉ đã xác nhận cần đủ số nhà/đường, phường/xã, tỉnh/thành, mã bưu chính và quốc gia.');
 if(b.shippingConfirmed&&(![b.handlingMin,b.handlingMax,b.transitMin,b.transitMax].every(v=>v!=='')||Number(b.handlingMin)>Number(b.handlingMax)||Number(b.transitMin)>Number(b.transitMax)))fail('Điền đủ thời gian giao hàng; số ngày tối đa phải lớn hơn hoặc bằng tối thiểu.');
 if(b.returnsConfirmed&&(!b.returnCategory||(b.returnCategory!=='none'&&(!b.returnMethod||!b.returnFees))||(b.returnCategory==='finite'&&(!b.returnDays||Number(b.returnDays)<1))))fail('Điền đầy đủ điều kiện đổi trả đã xác nhận.');
});
export function business(b:Brand){const result=businessSchema.safeParse({...emptyBusiness,...b.business});return result.success?result.data:emptyBusiness}
export function businessAddress(b:Brand){const v=business(b);return v.addressConfirmed?[v.street,v.locality,v.region,v.postalCode,'Việt Nam'].join(', '):b.address}
export function policyLines(data:PublicData,id:string){const b=business(data.brand),s=data.shipping;
 if(id==='giao-hang'&&b.shippingConfirmed&&s.enabled)return [`Giao hàng tại ${s.regionLabel}, Việt Nam. Phí ${s.fee.toLocaleString('vi-VN')} ₫; miễn phí khi tổng tiền hàng trước giảm giá từ ${s.freeFrom.toLocaleString('vi-VN')} ₫.`,`Chuẩn bị đơn: ${b.handlingMin}–${b.handlingMax} ngày; vận chuyển: ${b.transitMin}–${b.transitMax} ngày.`];
 if(id==='doi-tra'&&b.returnsConfirmed)return b.returnCategory==='none'?['Không chấp nhận trả hàng.']:['Áp dụng tại Việt Nam. '+(b.returnCategory==='finite'?`Thời hạn trả hàng: ${b.returnDays} ngày kể từ khi nhận hàng.`:'Không giới hạn thời hạn trả hàng.'),b.returnMethod==='mail'?'Trả hàng qua đường bưu điện.':'Trả hàng tại cửa hàng.',b.returnFees==='free'?'Trả hàng miễn phí.':'Khách hàng thanh toán chi phí vận chuyển trả hàng.',...(b.refundType?[refundLabels[b.refundType]+'.']:[])];
 return [];
}
