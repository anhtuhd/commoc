// Run only against a disposable local VPS-runtime server (see docs/sepay.md).
import assert from 'node:assert/strict';
import {createHmac} from 'node:crypto';
const base=process.env.SEPAY_QA_URL||'http://localhost:3100';
if(!['localhost','127.0.0.1'].includes(new URL(base).hostname)||process.env.SEPAY_QA!=='1')throw Error('Disposable localhost QA server required');
const secret=process.env.SEPAY_QA_SECRET||'local-qa-sepay-secret-only-not-production';
const headers={'Content-Type':'application/json',Origin:base};
async function post(path,body,extra={}){return fetch(base+path,{method:'POST',headers:{...headers,...extra},body:JSON.stringify(body)})}
const pub=await (await fetch(base+'/api/store/public')).json();
assert.equal(pub.payments,undefined);assert.equal(pub.transactions,undefined);
assert.equal((await fetch(base+'/api/admin/payments')).status,403);
const login=await fetch(base+'/api/auth/login',{method:'POST',headers:{Origin:base,'Content-Type':'application/x-www-form-urlencoded'},body:new URLSearchParams({username:'qa-owner',password:'qa-local-only'}),redirect:'manual'});
assert.equal(login.status,303);const cookie=login.headers.get('set-cookie')?.split(';')[0];assert.ok(cookie);
const customer={name:'Khách QA SePay',phone:'0901234567',address:'12 Phố Huế Hà Nội',region:'hanoi',note:'TEST ONLY'};
const product=pub.products.find(p=>p.mode==='fixed'&&p.stock>5);
const draft={items:[{id:product.id,quantity:1}],customer,payment:'bank',key:crypto.randomUUID()};
const created=await post('/api/store/orders',draft);assert.equal(created.status,201);const order=await created.json();assert.match(order.paymentCode,/^MV[A-F0-9]{12}$/);
assert.equal((await (await post('/api/store/orders',draft)).json()).id,order.id);
const quote=await (await post('/api/store/quote',{items:draft.items,region:'hanoi'})).json();assert.equal(order.total,quote.total);
const payload={id:Date.now(),gateway:'Vietcombank',transactionDate:new Date(Date.now()+7*3600000).toISOString().slice(0,19).replace('T',' '),accountNumber:'123456789',code:order.paymentCode,content:order.paymentCode,transferType:'in',transferAmount:order.total,referenceCode:'QA'};
function signed(p){const raw=JSON.stringify(p),timestamp=String(Math.floor(Date.now()/1000));return {'Content-Type':'application/json','X-SePay-Timestamp':timestamp,'X-SePay-Signature':'sha256='+createHmac('sha256',secret).update(timestamp+'.'+raw).digest('hex')}}
async function webhook(p,override){return fetch(base+'/api/webhooks/sepay',{method:'POST',headers:override||signed(p),body:JSON.stringify(p)})}
assert.equal((await webhook(payload,{'Content-Type':'application/json'})).status,401);
const delivered=await webhook(payload);assert.equal(delivered.status,200);assert.deepEqual(await delivered.json(),{success:true});
assert.equal((await webhook(payload)).status,200);
let tracked=await (await post('/api/store/track-order',{number:order.number,phone:customer.phone})).json();assert.equal(tracked.paymentStatus,'paid');assert.equal(tracked.status,'new');assert.equal(tracked.paymentCode,order.paymentCode);
const dashboard=await (await fetch(base+'/api/admin/payments',{headers:{Cookie:cookie}})).json();
assert.equal(dashboard.transactions.find(t=>t.id==='test:'+payload.id).status,'matched');assert.equal(dashboard.outbox.filter(j=>j.id==='test:'+payload.id+':payment.received').length,1);assert.equal(JSON.stringify(dashboard).includes(secret),false);
assert.equal((await post('/api/admin/payments',{action:'settings',config:dashboard.config},{Cookie:cookie,Origin:'https://evil.example'})).status,403);
const second=await (await post('/api/store/orders',{...draft,key:crypto.randomUUID()})).json();
const unknown={...payload,transactionDate:new Date(Date.now()+7*3600000).toISOString().slice(0,19).replace('T',' '),id:payload.id+1,code:'',content:'Sai noi dung',transferAmount:second.total};
assert.equal((await webhook(unknown)).status,200);
const resolve=await post('/api/admin/payments',{action:'resolve',id:'test:'+unknown.id,number:second.number,reason:'QA xác nhận nội dung với khách'},{Cookie:cookie});assert.equal(resolve.status,200);
assert.equal((await post('/api/admin/payments',{action:'resolve',id:'test:'+unknown.id,number:second.number,reason:'QA thử gắn lại giao dịch'},{Cookie:cookie})).status,400);
tracked=await (await post('/api/store/track-order',{number:second.number,phone:customer.phone})).json();assert.equal(tracked.paymentStatus,'paid');
const cod=await (await post('/api/store/orders',{...draft,payment:'cod',key:crypto.randomUUID()})).json();assert.equal(cod.paymentStatus,'unpaid');assert.equal(cod.paymentCode,undefined);
console.log(JSON.stringify({passed:true,checks:['public-data privacy','owner authentication','order/quote/idempotency','HMAC without Origin','webhook replay','paid order lookup','durable single outbox','admin CSRF','manual reconciliation','COD unchanged'],orders:[order.number,second.number,cod.number]}));
