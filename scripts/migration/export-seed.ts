import {initialState} from '../../lib/shop/seed.ts';
import {writeFileSync} from 'node:fs';
// Development fixtures only. Never use this seed to replace an existing store.
writeFileSync('backend/internal/shop/seed.json',JSON.stringify(initialState(),null,2));
