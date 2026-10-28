import Image from 'next/image';
import type {ImgHTMLAttributes} from 'react';
type Props=Omit<ImgHTMLAttributes<HTMLImageElement>,'src'|'width'|'height'|'srcSet'> & {src:string;width?:number;height?:number};
export function ShopImage({src,alt='',width=1200,height=1200,className='',sizes='(max-width: 600px) 100vw, (max-width: 900px) 50vw, 600px',...props}:Props){
 return <Image {...props} src={src} alt={alt} width={width} height={height} className={'shop-image '+className} sizes={sizes} unoptimized={process.env.NEXT_PUBLIC_RUNTIME!=='vps'||!src.startsWith('/')||src.endsWith('.svg')}/>;
}
