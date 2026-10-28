import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createHmac,randomUUID} from 'node:crypto';
const base=process.env.QA_URL||'http://localhost:3401';
if(!/^http:\/\/(localhost|127\.0\.0\.1):3401$/.test(base))throw Error('QA localhost only');
const env=Object.fromEntries(readFileSync('work/go/qa.env','utf8').trim().split('\n').map(x=>{const i=x.indexOf('=');return[x.slice(0,i),x.slice(i+1)]}));
const login=await fetch(base+'/api/auth/login',{method:'POST',headers:{origin:base,'content-type':'application/x-www-form-urlencoded'},body:new URLSearchParams({username:env.ADMIN_USER,password:readFileSync('work/go/qa-password','utf8')}),redirect:'manual'});assert.equal(login.status,303);const cookie=login.headers.get('set-cookie')?.split(';')[0];assert(cookie?.includes('__Host-commo_session='));
async function get(path){const r=await fetch(base+path,{headers:{cookie}});assert.equal(r.status,200,path);return r.json()}
async function post(path,data,expected=200){const r=await fetch(base+path,{method:'POST',headers:{cookie,origin:base,'content-type':'application/json'},body:JSON.stringify(data)});const value=await r.json();assert.equal(r.status,expected,JSON.stringify(value));return value}
const session=await get('/api/store/session');assert.equal(session.role,'owner');
assert.equal((await fetch(base+'/api/store/admin')).status,401);
assert.equal((await fetch(base+'/api/store/orders',{method:'POST',headers:{'content-type':'application/json'},body:'{}'})).status,403);
const input={key:randomUUID(),items:[{id:'com-tuoi',quantity:1}],customer:{name:'QA HTTP',phone:'0900000000',address:'Địa chỉ kiểm thử Hà Nội',region:'hanoi',note:''},payment:'bank'};
const order=await post('/api/store/orders',input,201);assert.equal(order.paymentStatus,'unpaid');assert(!order.customer);assert(!order.key);const again=await post('/api/store/orders',input,201);assert.equal(again.id,order.id);
async function pay(o,id){const date=new Date(Date.now()+7*3600000).toISOString().slice(0,19).replace('T',' ');const t={id,gateway:'MB',transactionDate:date,accountNumber:'123456',content:'Thanh toan '+o.paymentCode,code:o.paymentCode,transferType:'in',transferAmount:o.total};const raw=JSON.stringify(t),timestamp=String(Math.floor(Date.now()/1000));const signature='sha256='+createHmac('sha256',env.SEPAY_WEBHOOK_SECRET).update(timestamp+'.'+raw).digest('hex');const req=()=>fetch(base+'/api/webhooks/sepay',{method:'POST',headers:{'content-type':'application/json','X-SePay-Timestamp':timestamp,'X-SePay-Signature':signature},body:raw});assert.equal((await req()).status,200);assert.equal((await req()).status,200)}
await pay(order,Date.now());
const browserOrder=await post('/api/store/track-order',{number:'MV-FB5C0689',phone:'0900000000'});if(browserOrder.paymentStatus==='unpaid')await pay(browserOrder,Date.now()+1);
const tracked=await post('/api/store/track-order',{number:order.number,phone:'0900000000'});assert.equal(tracked.paymentStatus,'paid');assert.equal(tracked.status,'new');
for(const status of ['confirmed','preparing','shipping','completed'])await post('/api/store/order',{id:order.id,status,paymentStatus:'paid',note:'QA'});
const review=await post('/api/store/reviews',{number:order.number,phone:'0900000000',productId:'com-tuoi',author:'Khách QA',rating:5,body:'Đánh giá kiểm thử từ đơn đã thanh toán.'},201);
let pub=await get('/api/store/public');assert(!pub.reviews.some(r=>r.id===review.id));await post('/api/store/review-moderate',{id:review.id,status:'approved',reason:''});pub=await get('/api/store/public');assert(pub.reviews.some(r=>r.id===review.id));
const html=await(await fetch(base+'/san-pham/com-tuoi')).text();const graphs=[...html.matchAll(/<script[^>]*type="application\/ld\+json"[^>]*>([\s\S]*?)<\/script>/g)].flatMap(m=>{const x=JSON.parse(m[1]);return x['@graph']||[x]});const product=graphs.find(x=>x['@type']==='Product');assert(product);assert.equal(graphs.find(x=>x["@id"]===product.brand["@id"]).name,pub.brand.name);assert.equal(Number(product.offers.price),pub.products.find(p=>p.id==='com-tuoi').price);assert(product.aggregateRating);assert.equal(product.aggregateRating.reviewCount,pub.reviews.filter(r=>r.productId==="com-tuoi").length);
const admin=await get('/api/store/admin'),brand={...admin.brand};delete brand.draft;await post('/api/store/brand',{...brand,name:'QA Draft'});assert.equal((await get('/api/store/public')).brand.name,brand.name);await post('/api/store/brand-publish',{});assert.equal((await get('/api/store/public')).brand.name,'QA Draft');await post('/api/store/brand-restore',{});assert.equal((await get('/api/store/public')).brand.name,brand.name);
for(const path of ['/quan-tri','/tra-cuu','/thanh-toan']){const h=await(await fetch(base+path)).text();assert(!h.includes('type="application/ld+json"'),path)}
assert.equal((await fetch(base+'/api/webhooks/sepay',{method:'POST',body:'{}'})).status,401);
const dashboard=await get('/api/admin/payments');assert(dashboard.transactions.some(t=>t.status==='matched'));
console.log('PASS: login, role, CSRF, checkout, idempotency, signed webhook replay, paid tracking, order workflow, review moderation, JSON-LD, draft/publish/restore, private pages, payment dashboard');
