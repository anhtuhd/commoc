import assert from 'node:assert/strict';
const base='http://127.0.0.1:5173';
let cookie='';
async function request(path,body,auth=true){const r=await fetch(base+'/api/store/'+path,{method:body===undefined?'GET':'POST',headers:{Origin:base,'Content-Type':'application/json',...(auth&&cookie?{Cookie:cookie}:{})},body:body===undefined?undefined:JSON.stringify(body)});return {status:r.status,data:await r.json()}}
async function ok(path,body){const r=await request(path,body);assert.ok(r.status<300,JSON.stringify(r));return r.data}
assert.equal((await request('admin',undefined,false)).status,401);
assert.equal((await request('brand',{},false)).status,401);
const signin=await fetch(base+'/signin-with-chatgpt?return_to=/quan-tri',{redirect:'manual'});cookie=signin.headers.get('set-cookie').split(';')[0];
if((await ok('session')).setup)await ok('setup',{});
const backup=await ok('backup');
const stamp=Date.now();
try {
 const draft={...backup.state.brand,name:'Thương hiệu kiểm thử'};delete draft.draft;
 await ok('brand',draft);assert.equal((await ok('public')).brand.name,backup.state.brand.name);
 await ok('brand-publish',{});assert.equal((await ok('public')).brand.name,draft.name);
 await ok('brand-restore',{});assert.equal((await ok('public')).brand.name,backup.state.brand.name);
 console.log('PASS brand draft / publish / rollback');
 const product={...backup.state.products[0],id:'qa-'+stamp,sku:'QA-'+stamp,stock:1};await ok('product',product);
 const customer={name:'Khách QA',phone:'0901234567',address:'12 Phố Huế, Hà Nội',region:'hanoi',note:'Đơn kiểm thử local'};
 const make=(key)=>({items:[{id:product.id,quantity:1,price:1}],customer,payment:'cod',key});
 const keys=['qa-order-a-'+stamp,'qa-order-b-'+stamp];
 const race=await Promise.all(keys.map(k=>request('orders',make(k),false)));
 assert.equal(race.filter(r=>r.status===201).length,1);assert.equal(race.filter(r=>r.status===400).length,1);
 const winner=race.findIndex(r=>r.status===201);const order=race[winner].data;
 assert.equal(order.total,product.price+backup.state.shipping.fee);assert.equal(order.customer,undefined);assert.equal(order.key,undefined);
 assert.equal((await ok('orders',make(keys[winner]))).id,order.id);
 assert.equal((await ok('admin')).products.find(p=>p.id===product.id).stock,0);
 await ok('order',{id:order.id,status:'cancelled',paymentStatus:'unpaid',note:'QA kết thúc'});
 await ok('order',{id:order.id,status:'cancelled',paymentStatus:'unpaid',note:'QA lặp'});
 assert.equal((await ok('admin')).products.find(p=>p.id===product.id).stock,1);
 assert.equal((await request('order',{id:order.id,status:'confirmed',paymentStatus:'unpaid'})).status,400);
 console.log('PASS concurrent last-stock order / idempotency / price authority / cancellation');
 assert.equal((await ok('track-order',{number:order.number,phone:customer.phone})).number,order.number);
 assert.equal((await request('track-order',{number:order.number,phone:'0900000000'})).status,400);
 for(let i=0;i<14;i++)await request('track-order',{number:'NOT-FOUND',phone:customer.phone});
 assert.match((await request('track-order',{number:order.number,phone:customer.phone})).data.error,/nhiều yêu cầu/);
 console.log('PASS order privacy / failed-attempt rate limit');
 const csrf=await fetch(base+'/api/store/brand',{method:'POST',headers:{Origin:'https://other.invalid',Cookie:cookie,'Content-Type':'application/json'},body:'{}'});assert.equal(csrf.status,403);
 const form=new FormData();form.set('file',new Blob(['<script>bad</script>'],{type:'image/png'}),'fake.png');const upload=await fetch(base+'/api/store/upload',{method:'POST',headers:{Origin:base,Cookie:cookie},body:form});assert.equal(upload.status,400);
 console.log('PASS anonymous admin denial / CSRF / invalid upload');
 const coupon='QA'+stamp;await ok('coupon',{code:coupon,percent:10,minimum:0,maxDiscount:50000,limit:1,endsAt:'2027-12-31',active:true});await ok('product',{...product,stock:3});
 const discounted=await Promise.all([1,2].map(n=>request('orders',{...make('qa-coupon-'+stamp+'-'+n),coupon})));
 assert.equal(discounted.filter(r=>r.status===201).length,1);
 const won=discounted.find(r=>r.status===201).data;await ok('order',{id:won.id,status:'cancelled',paymentStatus:'unpaid',note:'QA kết thúc'});
 console.log('PASS concurrent last coupon use');
 const before=(await ok('admin')).orders.length;await ok('restore',backup);assert.equal((await ok('admin')).orders.length,before);assert.equal((await ok('public')).brand.name,backup.state.brand.name);
 console.log('PASS backup configuration restore preserves orders');
} finally {await ok('restore',backup)}
console.log('All API integration checks passed (local only).');
