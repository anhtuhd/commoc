import {cache} from 'react';
import {backend} from '../go/client';
import type {ShopState,PublicData} from './types';
// Server rendering only consumes the public projection from Go. No database or secrets in Next.
export async function readState():Promise<{state:ShopState;revision:number}>{
 const data=await backend('/api/store/public') as PublicData;
 return {state:{...data,brandHistory:[],orders:[],inquiries:[],admins:[],audit:[],coupons:[],events:[],rate:{}} as ShopState,revision:0};
}
export const readPageState=cache(readState);
export function publicData(s:ShopState):PublicData{return {brand:s.brand,products:s.products,collections:s.collections,campaigns:s.campaigns,articles:s.articles,shipping:s.shipping,reviews:s.reviews||[]}}
