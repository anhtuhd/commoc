import type {Product} from './types';
export const productOptionsEnabled=process.env.NEXT_PUBLIC_PRODUCT_OPTIONS_ENABLED==='1';
export type CartItem={id:string;quantity:number;optionIds?:string[];note?:string};
export const cartKey=(item:CartItem)=>JSON.stringify([item.id,[...(item.optionIds||[])].sort(),(item.note||'').trim()]);
export const configurable=(p:Product)=>(p.optionGroups||[]).some(g=>g.active)||!!p.noteEnabled;
export function selectedChoices(p:Product|undefined,item:CartItem){return (p?.optionGroups||[]).flatMap(g=>g.choices.filter(c=>(item.optionIds||[]).includes(c.id)).map(c=>({group:g.name,...c})));}
export const itemPrice=(p:Product|undefined,item:CartItem)=>(p?.price||0)+selectedChoices(p,item).reduce((n,c)=>n+c.priceDelta,0);
export function priceRange(p:Product){let min=p.price,max=p.price;for(const g of p.optionGroups||[]){if(!g.active)continue;const prices=g.choices.filter(c=>c.active).map(c=>c.priceDelta).sort((a,b)=>a-b);if(prices.length<g.min)return null;min+=prices.slice(0,g.min).reduce((a,b)=>a+b,0)+prices.slice(g.min,g.max).filter(x=>x<0).reduce((a,b)=>a+b,0);const descending=[...prices].reverse();max+=descending.slice(0,g.min).reduce((a,b)=>a+b,0)+descending.slice(g.min,g.max).filter(x=>x>0).reduce((a,b)=>a+b,0)}return {min,max}}
