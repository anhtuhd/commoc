import {PerformanceMonitor} from '../components/diagnostics/monitor';
import {SITE_URL} from '../lib/shop/structured-data';
import type {Metadata,Viewport} from 'next';import {readPageState} from '../lib/shop/repository';import './fonts.css';import './globals.css';import './responsive.css';
export const dynamic='force-dynamic';
export async function generateMetadata():Promise<Metadata>{const {state}=await readPageState();const b=state.brand;return {alternates:{canonical:'/'},title:b.seoTitle||b.name,description:b.seoDescription,icons:{icon:b.favicon||'/favicon.svg'},openGraph:{siteName:b.name,title:b.seoTitle||b.name,description:b.seoDescription,images:[b.shareImage]},metadataBase:new URL(SITE_URL)}}
export async function generateViewport():Promise<Viewport>{return {themeColor:(await readPageState()).state.brand.color}}
export default function RootLayout({children}:{children:React.ReactNode}){return <html lang="vi"><body><PerformanceMonitor/>{children}</body></html>}
