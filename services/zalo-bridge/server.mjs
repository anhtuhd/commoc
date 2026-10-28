import http from 'node:http';
import {mkdirSync,readFileSync,writeFileSync,renameSync,rmSync,existsSync} from 'node:fs';
import {timingSafeEqual} from 'node:crypto';
import {Zalo,ThreadType,LoginQRCallbackEventType} from 'zca-js';
const root=process.env.DATA_DIR||'/data',file=root+'/session.json',secret=process.env.ZALO_BRIDGE_SECRET;
if(!secret||secret.length<32)throw Error('Bridge secret required');
mkdirSync(root,{recursive:true,mode:0o700});
let api=null,generation=0,status={state:'disconnected',updatedAt:new Date().toISOString()},lastCommand='';
const update=v=>{status={...v,updatedAt:new Date().toISOString()}};
const client=()=>new Zalo({logging:false,selfListen:false});
async function connect(current){let saved;update({state:'connecting'});try{
 const connected=await client().loginQR({},event=>{if(current!==generation){event.actions?.abort();return}if(event.type===LoginQRCallbackEventType.QRCodeGenerated)update({state:'qr',qr:event.data.image,expiresAt:Date.now()+120000});if([LoginQRCallbackEventType.QRCodeExpired,LoginQRCallbackEventType.QRCodeDeclined].includes(event.type))event.actions.abort();if(event.type===LoginQRCallbackEventType.GotLoginInfo)saved=event.data});
 if(current!==generation)return;if(!saved)throw Error();writeFileSync(file+'.tmp',JSON.stringify(saved),{mode:0o600});renameSync(file+'.tmp',file);api=connected;update({state:'connected'});
 }catch{if(current===generation){api=null;update({state:'error',message:'Không thể kết nối hoặc QR đã hết hạn. Hãy tạo QR mới.'})}}}
if(existsSync(file)){const current=generation;client().login(JSON.parse(readFileSync(file,'utf8'))).then(v=>{if(current===generation){api=v;update({state:'connected'})}}).catch(()=>update({state:'error',message:'Kết nối lại bằng QR.'}))}
http.createServer(async(req,res)=>{res.setHeader('Content-Type','application/json');res.setHeader('Cache-Control','no-store');const token=Buffer.from(req.headers.authorization||''),expected=Buffer.from('Bearer '+secret);if(token.length!==expected.length||!timingSafeEqual(token,expected)){res.writeHead(403);res.end('{}');return}try{
 if(req.method==='GET'&&req.url==='/status'){res.end(JSON.stringify(status));return}
 if(req.method!=='POST')throw Error();let raw='';for await(const chunk of req){raw+=chunk;if(raw.length>16000)throw Error()}const b=JSON.parse(raw);
 if(req.url==='/command'){if(!['connect','disconnect'].includes(b.command)||typeof b.requestId!=='string')throw Error();if(b.requestId!==lastCommand){lastCommand=b.requestId;generation++;api=null;if(b.command==='disconnect'){rmSync(file,{force:true});update({state:'disconnected'})}else void connect(generation)}res.end('{"ok":true}');return}
 if(req.url==='/send'){if(!api||!/^0\d{9}$/.test(b.phone)||typeof b.message!=='string'||b.message.length>8000)throw Error();const recipient=await api.findUser(b.phone);await api.sendMessage(b.message,recipient.uid,ThreadType.User);res.end('{"ok":true}');return}res.writeHead(404);res.end('{}');
 }catch{res.writeHead(503);res.end('{"error":"Zalo operation failed"}')}}).listen(8081,'0.0.0.0');
