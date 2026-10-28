import {db} from '../vps/runtime.ts';
import {initializePayments} from './sqlite.ts';
export function paymentDB(){const connection=db();initializePayments(connection);return connection}
export function sepayEnvironment(){
 const mode=process.env.SEPAY_MODE||'live',secret=process.env.SEPAY_WEBHOOK_SECRET||'';
 const validMode=mode==='live'||mode==='test';
 const productionTest=mode==='test'&&process.env.NEXT_PUBLIC_SITE_URL==='https://pishare.cloud';
 return {mode:mode as 'live'|'test',secret,configured:process.env.COMMO_RUNTIME==='vps'&&validMode&&!productionTest&&secret.length>=32,autoMatch:process.env.SEPAY_AUTO_MATCH!=='false'};
}
