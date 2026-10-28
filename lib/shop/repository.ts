import {cache} from 'react';
import {publicReviews} from './reviews';
import {expireOrders} from './commerce';
import { env } from 'cloudflare:workers';
import { initialState } from './seed';
import type { ShopState,PublicData } from './types';
function newState(){const s=initialState();const e=env as any;if(e.VPS_OWNER_ID)s.admins=[{id:e.VPS_OWNER_ID,email:e.VPS_OWNER_EMAIL,role:'owner'}];return s}
export async function readState():Promise<{state:ShopState;revision:number}>{
 if(!env.DB)throw Error('Cơ sở dữ liệu chưa được kết nối.');
 let row=await env.DB.prepare('SELECT body,revision FROM shop_state WHERE id=?').bind('shop').first<{body:string;revision:number}>();
 if(!row){await env.DB.prepare('INSERT OR IGNORE INTO shop_state (id,body,revision) VALUES (?,?,0)').bind('shop',JSON.stringify(newState())).run();row=await env.DB.prepare('SELECT body,revision FROM shop_state WHERE id=?').bind('shop').first<{body:string;revision:number}>()}
 if(!row)throw Error('Không thể khởi tạo cửa hàng.');return {state:JSON.parse(row.body),revision:row.revision};
}
export async function updateState<T>(fn:(s:ShopState)=>T):Promise<T>{for(let n=0;n<8;n++){const {state,revision}=await readState();expireOrders(state);const result=fn(state);const changed=await env.DB!.prepare('UPDATE shop_state SET body=?,revision=revision+1 WHERE id=? AND revision=?').bind(JSON.stringify(state),'shop',revision).run();if(changed.meta.changes===1)return result}throw Error('Cửa hàng đang bận. Vui lòng thử lại.');}
export function publicData(s:ShopState):PublicData {const {draft,...brand}=s.brand;void draft;return {reviews:publicReviews(s),brand,products:s.products.filter(p=>p.active),collections:[...s.collections].sort((a,b)=>(a.sortOrder||0)-(b.sortOrder||0)),campaigns:s.campaigns.filter(c=>c.active&&new Date(c.startsAt+'T00:00:00+07:00').getTime()<=Date.now()&&new Date(c.endsAt+'T23:59:59+07:00').getTime()>=Date.now()),articles:s.articles.filter(a=>a.published),shipping:s.shipping}}
export function audit(s:ShopState,actor:string,action:string){s.audit.unshift({at:new Date().toISOString(),actor,action});s.audit=s.audit.slice(0,2000)}
export function rateLimit(s:ShopState,key:string,limit=15){const now=Date.now();for(const k in s.rate)if(s.rate[k].until<now)delete s.rate[k];const r=s.rate[key]??{count:0,until:now+600000};if(r.count>=limit)throw Error('Bạn đã gửi nhiều yêu cầu. Vui lòng thử lại sau ít phút.');r.count++;s.rate[key]=r}

// Request-scoped only: mutations and CAS retries always call uncached readState.
export const readPageState=cache(readState);
