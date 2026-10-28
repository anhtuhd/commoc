import {SITE_URL} from '../../lib/shop/structured-data';
export function GET(){return new Response('User-agent: *\nAllow: /\nDisallow: /quan-tri\nDisallow: /api/\nDisallow: /thanh-toan\nDisallow: /tra-cuu\nDisallow: /gio-hang\nDisallow: /dat-hang-thanh-cong\nDisallow: /dang-nhap\nSitemap: '+SITE_URL+'/sitemap.xml\n',{headers:{'Content-Type':'text/plain'}})}
