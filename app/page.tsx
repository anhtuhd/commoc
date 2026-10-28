import {JsonLd} from '../components/shop/json-ld';
import StoreApp from '../components/shop/app';import {readPageState,publicData} from '../lib/shop/repository';
export const dynamic='force-dynamic';
export default async function Page(){const {state}=await readPageState();const data=publicData(state);return <><JsonLd data={data}/><StoreApp data={data}/></>}
