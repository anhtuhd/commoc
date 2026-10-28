export async function boundedBody(req:Request,max=65536):Promise<Uint8Array>{
 if(Number(req.headers.get('content-length')||0)>max)throw Error('Payload too large');
 const reader=req.body?.getReader();if(!reader)return new Uint8Array();
 const chunks:Uint8Array[]=[];let size=0;
 try{for(;;){const part=await reader.read();if(part.done)break;size+=part.value.length;if(size>max){await reader.cancel();throw Error('Payload too large')}chunks.push(part.value)}}finally{reader.releaseLock()}
 const result=new Uint8Array(size);let offset=0;for(const chunk of chunks){result.set(chunk,offset);offset+=chunk.length}return result;
}
