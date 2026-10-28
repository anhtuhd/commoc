const base=()=>process.env.GO_API_URL||'http://127.0.0.1:8080';
export async function backend<T=unknown>(path:string,init:RequestInit={}){
 const response=await fetch(new URL(path,base()),{...init,cache:'no-store',signal:init.signal||AbortSignal.timeout(20000)});
 if(!response.ok)throw Error(`Backend unavailable (${response.status})`);
 return response.json() as Promise<T>;
}
export async function proxy(req:Request){
 const url=new URL(req.url),headers=new Headers();
 for(const key of ['content-type','cookie','origin','x-commo-client-ip','x-sepay-timestamp','x-sepay-signature']){const v=req.headers.get(key);if(v)headers.set(key,v)}
 try{const r=await fetch(new URL(url.pathname+url.search,base()),{method:req.method,headers,body:['GET','HEAD'].includes(req.method)?undefined:await req.arrayBuffer(),redirect:'manual',cache:'no-store',signal:AbortSignal.timeout(25000)});
 const result=new Headers();for(const key of ['content-type','cache-control','set-cookie','location','x-request-id','server-timing','x-content-type-options','referrer-policy']){const v=r.headers.get(key);if(v)result.set(key,v)}
 return new Response(r.body,{status:r.status,headers:result});
 }catch{return Response.json({error:'Máy chủ đang bận. Vui lòng thử lại.'},{status:503})}
}

export type Session={user:{id:string;name:string;email:string}|null;role:string|null};
