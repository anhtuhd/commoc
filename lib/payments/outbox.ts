import type {DatabaseSync} from 'node:sqlite';
import {atomic} from './sqlite.ts';
export type PaymentJob={id:string;message:string;lease:string;attempts:number};
export function claimPaymentNotification(db:DatabaseSync,now=Date.now()):PaymentJob|null {
 return atomic(db,()=>{
  const job=db.prepare("SELECT id,message,attempts FROM payment_outbox WHERE (status IN ('pending','failed') OR (status='sending' AND lease_until<=?)) AND attempts<6 AND retry_at<=? ORDER BY created_at LIMIT 1").get(now,now) as Omit<PaymentJob,'lease'>|undefined;
  if(!job)return null;
  const lease=crypto.randomUUID();db.prepare("UPDATE payment_outbox SET status='sending',lease=?,lease_until=?,attempts=attempts+1 WHERE id=?").run(lease,now+120000,job.id);
  return {...job,lease,attempts:job.attempts+1};
 });
}
export function finishPaymentNotification(db:DatabaseSync,job:PaymentJob,sent:boolean,now=Date.now()){
 db.prepare('UPDATE payment_outbox SET status=?,sent_at=?,last_error=?,retry_at=?,lease=NULL,lease_until=0 WHERE id=? AND lease=?').run(sent?'sent':'failed',sent?new Date(now).toISOString():null,sent?null:'Gửi Zalo thất bại hoặc phiên không còn hợp lệ',now+Math.min(3600000,60000*2**Math.min(job.attempts-1,6)),job.id,job.lease);
}
export function retryPaymentNotification(db:DatabaseSync,id:string,now=Date.now()){
 const result=db.prepare("UPDATE payment_outbox SET status='pending',attempts=0,retry_at=0,lease=NULL,lease_until=0 WHERE id=? AND (status='failed' OR (status='sending' AND lease_until<=?))").run(id,now);
 if(result.changes!==1)throw Error('Thông báo chưa lỗi hoặc vẫn đang được gửi.');
}

// The transport is injected so retries can be tested without sending real Zalo messages.
export async function deliverPaymentNotifications(db:DatabaseSync,send:(message:string)=>Promise<unknown>,canSend:()=>boolean){
 let delivered=0,failed=0;
 for(let n=0;n<5&&canSend();n++){
  const job=claimPaymentNotification(db);if(!job)break;let sent=false;
  try{await send(job.message);sent=true}catch{/* Durable queue retains the event for retry. */}
  finishPaymentNotification(db,job,sent);if(sent)delivered++;else failed++;
 }
 return {delivered,failed};
}
