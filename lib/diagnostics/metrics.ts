import {z} from 'zod';
export const metricNames=['LCP','INP','CLS','FCP','TTFB','NAV','API'] as const;
import {publicPath} from './paths.ts';
export const metricSchema=z.object({id:z.string().regex(/^[\w-]{1,100}$/),name:z.enum(metricNames),path:z.string().max(200).refine(p=>publicPath(p)===p),value:z.number().finite().min(0).max(120000),device:z.enum(['mobile','desktop'])}).superRefine((m,c)=>{if(m.name==='CLS'&&m.value>10)c.addIssue({code:'custom',message:'Invalid CLS'});});
export type Sample=z.infer<typeof metricSchema>&{at?:number};
const limits:Record<string,[number,number]>={LCP:[2500,4000],INP:[200,500],CLS:[.1,.25],FCP:[1800,3000],TTFB:[800,1800],NAV:[1000,2500],API:[500,1500]};
export function rating(name:string,value:number){const [good,poor]=limits[name]||[1000,2500];return value<=good?'good':value<=poor?'needs-improvement':'poor';}
export function summarize(samples:Sample[]){const groups=new Map<string,Sample[]>();for(const s of samples){const key=[s.name,s.path,s.device].join('|');const group=groups.get(key);if(group)group.push(s);else groups.set(key,[s]);}return [...groups.values()].map(rows=>{const values=rows.map(r=>r.value).sort((a,b)=>a-b),p75=values[Math.ceil(values.length*.75)-1];return {name:rows[0].name,path:rows[0].path,device:rows[0].device,count:rows.length,p75,rating:rating(rows[0].name,p75)};}).sort((a,b)=>b.count-a.count);}
