import type {PublicData} from '../../lib/shop/types';
import {structuredData,serializeJsonLd} from '../../lib/shop/structured-data';
export function JsonLd({data,segments=[]}:{data:PublicData;segments?:string[]}){
 const schema=structuredData(data,segments);
 return schema?<script type="application/ld+json" dangerouslySetInnerHTML={{__html:serializeJsonLd(schema)}}/>:null;
}
