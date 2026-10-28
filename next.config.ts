import type {NextConfig} from 'next';
import path from 'node:path';
const vps=process.env.COMMO_RUNTIME==='vps';
const nextConfig:NextConfig={async headers(){return [{source:'/images/:path*',headers:[{key:'Cache-Control',value:'public, max-age=604800, stale-while-revalidate=86400'}]},{source:'/fonts/:path*',headers:[{key:'Cache-Control',value:'public, max-age=31536000, immutable'}]}]},...(vps?{output:'standalone',experimental:{cpus:1},webpack(config,{webpack}){config.plugins.push(new webpack.NormalModuleReplacementPlugin(/^cloudflare:workers$/,path.resolve('lib/go/runtime.ts')));return config}}:{})};
export default nextConfig;
