import {test} from 'node:test';
import assert from 'node:assert/strict';
import {createHmac} from 'node:crypto';
import {DatabaseSync} from 'node:sqlite';
import {initialState} from '../lib/shop/seed.ts';
import {placeOrder,expireOrders,changeOrder} from '../lib/shop/commerce.ts';
import {bankQr} from '../lib/shop/payment.ts';
import {verifySignature,parseTransaction,matchOrder} from '../lib/payments/sepay.ts';
import {initializePayments,receivePayment,resolvePayment,paymentDashboard} from '../lib/payments/repository.ts';
import {claimPaymentNotification,finishPaymentNotification,deliverPaymentNotifications} from '../lib/payments/outbox.ts';
import {boundedBody} from '../lib/payments/http.ts';

const secret='test-secret-for-hmac-only',now=Date.now();
function fixture(){
 const state=initialState();Object.assign(state.brand,{bankAccount:'123456789',bankName:'Vietcombank',bankBin:'970436',bankHolder:'TEST'});
 state.payments={enabled:true,accountNumber:'123456789',gateway:'Vietcombank',bankBin:'970436'};
 const order=placeOrder(state,{items:[{id:state.products.find(p=>p.mode==='fixed'&&p.active)!.id,quantity:1}],customer:{name:'Khách thử',phone:'0901234567',address:'12 Phố Huế Hà Nội',region:'hanoi',note:''},payment:'bank',key:crypto.randomUUID()});
 order.createdAt=new Date(now-10000).toISOString();order.paymentExpiresAt=new Date(now+60000).toISOString();
 const payload={id:92704,gateway:'Vietcombank',transactionDate:new Date(now+7*3600000).toISOString().slice(0,19).replace('T',' '),accountNumber:'123456789',subAccount:'',code:order.paymentCode!,content:order.paymentCode!+' thanh toan',transferType:'in',transferAmount:order.total,referenceCode:'FT92704'};
 const db=new DatabaseSync(':memory:');db.exec('CREATE TABLE shop_state(id TEXT PRIMARY KEY,body TEXT NOT NULL,revision INTEGER NOT NULL DEFAULT 0)');db.prepare('INSERT INTO shop_state VALUES(?,?,0)').run('shop',JSON.stringify(state));initializePayments(db);
 return {state,order,payload,db};
}
const read=(db:DatabaseSync)=>JSON.parse((db.prepare('SELECT body FROM shop_state').get() as {body:string}).body);
test('HMAC covers raw bytes, rejects tampering and replay timestamps',()=>{
 const raw=Buffer.from('{ "id": 1 }'),ts=String(Math.floor(now/1000));const signature='sha256='+createHmac('sha256',secret).update(ts+'.').update(raw).digest('hex');
 assert.equal(verifySignature(raw,ts,signature,secret,now),true);
 assert.equal(verifySignature(Buffer.from('{"id":1}'),ts,signature,secret,now),false);
 assert.equal(verifySignature(raw,ts,signature,secret,now+301000),false);
 assert.equal(verifySignature(raw,ts,'sha256=x',secret,now),false);
});
test('QR uses unique stable payment code; bank snapshot remains unchanged',()=>{
 const {state,order}=fixture();assert.match(order.paymentCode!,/^MV[A-F0-9]{12}$/);
 assert.equal(new URL(bankQr(order,now)!).searchParams.get('addInfo'),order.paymentCode);
 state.brand.bankAccount='999999';assert.equal(order.brand.bankAccount,'123456789');
});
test('received payment atomically updates paid and outbox; duplicate is a no-op',()=>{
 const {db,payload}=fixture();assert.equal(receivePayment(db,parseTransaction(payload),'live',true,now).status,'matched');
 assert.equal(read(db).orders[0].paymentStatus,'paid');assert.equal(read(db).orders[0].status,'new');
 assert.equal(receivePayment(db,parseTransaction(payload),'live',true,now).duplicate,true);
 assert.equal(db.prepare('SELECT COUNT(*) n FROM payment_outbox').get()!.n,1);
 assert.equal(db.prepare('SELECT revision FROM shop_state').get()!.revision,1);db.close();
});
test('reject mismatches, late payments, multiple codes, wrong bank and COD',()=>{
 for(const scenario of ['under','over','bank','account','expired','cancelled','cod','before','multiple','suffix','subaccount']){
  const {state,order,payload,db}=fixture();
  if(scenario==='under')payload.transferAmount--;
  if(scenario==='over')payload.transferAmount++;
  if(scenario==='bank')payload.gateway='OtherBank';
  if(scenario==='account')payload.accountNumber='00000000';
  if(scenario==='expired')order.paymentExpiresAt=new Date(now-1).toISOString();
  if(scenario==='cancelled')order.status='cancelled';
  if(scenario==='cod')order.payment='cod';
  if(scenario==='before')order.createdAt=new Date(now+60000).toISOString();
  if(scenario==='multiple')payload.content+=' MV123456789ABC';
  if(scenario==='suffix')payload.content=payload.code+'XYZ';
  if(scenario==='subaccount')payload.subAccount='unconfigured-va';
  assert.notEqual(matchOrder(state,parseTransaction(payload),now).status,'matched',scenario);db.close();
 }
});
test('invalid numeric/date input rejected and outgoing does not mark paid',()=>{
 const {db,payload}=fixture();for(const bad of [{transferAmount:1.5},{id:-1},{transactionDate:'2026-02-30 12:00:00'},{accountNumber:123},{transferAmount:Number.MAX_SAFE_INTEGER+1}])assert.throws(()=>parseTransaction({...payload,...bad}));
 receivePayment(db,parseTransaction({...payload,transferType:'out'}),'live',true,now);assert.equal(read(db).orders[0].paymentStatus,'unpaid');db.close();
});
test('same ID with changed payload cannot overwrite an existing payment',()=>{
 const {db,payload}=fixture();receivePayment(db,parseTransaction(payload),'live',true,now);
 assert.equal(receivePayment(db,parseTransaction({...payload,transferAmount:1}),'live',true,now).status,'conflict');
 assert.equal(read(db).orders[0].paymentStatus,'paid');assert.equal(db.prepare('SELECT COUNT(*) n FROM payment_outbox').get()!.n,1);db.close();
});
test('transaction rollback does not leave paid state or ledger after outbox failure',()=>{
 const {db,payload}=fixture();db.exec("CREATE TRIGGER fail_outbox BEFORE INSERT ON payment_outbox BEGIN SELECT RAISE(ABORT,'test failure'); END");
 assert.throws(()=>receivePayment(db,parseTransaction(payload),'live',true,now));
 assert.equal(read(db).orders[0].paymentStatus,'unpaid');assert.equal(db.prepare('SELECT COUNT(*) n FROM payment_transactions').get()!.n,0);db.close();
});
test('manual reconciliation requires reason, correct amount and active order; cannot reuse transaction',()=>{
 const {db,payload,order}=fixture();payload.code='';payload.content='Sai noi dung';
 receivePayment(db,parseTransaction(payload),'live',true,now);
 assert.throws(()=>resolvePayment(db,'live:92704',order.number,'','owner',now));
 resolvePayment(db,'live:92704',order.number,'Đã xác nhận với khách','owner',now);
 assert.equal(read(db).orders[0].paymentStatus,'paid');
 assert.throws(()=>resolvePayment(db,'live:92704',order.number,'Thử gắn lại','owner',now));db.close();
});
test('outbox leases work across ticks, retries back off, stale completions do not win',()=>{
 const {db,payload}=fixture();receivePayment(db,parseTransaction(payload),'live',true,now);
 const job=claimPaymentNotification(db,now)!;assert.ok(job);assert.equal(claimPaymentNotification(db,now),null);
 finishPaymentNotification(db,job,false,now);assert.equal(claimPaymentNotification(db,now+1000),null);
 const retry=claimPaymentNotification(db,now+60001)!;assert.ok(retry);
 finishPaymentNotification(db,job,true,now+60002);assert.notEqual(paymentDashboard(db).outbox[0].status,'sent');
 finishPaymentNotification(db,retry,true,now+60003);assert.equal(paymentDashboard(db).outbox[0].status,'sent');db.close();
});
test('paid state survives expiry and stale CAS writer cannot undo it',()=>{
 const {db,payload}=fixture();const stale=read(db);receivePayment(db,parseTransaction(payload),'live',true,now);
 assert.equal(db.prepare("UPDATE shop_state SET body=?,revision=revision+1 WHERE revision=0").run(JSON.stringify(stale)).changes,0);
 const s=read(db),stock=s.products[0].stock;expireOrders(s,now+120000);assert.equal(s.orders[0].status,'new');assert.equal(s.products[0].stock,stock);
 assert.throws(()=>changeOrder(s,s.orders[0].id,'new','unpaid',''));db.close();
});
test('old order code can match exactly; disabled auto-match retains a review event',()=>{
 const {db,state,order,payload}=fixture();delete order.paymentCode;payload.code='';payload.content='Chuyen '+order.number+' thanh toan';
 assert.equal(matchOrder(state,parseTransaction(payload),now).status,'matched');
 payload.content=order.number+'123';assert.equal(matchOrder(state,parseTransaction(payload),now).status,'needs_review');
 payload.content=order.number;db.prepare('UPDATE shop_state SET body=?').run(JSON.stringify(state));receivePayment(db,parseTransaction(payload),'live',false,now);
 assert.equal(read(db).orders[0].paymentStatus,'unpaid');assert.equal(paymentDashboard(db).transactions[0].status,'needs_review');db.close();
});
test('Zalo sender is isolated from payment confirmation and retains failed jobs',async()=>{
 const {db,payload}=fixture();receivePayment(db,parseTransaction(payload),'live',true,now);const messages:string[]=[];
 assert.deepEqual(await deliverPaymentNotifications(db,async m=>{messages.push(m)},()=>false),{delivered:0,failed:0});
 assert.deepEqual(await deliverPaymentNotifications(db,async()=>{throw Error('Offline')},()=>true),{delivered:0,failed:1});assert.equal(read(db).orders[0].paymentStatus,'paid');
 db.prepare('UPDATE payment_outbox SET retry_at=0').run();
 assert.deepEqual(await deliverPaymentNotifications(db,async m=>{messages.push(m)},()=>true),{delivered:1,failed:0});
 assert.match(messages[0],/ĐÃ NHẬN THANH TOÁN/);assert.match(messages[0],/live:92704/);
 assert.deepEqual(await deliverPaymentNotifications(db,async m=>{messages.push(m)},()=>true),{delivered:0,failed:0});db.close();
});
test('streamed body limit applies even without Content-Length',async()=>{
 await assert.rejects(boundedBody(new Request('https://example.test',{method:'POST',body:'x'.repeat(100)}),50));
 assert.equal(new TextDecoder().decode(await boundedBody(new Request('https://example.test',{method:'POST',body:'{ }'}))),'{ }');
});
