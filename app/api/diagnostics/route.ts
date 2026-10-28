import {getChatGPTUser} from '../../chatgpt-auth';
import {readState} from '../../../lib/shop/repository';
import {authorize} from '../../../lib/shop/access';
import {metricSchema} from '../../../lib/diagnostics/metrics';
import {saveSample,performanceReport} from '../../../lib/diagnostics/storage';
export const dynamic='force-dynamic';
const buckets=new Map<string,{count:number;until:number}>();
export async function POST(req:Request){
 const origin=process.env.COMMO_RUNTIME==='vps'?(process.env.COMMO_SITE_ORIGIN||process.env.NEXT_PUBLIC_SITE_URL):new URL(req.url).origin;
 if(req.headers.get('origin')!==origin)return new Response(null,{status:403});
 if(Number(req.headers.get('content-length')||0)>2048)return new Response(null,{status:413});
 try{const raw=await req.text();if(raw.length>2048)return new Response(null,{status:413});const sample=metricSchema.parse(JSON.parse(raw));const now=Date.now();for(const [k,b] of buckets)if(b.until<now)buckets.delete(k);const key=req.headers.get('cf-connecting-ip')||'local';const bucket=buckets.get(key)||{count:0,until:now+60000};if(bucket.count>=120||buckets.size>5000)return new Response(null,{status:429});bucket.count++;buckets.set(key,bucket);try{await saveSample(sample)}catch{console.error(JSON.stringify({event:'telemetry_write_failed',requestId:crypto.randomUUID()}));return new Response(null,{status:503})}return new Response(null,{status:204,headers:{'Cache-Control':'no-store'}});}catch{return new Response(null,{status:400});}
}
export async function GET(){const user=await getChatGPTUser();if(!user)return Response.json({error:'Vui lòng đăng nhập.'},{status:401});try{authorize((await readState()).state,user.userId,['owner']);}catch{return Response.json({error:'Không có quyền xem báo cáo.'},{status:403});}try{return Response.json(await performanceReport(),{headers:{'Cache-Control':'no-store'}});}catch{console.error(JSON.stringify({event:'telemetry_read_failed',requestId:crypto.randomUUID()}));return Response.json({error:'Không thể đọc số đo. Kiểm tra log máy chủ.'},{status:503});}}
