import {SITE_URL} from '../../lib/shop/structured-data';
import {readState} from '../../lib/shop/repository';
export const dynamic='force-dynamic';
export async function GET(){const {state}=await readState();const paths=['','/san-pham','/bo-suu-tap','/cau-chuyen','/lien-he','/bai-viet','/chinh-sach/giao-hang','/chinh-sach/doi-tra','/chinh-sach/bao-mat',...state.products.filter(p=>p.active).map(p=>'/san-pham/'+p.id),...state.articles.filter(a=>a.published).map(a=>'/bai-viet/'+a.id)];return new Response('<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">'+paths.map(p=>'<url><loc>'+(SITE_URL+p).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')+'</loc></url>').join('')+'</urlset>',{headers:{'Content-Type':'application/xml'}})}
