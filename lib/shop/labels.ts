import type {Brand} from './types.ts';
// One published label for headings, navigation, metadata and breadcrumb JSON-LD.
export function journalTitle(brand:Pick<Brand,'journalTitle'>){
 return brand.journalTitle?.trim()||'Chuyện bên chén trà';
}
