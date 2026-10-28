import {journalTitle} from './labels.ts';
import type {PublicData} from './types.ts';
const privateRoutes=new Set(['quan-tri','gio-hang','thanh-toan','tra-cuu','dat-hang-thanh-cong']);
export function breadcrumbs(data:PublicData,segments:string[]){
 if(!segments.length||privateRoutes.has(segments[0]))return [];
 const labels:Record<string,string>={'san-pham':'Thức quà','bo-suu-tap':'Bộ sưu tập','cau-chuyen':'Câu chuyện','lien-he':'Liên hệ','bai-viet':journalTitle(data.brand)};
 const policies:Record<string,string>={'giao-hang':'Giao hàng & thanh toán','doi-tra':'Đổi trả & hỗ trợ','bao-mat':'Bảo mật thông tin'};
 const items=[{name:'Trang chủ',path:'/'}];
 if(segments[0]==='chinh-sach'&&segments.length===2&&policies[segments[1]])return [...items,{name:policies[segments[1]],path:'/'+segments.join('/')}];
 if(!labels[segments[0]])return [];
 items.push({name:labels[segments[0]],path:'/'+segments[0]});
 if(segments.length===1)return items;
 const detail=segments.length===2?(segments[0]==='san-pham'?data.products.find(p=>p.id===segments[1]&&p.active)?.name:segments[0]==='bai-viet'?data.articles.find(a=>a.id===segments[1]&&a.published)?.title:null):null;
 return detail?[...items,{name:detail,path:'/'+segments.join('/')}]:[];
}
