import {z} from 'zod';
import type {ShopState,PublicReview} from './types.ts';
const input=z.object({number:z.string().trim().min(1).max(100),phone:z.string().regex(/^0\d{9}$/),productId:z.string().min(1).max(200),author:z.string().trim().min(2).max(60),rating:z.number().int().min(1).max(5),body:z.string().trim().min(10).max(2000)});
export function submitReview(s:ShopState,raw:unknown){
 const b=input.parse(raw),o=s.orders.find(o=>o.number===b.number.toUpperCase()&&o.customer.phone===b.phone);
 if(!o)throw Error('Mã đơn và điện thoại không khớp.');
 if(o.status!=='completed'||o.paymentStatus!=='paid')throw Error('Chỉ đánh giá đơn đã hoàn tất và thanh toán.');
 if(!o.items.some(i=>i.id===b.productId)||!s.products.some(p=>p.id===b.productId&&p.active))throw Error('Sản phẩm không thuộc đơn hàng hoặc đã ngừng hiển thị.');
 if(s.reviews?.some(r=>r.orderId===o.id&&r.productId===b.productId))throw Error('Bạn đã đánh giá sản phẩm này trong đơn hàng.');
 const review={id:crypto.randomUUID(),orderId:o.id,productId:b.productId,author:b.author,rating:b.rating,body:b.body,createdAt:new Date().toISOString(),status:'pending' as const};
 (s.reviews??=[]).unshift(review);return {id:review.id,status:review.status};
}
export function moderateReview(s:ShopState,raw:unknown,actor:string){
 const b=z.object({id:z.string(),status:z.enum(['approved','rejected','hidden']),reason:z.string().trim().max(500).default('')}).parse(raw);
 if(b.status!=='approved'&&!b.reason)throw Error('Vui lòng ghi lý do từ chối hoặc ẩn đánh giá.');
 const r=s.reviews?.find(r=>r.id===b.id);if(!r)throw Error('Không tìm thấy đánh giá.');
 r.status=b.status;r.reason=b.reason;r.moderatedAt=new Date().toISOString();r.moderatedBy=actor;
}
export function publicReviews(s:ShopState):PublicReview[]{
 return (s.reviews||[]).filter(r=>r.status==='approved'&&s.products.some(p=>p.active&&p.id===r.productId)).map(({id,productId,author,rating,body,createdAt})=>({id,productId,author,rating,body,createdAt}));
}
export {reviewSummary} from './review-summary.ts';
