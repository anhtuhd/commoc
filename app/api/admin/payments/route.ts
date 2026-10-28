import {getChatGPTUser} from '../../../chatgpt-auth';
import {readState} from '../../../../lib/shop/repository';
import {authorize} from '../../../../lib/shop/access';
import {paymentDB,sepayEnvironment} from '../../../../lib/payments/server';
import {paymentDashboard,resolvePayment} from '../../../../lib/payments/repository';
import {retryPaymentNotification} from '../../../../lib/payments/outbox';
import {atomic} from '../../../../lib/payments/sqlite';
import {settingsSchema} from '../../../../lib/payments/sepay';
import {boundedBody} from '../../../../lib/payments/http';
import type {ShopState} from '../../../../lib/shop/types';
export const runtime='nodejs';
export const dynamic='force-dynamic';
const json=(data:unknown,status=200)=>Response.json(data,{status,headers:{'Cache-Control':'no-store','X-Content-Type-Options':'nosniff'}});
async function owner(){if(process.env.COMMO_RUNTIME!=='vps')return null;const user=await getChatGPTUser();if(!user)return null;const {state}=await readState();try{authorize(state,user.userId,['owner'])}catch{return null}return {user,state}}
export async function GET(){const access=await owner();if(!access)return json({error:'Chỉ chủ cửa hàng có quyền xem thanh toán.'},403);const config=sepayEnvironment();return json({config:access.state.payments||{enabled:false,accountNumber:'',gateway:'',bankBin:''},secretConfigured:config.configured,mode:config.mode,autoMatch:config.autoMatch,...paymentDashboard(paymentDB())})}
export async function POST(req:Request){
 if(req.headers.get('origin')!==(process.env.COMMO_SITE_ORIGIN||process.env.NEXT_PUBLIC_SITE_URL))return json({error:'Nguồn yêu cầu không hợp lệ.'},403);
 const access=await owner();if(!access)return json({error:'Chỉ chủ cửa hàng có quyền đối soát.'},403);
 try{const body=JSON.parse(new TextDecoder().decode(await boundedBody(req,8192))),db=paymentDB();
  if(body.action==='resolve')return json(resolvePayment(db,String(body.id||''),body.number,body.reason,access.user.userId));
  if(body.action!=='settings'&&body.action!=='retry')throw Error('Thao tác không hợp lệ.');
  atomic(db,()=>{
   const row=db.prepare("SELECT body FROM shop_state WHERE id='shop'").get() as {body:string};const s=JSON.parse(row.body) as ShopState;
   authorize(s,access.user.userId,['owner']);
   if(body.action==='settings'){
    const config=settingsSchema.parse(body.config);
    if(config.enabled&&!sepayEnvironment().configured)throw Error('Cần cấu hình secret SePay trên VPS trước khi bật.');
    if(config.enabled&&(s.brand.bankAccount!==config.accountNumber||s.brand.bankBin!==config.bankBin))throw Error('Tài khoản và BIN phải khớp thương hiệu đã xuất bản.');
    s.payments=config;
   }else retryPaymentNotification(db,String(body.id||''));
   s.audit.unshift({at:new Date().toISOString(),actor:access.user.userId,action:'SePay: '+body.action});s.audit=s.audit.slice(0,2000);
   db.prepare("UPDATE shop_state SET body=?,revision=revision+1 WHERE id='shop'").run(JSON.stringify(s));
  });return json({ok:true});
 }catch(e){const message=e instanceof Error?e.message:'Không thể lưu.';return json({error:message.startsWith('[\n')?'Kiểm tra số tài khoản, tên gateway và BIN 6 số.':message},400)}
}
