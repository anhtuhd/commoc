import {test} from 'node:test';
import assert from 'node:assert/strict';
import {initialState} from '../lib/shop/seed.ts';
import {structuredData,serializeJsonLd,breadcrumbs,SITE_URL} from '../lib/shop/structured-data.ts';
const nodes=(s=initialState(),path:string[]=[])=>structuredData(s,path)?.['@graph'] as any[];
test('published brand drives organization, product brand and absolute uploaded logo',()=>{
 const s=initialState();s.brand.draft={name:'Nháp không công khai'};
 assert.equal(nodes(s).find(n=>n['@type']==='Organization').name,'Mộc Vị');
 Object.assign(s.brand,{name:'Cốm Mới',logo:'/api/store/media/logo.webp',phone:'0901234567'});
 const org=nodes(s).find(n=>n['@type']==='Organization');assert.equal(org.name,'Cốm Mới');assert.equal(org.logo,SITE_URL+'/api/store/media/logo.webp');assert.equal(org.telephone,'0901234567');assert.ok(!serializeJsonLd(nodes(s)).includes('Nháp không công khai'));
 assert.equal(nodes(s,['san-pham','com-tuoi']).find(n=>n['@type']==='Brand').name,'Cốm Mới');
});
test('product uses live SKU, price and stock with no invented reviews',()=>{
 const s=initialState();s.products[0].price=155000;s.products[0].stock=0;
 const p=nodes(s,['san-pham','com-tuoi']).find(n=>n['@type']==='Product');
 assert.equal(p.offers.price,155000);assert.equal(p.offers.priceCurrency,'VND');assert.equal(p.offers.availability,'https://schema.org/OutOfStock');assert.equal(p.sku,s.products[0].sku);assert.equal(p.aggregateRating,undefined);
 s.products[0].stock=3;assert.equal(nodes(s,['san-pham','com-tuoi']).find(n=>n['@type']==='Product').offers.availability,'https://schema.org/InStock');
 s.shipping.enabled=false;assert.equal(nodes(s,['san-pham','com-tuoi']).find(n=>n['@type']==='Product'),undefined);
});
test('quote products do not advertise zero price; hidden products and private routes have no schema',()=>{
 const s=initialState();assert.equal(nodes(s,['san-pham','qua-doanh-nghiep']).find(n=>n['@type']==='Product'),undefined);
 s.products[0].active=false;assert.equal(structuredData(s,['san-pham','com-tuoi']),null);
 for(const route of ['quan-tri','thanh-toan','tra-cuu','gio-hang','dat-hang-thanh-cong'])assert.equal(structuredData(s,[route]),null);
});
test('breadcrumb hierarchy matches visible labels and live product/article titles',()=>{
 const s=initialState();s.products[0].name='Cốm mới';const path=['san-pham','com-tuoi'];const visible=breadcrumbs(s,path);const schema=nodes(s,path).find(n=>n['@type']==='BreadcrumbList').itemListElement;
 assert.deepEqual(schema.map((x:any)=>x.name),visible.map(x=>x.name));assert.deepEqual(schema.map((x:any)=>x.position),[1,2,3]);assert.equal(schema[2].name,'Cốm mới');assert.equal(schema[2].item,SITE_URL+'/san-pham/com-tuoi');assert.equal(breadcrumbs(s,['chinh-sach','bao-mat']).length,2);assert.equal(breadcrumbs(s,['bai-viet',s.articles[0].id])[2].name,s.articles[0].title);
});
test('CMS script injection is escaped while JSON retains original content',()=>{
 const data={name:'</script><script>alert(1)</script>\u2028'};const serialized=serializeJsonLd(data);assert.ok(!serialized.includes('<'));assert.deepEqual(JSON.parse(serialized),data);
});

test('published journal heading drives visible and structured breadcrumbs, draft stays private',()=>{
 const s=initialState();s.brand.journalTitle='Tin từ cửa hàng';s.brand.draft={journalTitle:'Bản nháp chưa công bố'};
 assert.equal(breadcrumbs(s,['bai-viet'])[1].name,'Tin từ cửa hàng');
 assert.equal(breadcrumbs(s,['bai-viet',s.articles[0].id])[1].name,'Tin từ cửa hàng');
 const schema=nodes(s,['bai-viet']).find(n=>n['@type']==='BreadcrumbList');
 assert.equal(schema.itemListElement[1].name,'Tin từ cửa hàng');
 s.brand.journalTitle='';assert.equal(breadcrumbs(s,['bai-viet'])[1].name,'Chuyện bên chén trà');
});
