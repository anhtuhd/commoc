import type {ShopState} from './types.ts';
export const roleFor=(s:ShopState,id:string)=>s.admins.find(a=>a.id===id)?.role;
export function authorize(s:ShopState,id:string,roles:string[]){if(!roles.includes(roleFor(s,id)||''))throw Error('Bạn không có quyền thực hiện thao tác này.')}
