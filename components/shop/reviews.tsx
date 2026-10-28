'use client';
import {ValidatedForm,FormInput,FormTextarea,FormSelect,showServerErrors} from './form';
import {useState} from 'react';
import Link from 'next/link';
import type {Review} from '../../lib/shop/types';
import {reviewSummary} from '../../lib/shop/review-summary';
import {api,useShop} from './context';
export function ProductReviews({productId}:{productId:string}){
 const {data}=useShop(),summary=reviewSummary(data.reviews,productId);
 return <section className="panel product-reviews" aria-labelledby="reviews-heading"><h2 id="reviews-heading">Đánh giá từ khách đã mua</h2>{summary.count?<><p><strong>{summary.rating}/5</strong> · {summary.count} đánh giá đã duyệt</p>{summary.items.map(r=><article key={r.id} id={'review-'+r.id} className="review-item"><strong>{r.author}</strong><p>{r.rating}/5 · Đã mua hàng · <time dateTime={r.createdAt}>{new Date(r.createdAt).toLocaleDateString('vi-VN',{timeZone:'Asia/Ho_Chi_Minh'})}</time></p><p className="review-body">{r.body}</p></article>)}</>:<p>Chưa có đánh giá được duyệt cho sản phẩm này.</p>}<p>Đánh giá được xác minh bằng đơn hàng đã hoàn tất và thanh toán. Nội dung được duyệt trước khi hiển thị; điểm đánh giá không bị chỉnh sửa.</p><Link className="text-link" href="/tra-cuu">Đã mua hàng? Tra cứu đơn để viết đánh giá →</Link></section>;
}
function ReviewForm({item,credentials}:{item:{id:string;name:string};credentials:{number:string;phone:string}}){
 const [sent,setSent]=useState(false),[busy,setBusy]=useState(false),[error,setError]=useState('');
 if(sent)return <p role="status">Đã nhận đánh giá cho {item.name}. Nội dung đang chờ duyệt.</p>;
 return <ValidatedForm className="panel stack" onSubmit={async e=>{e.preventDefault();const form=e.currentTarget;const f=new FormData(form);setBusy(true);setError('');try{await api('reviews',{...credentials,productId:item.id,author:f.get('author'),rating:Number(f.get('rating')),body:f.get('body')});setSent(true)}catch(e){setError(e instanceof Error?e.message:'Không thể gửi đánh giá.');showServerErrors(form,e)}finally{setBusy(false)}}}><h3>{item.name}</h3><label>Tên hiển thị công khai<FormInput name="author" required minLength={2} maxLength={60} autoComplete="nickname"/></label><label>Điểm đánh giá<FormSelect name="rating" required defaultValue=""><option value="" disabled>Chọn điểm</option>{[5,4,3,2,1].map(n=><option key={n} value={n}>{n}/5</option>)}</FormSelect></label><label>Cảm nhận của bạn<FormTextarea name="body" required minLength={10} maxLength={2000} rows={4}/></label><small>Tên hiển thị và nội dung sẽ công khai sau khi duyệt. Không ghi số điện thoại, địa chỉ hoặc thông tin riêng tư.</small>{error&&<p className="error" role="alert">{error}</p>}<button className="button" disabled={busy}>{busy?'Đang gửi…':'Gửi đánh giá'}</button></ValidatedForm>;
}
export function OrderReviews({order,credentials}:{order:any;credentials:{number:string;phone:string}}){
 if(order.status!=='completed'||order.paymentStatus!=='paid')return <p>Đánh giá sản phẩm sẽ mở khi đơn đã hoàn tất và thanh toán.</p>;
 return <section><h2>Chia sẻ trải nghiệm của bạn</h2><p>Mỗi sản phẩm trong đơn được đánh giá một lần. Cả đánh giá tích cực và góp ý đều được tiếp nhận.</p>{order.items.map((i:any)=>order.reviewedProductIds?.includes(i.id)?<p key={i.id}>Bạn đã gửi đánh giá cho {i.name}.</p>:<ReviewForm key={order.number+i.id} item={i} credentials={credentials}/>)}</section>;
}
const labels={pending:'Chờ duyệt',approved:'Đang hiển thị',rejected:'Đã từ chối',hidden:'Đã ẩn'};
function ReviewModeration({review:r,save,productName}:{review:Review;save:(path:string,body:any)=>Promise<void>;productName:string}){
 const [reason,setReason]=useState(r.reason||''),[busy,setBusy]=useState(false),[error,setError]=useState('');
 async function change(status:string){setBusy(true);setError('');try{await save('review-moderate',{id:r.id,status,reason})}catch(e){setError(e instanceof Error?e.message:'Không thể duyệt.')}finally{setBusy(false)}}
 return <article className="panel stack"><h3>{productName} · {r.rating}/5</h3><p>{r.author} · {labels[r.status]} · {new Date(r.createdAt).toLocaleString('vi-VN')}</p><small>Mã nội bộ đơn: {r.orderId}</small><p className="review-body">{r.body}</p><label>Lý do duyệt / từ chối / ẩn (nội bộ)<input value={reason} maxLength={500} onChange={e=>setReason(e.target.value)}/></label><div className="review-actions">{r.status!=='approved'&&<button className="button small" disabled={busy} onClick={()=>change('approved')}>Duyệt hiển thị</button>}{r.status==='pending'&&<button className="button outline small" disabled={busy} onClick={()=>change('rejected')}>Từ chối</button>}{r.status==='approved'&&<button className="button outline small" disabled={busy} onClick={()=>change('hidden')}>Ẩn đánh giá</button>}</div>{error&&<p className="error" role="alert">{error}</p>}</article>;
}
export function AdminReviews({reviews,products,save}:{reviews:Review[];products:{id:string;name:string}[];save:(path:string,body:any)=>Promise<void>}){
 const [filter,setFilter]=useState('pending');const rows=reviews.filter(r=>!filter||r.status===filter);
 return <><p>Chỉ duyệt nội dung từ đơn đã hoàn tất và thanh toán. Không từ chối chỉ vì điểm thấp; dùng lý do rõ ràng cho nội dung spam, riêng tư hoặc không liên quan. Điểm và nội dung gốc được giữ nguyên.</p><label>Lọc đánh giá<select value={filter} onChange={e=>setFilter(e.target.value)}><option value="">Tất cả</option>{Object.entries(labels).map(([k,v])=><option key={k} value={k}>{v} ({reviews.filter(r=>r.status===k).length})</option>)}</select></label>{rows.length?rows.map(r=><ReviewModeration key={r.id+r.status} review={r} save={save} productName={products.find(p=>p.id===r.productId)?.name||r.productId}/>):<p className="panel">Chưa có đánh giá trong mục này.</p>}</>;
}
