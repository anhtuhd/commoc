import type {PublicReview} from './types.ts';
export function reviewSummary(reviews:PublicReview[]|undefined,productId:string){
 const items=(reviews||[]).filter(r=>r.productId===productId);
 return {items,count:items.length,rating:items.length?Math.round(items.reduce((n,r)=>n+r.rating,0)/items.length*100)/100:null};
}
