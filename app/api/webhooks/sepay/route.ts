import {parseTransaction,verifySignature} from '../../../../lib/payments/sepay';
import {receivePayment} from '../../../../lib/payments/repository';
import {paymentDB,sepayEnvironment} from '../../../../lib/payments/server';
import {boundedBody} from '../../../../lib/payments/http';
export const runtime='nodejs';
export const dynamic='force-dynamic';
const reply=(success:boolean,status:number)=>Response.json({success},{status,headers:{'Cache-Control':'no-store'}});
export async function POST(req:Request){
 const start=performance.now(),config=sepayEnvironment();
 if(!config.configured)return reply(false,503);
 let raw:Uint8Array;try{raw=await boundedBody(req)}catch{return reply(false,413)}
 if(!verifySignature(raw,req.headers.get('x-sepay-timestamp'),req.headers.get('x-sepay-signature'),config.secret)){
  console.warn(JSON.stringify({event:'sepay_webhook',status:'unauthorized'}));return reply(false,401);
 }
 let transaction;try{transaction=parseTransaction(JSON.parse(new TextDecoder('utf-8',{fatal:true}).decode(raw)))}catch{return reply(false,400)}
 try{
  const result=receivePayment(paymentDB(),transaction,config.mode,config.autoMatch);
  console.info(JSON.stringify({event:'sepay_webhook',transactionId:transaction.id,...result,durationMs:Math.round(performance.now()-start)}));
  return reply(true,200);
 }catch{console.error(JSON.stringify({event:'sepay_webhook',transactionId:transaction.id,status:'storage_error'}));return reply(false,503)}
}
