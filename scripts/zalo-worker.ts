import {initializePayments} from '../lib/payments/sqlite.ts';
import {deliverPaymentNotifications} from '../lib/payments/outbox.ts';
import type {ShopState} from '../lib/shop/types.ts';
import {DatabaseSync} from 'node:sqlite';
import {mkdirSync,readFileSync,writeFileSync,renameSync,rmSync,existsSync} from 'node:fs';
import {Zalo,ThreadType,LoginQRCallbackEventType,type API,type Credentials} from 'zca-js';
import {pendingNotifications,recordNotification,notificationMessage} from '../lib/shop/notifications.ts';
const root=process.env.DATA_DIR||'/data';const credentials=root+'/zalo/session.json';
mkdirSync(root+'/zalo',{recursive:true,mode:0o700});
const db=new DatabaseSync(root+'/shop.sqlite');db.exec('PRAGMA busy_timeout=10000');initializePayments(db);
function read(){const row=db.prepare("SELECT body FROM shop_state WHERE id='shop'").get() as {body:string};return JSON.parse(row.body) as ShopState}
function update(fn:(s:ShopState)=>void){db.exec('BEGIN IMMEDIATE');try{const s=read();fn(s);db.prepare("UPDATE shop_state SET body=?,revision=revision+1 WHERE id='shop'").run(JSON.stringify(s));db.exec('COMMIT')}catch(e){db.exec('ROLLBACK');throw e}}
let api:API|null=null;let loginRetryAt=0;
function status(value:Omit<NonNullable<ShopState['zaloStatus']>,'updatedAt'>){update(s=>{s.zaloStatus={...value,updatedAt:new Date().toISOString()}})}
const zalo=()=>new Zalo({logging:false,selfListen:false});
async function connect(requestId:string){
 let saved:Credentials|null=null;
 status({state:'connecting'});
 try{const connected=await zalo().loginQR({},event=>{
  if(read().notifications?.requestId!==requestId){event.actions?.abort();return}
  if(event.type===LoginQRCallbackEventType.QRCodeGenerated)status({state:'qr',qr:event.data.image,expiresAt:Date.now()+120000});
  if(event.type===LoginQRCallbackEventType.QRCodeExpired||event.type===LoginQRCallbackEventType.QRCodeDeclined)event.actions.abort();
  if(event.type===LoginQRCallbackEventType.GotLoginInfo)saved=event.data;
 });
 if(read().notifications?.requestId!==requestId)return;
 if(!saved)throw Error('No credentials');
 writeFileSync(credentials+'.tmp',JSON.stringify(saved),{mode:0o600});renameSync(credentials+'.tmp',credentials);api=connected;status({state:'connected'});
 }catch{api=null;status({state:'error',message:'Không thể kết nối hoặc QR đã hết hạn. Hãy tạo QR mới.'})}
}
async function tick(){
 const config=read().notifications;
 if(config?.command){update(s=>{if(s.notifications?.requestId===config.requestId)delete s.notifications!.command});
  if(config.command==='disconnect'){api=null;rmSync(credentials,{force:true});status({state:'disconnected'});return}
  await connect(config.requestId!);return;
 }
 if(!config?.enabled)return;
 if(!api&&existsSync(credentials)&&Date.now()>=loginRetryAt){try{api=await zalo().login(JSON.parse(readFileSync(credentials,'utf8')));status({state:'connected'})}catch{loginRetryAt=Date.now()+300000;status({state:'error',message:'Phiên Zalo không còn hợp lệ. Kết nối lại bằng QR.'});return}}
 if(!api)return;
 const jobs=pendingNotifications(read());
 const hasPayments=!!db.prepare("SELECT id FROM payment_outbox WHERE status IN ('pending','failed','sending') AND attempts<6 AND retry_at<=? LIMIT 1").get(Date.now());
 if((!jobs.length&&!hasPayments)||Date.now()<loginRetryAt)return;
 let recipient:{uid:string};
 try{recipient=await api.findUser(config.phone)}catch{status({state:'error',message:'Không tìm thấy số Zalo nhận đơn. Kiểm tra số và quyền tìm kiếm.'});loginRetryAt=Date.now()+300000;return}
 for(const o of jobs){const latest=read();if(!latest.notifications?.enabled||latest.notifications.phone!==config.phone)break;let sent=false;
  try{await api.sendMessage(notificationMessage(o),recipient.uid,ThreadType.User);sent=true}catch{}
  update(s=>recordNotification(s,o.id,sent));status({state:sent?'connected':'error',message:sent?'Đã gửi thông báo đơn gần nhất.':'Gửi thất bại, hệ thống thử lại tối đa 6 lần.'});
 }
 const result=await deliverPaymentNotifications(db,message=>api!.sendMessage(message,recipient.uid,ThreadType.User),()=>{const latest=read().notifications;return !!latest?.enabled&&latest.phone===config.phone});
 if(result.delivered||result.failed)status({state:result.failed?'error':'connected',message:result.failed?'Thông báo thanh toán đang chờ thử lại.':'Đã gửi thông báo thanh toán.'});
}
status({state:existsSync(credentials)?'starting':'disconnected'});
for(;;){try{await tick()}catch{console.error('Zalo worker operation failed; retrying')}await new Promise(resolve=>setTimeout(resolve,15000))}
