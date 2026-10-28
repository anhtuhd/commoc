import lighthouse from 'lighthouse';
import * as chromeLauncher from 'chrome-launcher';
import {mkdir,writeFile} from 'node:fs/promises';
const url=process.argv[2]||'https://pishare.cloud';
const output=process.argv[3]||'work/performance/latest';
if(!/^https?:\/\//.test(url))throw Error('Provide an http(s) URL');
await mkdir(output,{recursive:true});
const chrome=await chromeLauncher.launch({chromeFlags:['--headless','--no-first-run','--disable-extensions']});
try{const result=await lighthouse(url,{port:chrome.port,output:['html','json'],onlyCategories:['performance','accessibility','best-practices','seo'],logLevel:'error'});await writeFile(output+'/report.html',result.report[0]);await writeFile(output+'/report.json',result.report[1]);console.log(JSON.stringify({url,scores:Object.fromEntries(Object.entries(result.lhr.categories).map(([k,v])=>[k,Math.round(v.score*100)])),metrics:Object.fromEntries(['first-contentful-paint','largest-contentful-paint','total-blocking-time','cumulative-layout-shift','speed-index'].map(k=>[k,result.lhr.audits[k].displayValue])),failures:Object.entries(result.lhr.audits).filter(([,a])=>a.score!==null&&a.score<1&&a.details).map(([id,a])=>({id,title:a.title,value:a.displayValue})),report:output+'/report.html'},null,2));}finally{await chrome.kill()}
