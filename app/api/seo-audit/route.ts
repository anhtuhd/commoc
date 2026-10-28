import {getChatGPTUser} from '../../chatgpt-auth';
import {readState,publicData} from '../../../lib/shop/repository';
import {authorize} from '../../../lib/shop/access';
import {SITE_URL} from '../../../lib/shop/structured-data';
import {auditHtml} from '../../../lib/diagnostics/seo';
export const dynamic='force-dynamic';
let lastRun=0;
export async function GET(){const user=await getChatGPTUser();if(!user)return Response.json({error:'Vui lòng đăng nhập.'},{status:401});const {state}=await readState();try{authorize(state,user.userId,['owner']);}catch{return Response.json({error:'Không có quyền.'},{status:403});}if(Date.now()-lastRun<30000)return Response.json({error:'Chờ 30 giây trước khi kiểm tra lại.'},{status:429});lastRun=Date.now();
 const data=publicData(state),base=process.env.COMMO_SITE_ORIGIN||SITE_URL;
 const paths=['/','/san-pham','/bo-suu-tap','/cau-chuyen','/bai-viet','/lien-he','/chinh-sach/giao-hang','/chinh-sach/doi-tra','/chinh-sach/bao-mat',...data.products.map(p=>'/san-pham/'+p.id),...data.articles.map(a=>'/bai-viet/'+a.id)].slice(0,40);
 const pages=[];for(let i=0;i<paths.length;i+=3){pages.push(...await Promise.all(paths.slice(i,i+3).map(async path=>{try{const start=performance.now(),r=await fetch(new URL(path,base),{signal:AbortSignal.timeout(10000),redirect:'error',cache:'no-store'});const html=await r.text();return {...auditHtml(html,path,SITE_URL,r.status),responseMs:Math.round(performance.now()-start)};}catch{return {path,error:'Không tải được trang trong 10 giây.'};}})));}
 for(const page of pages){if('title' in page){const ok=pages.filter(p=>'title' in p&&p.title===page.title).length===1;page.checks.push({label:'Tiêu đề không trùng trang khác',ok});page.total++;if(ok)page.passed++;}}
 const files=await Promise.all(['/robots.txt','/sitemap.xml'].map(async path=>{try{const r=await fetch(new URL(path,base),{signal:AbortSignal.timeout(10000),redirect:'error'});return {path,ok:r.ok};}catch{return {path,ok:false};}}));
 return Response.json({at:new Date().toISOString(),pages,files,site:SITE_URL,scope:'Kiểm tra kỹ thuật HTML, không phải điểm xếp hạng Google.'},{headers:{'Cache-Control':'no-store'}});
}
