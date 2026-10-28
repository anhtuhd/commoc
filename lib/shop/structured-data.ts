import {reviewSummary} from './review-summary.ts';
import {breadcrumbs} from './breadcrumbs.ts';
export {breadcrumbs} from './breadcrumbs.ts';
import {business} from './business.ts';
import type {PublicData} from './types.ts';

export const SITE_URL=process.env.NEXT_PUBLIC_SITE_URL||'https://moc-vi-com-viet-231026.spunky-dune-5060.chatgpt.site';
const absolute=(path:string)=>new URL(path,SITE_URL).href;
export function structuredData(data:PublicData,segments:string[]=[]){
 const b=data.brand,details=business(b);
 const address=details.addressConfirmed?{'@type':'PostalAddress',streetAddress:details.street,addressLocality:details.locality,addressRegion:details.region,postalCode:details.postalCode,addressCountry:details.country}:null;
 const returns=details.returnsConfirmed?{'@type':'MerchantReturnPolicy',applicableCountry:'VN',returnPolicyCategory:'https://schema.org/'+({none:'MerchantReturnNotPermitted',finite:'MerchantReturnFiniteReturnWindow',unlimited:'MerchantReturnUnlimitedWindow'}[details.returnCategory as 'none'|'finite'|'unlimited']),merchantReturnLink:absolute('/chinh-sach/doi-tra'),...(details.returnCategory==='finite'?{merchantReturnDays:Number(details.returnDays)}:{}),...(details.returnCategory!=='none'?{...(details.refundType?{refundType:'https://schema.org/'+details.refundType}:{}),returnMethod:'https://schema.org/'+(details.returnMethod==='mail'?'ReturnByMail':'ReturnInStore'),returnFees:'https://schema.org/'+(details.returnFees==='free'?'FreeReturn':'ReturnFeesCustomerResponsibility')}:{})}:null;
 const shipping=details.shippingConfirmed&&data.shipping.enabled?{'@type':'OfferShippingDetails',shippingSettingsLink:absolute('/chinh-sach/giao-hang'),shippingDestination:{'@type':'DefinedRegion',addressCountry:'VN',addressRegion:data.shipping.regionLabel},shippingRate:{'@type':'MonetaryAmount',currency:'VND',value:data.shipping.fee},deliveryTime:{'@type':'ShippingDeliveryTime',handlingTime:{'@type':'QuantitativeValue',minValue:Number(details.handlingMin),maxValue:Number(details.handlingMax),unitCode:'DAY'},transitTime:{'@type':'QuantitativeValue',minValue:Number(details.transitMin),maxValue:Number(details.transitMax),unitCode:'DAY'}}}:null;
 const brand={'@type':'Brand','@id':SITE_URL+'/#brand',name:b.name,...(b.logo?{logo:absolute(b.logo)}:{})};
 const graph:Record<string,unknown>[]=[];
 if(!segments.length){
  graph.push(brand,{'@type':'Organization','@id':SITE_URL+'/#organization',name:b.name,url:SITE_URL+'/',description:b.description,brand:{'@id':brand['@id']},...(b.logo?{logo:absolute(b.logo)}:{}),...(b.phone?{telephone:b.phone}:{}),...(b.email?{email:b.email}:{}),...(address?{address}:{}),...(returns?{hasMerchantReturnPolicy:returns}:{}),...([b.facebook,b.zalo].some(Boolean)?{sameAs:[b.facebook,b.zalo].filter(Boolean)}:{})});
  graph.push({'@type':'WebSite','@id':SITE_URL+'/#website',name:b.name,url:SITE_URL+'/',inLanguage:'vi-VN',publisher:{'@id':SITE_URL+'/#organization'}});
 }
 if(segments.join('/')==='chinh-sach/giao-hang'&&shipping)graph.push({'@type':'ShippingRateSettings',shippingDestination:shipping.shippingDestination,shippingRate:shipping.shippingRate,freeShippingThreshold:{'@type':'MonetaryAmount',currency:'VND',value:data.shipping.freeFrom}});
 const crumbs=breadcrumbs(data,segments);
 if(crumbs.length){graph.push({'@type':'BreadcrumbList','@id':absolute('/'+segments.join('/'))+'#breadcrumb',itemListElement:crumbs.map((c,i)=>({'@type':'ListItem',position:i+1,name:c.name,item:absolute(c.path)}))})}
 const p=segments.length===2&&segments[0]==='san-pham'?data.products.find(p=>p.id===segments[1]&&p.active):null;
 if(p)graph.push(brand);
 const reviews=p?reviewSummary(data.reviews,p.id):null;
 // Google requires an Offer or real reviews; quote-only pages retain Brand/breadcrumb.
 if(p&&((p.mode==='fixed'&&p.price>0&&data.shipping.enabled)||reviews?.count)){const url=absolute('/san-pham/'+p.id);graph.push({'@type':'Product','@id':url+'#product',url,...(reviews?.count?{aggregateRating:{'@type':'AggregateRating',ratingValue:reviews.rating,reviewCount:reviews.count,bestRating:5,worstRating:1},review:reviews.items.map(r=>({'@type':'Review','@id':url+'#review-'+r.id,author:{'@type':'Person',name:r.author},datePublished:r.createdAt,reviewBody:r.body,reviewRating:{'@type':'Rating',ratingValue:r.rating,bestRating:5,worstRating:1}}))}:{}),name:p.name,description:p.description,sku:p.sku,category:p.category,...(p.image?{image:[absolute(p.image)]}:{}),brand:{'@id':brand['@id']},...(p.mode==='fixed'&&p.price>0&&data.shipping.enabled?{offers:{'@type':'Offer',url,priceCurrency:'VND',price:p.price,...(returns?{hasMerchantReturnPolicy:returns}:{}),...(shipping?{shippingDetails:{...shipping,shippingRate:{...shipping.shippingRate,value:p.price>=data.shipping.freeFrom?0:data.shipping.fee}}}:{}),availability:p.stock>0?'https://schema.org/InStock':'https://schema.org/OutOfStock',seller:{'@type':'Organization',name:b.name,'@id':SITE_URL+'/#organization'}}}:{})})}
 return graph.length?{'@context':'https://schema.org','@graph':graph}:null;
}
// Prevent CMS text from ending the script element; JSON.parse preserves the text.
export function serializeJsonLd(value:unknown){return JSON.stringify(value).replace(/</g,'\\u003c').replace(/\u2028/g,'\\u2028').replace(/\u2029/g,'\\u2029')}
