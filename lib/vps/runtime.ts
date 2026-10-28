// Node adapter used only by the VPS Next.js build. Sites keeps native D1/R2.
import {DatabaseSync} from 'node:sqlite';
import {mkdirSync} from 'node:fs';
import {readFile,writeFile} from 'node:fs/promises';
import path from 'node:path';
const root=process.env.DATA_DIR||'/data';
let connection:DatabaseSync|undefined;
export function db(){if(!connection){mkdirSync(root,{recursive:true});connection=new DatabaseSync(path.join(root,'shop.sqlite'));connection.exec('PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; CREATE TABLE IF NOT EXISTS shop_state (id TEXT PRIMARY KEY, body TEXT NOT NULL, revision INTEGER NOT NULL DEFAULT 0)');}return connection}
function prepared(sql:string,values:any[]=[]):any{return {bind:(...args:any[])=>prepared(sql,args),first:async()=>db().prepare(sql).get(...values)||null,run:async()=>({meta:{changes:Number(db().prepare(sql).run(...values).changes)}})}}
function asset(key:string){if(!/^[a-f0-9-]+\.(png|jpg|webp)$/.test(key))throw Error('Invalid media key');return path.join(root,'media',key)}
export const env={OWNER_SETUP_ENABLED:'false',VPS_OWNER_ID:process.env.ADMIN_USER||'',VPS_OWNER_EMAIL:process.env.ADMIN_EMAIL||'',DB:{prepare:prepared},BUCKET:{
 async get(key:string){try{return {body:new Uint8Array(await readFile(asset(key))),httpMetadata:{contentType:key.endsWith('.png')?'image/png':key.endsWith('.jpg')?'image/jpeg':'image/webp'}}}catch(e:any){if(e.code==='ENOENT')return null;throw e}},
 async put(key:string,bytes:Uint8Array){mkdirSync(path.join(root,'media'),{recursive:true});await writeFile(asset(key),bytes,{flag:'wx',mode:0o600});}
}};
