import {test} from 'node:test';
import assert from 'node:assert/strict';
import {InputError,customerErrors,fieldsFromError} from '../lib/shop/input-errors.ts';
test('customer validation maps invalid fields without dropping valid values',()=>{assert.deepEqual(customerErrors({name:'An',phone:'0912345678',address:'12 phố Huế',note:''}),{});assert.deepEqual(Object.keys(customerErrors({name:' ',phone:'123',address:'x',note:'x'.repeat(1001)})),['name','phone','address','note']);assert.ok(customerErrors(null).phone);});
test('API validation returns safe field messages and no arbitrary backend details',()=>{const fields={phone:'Kiểm tra số điện thoại'};assert.deepEqual(fieldsFromError(new InputError('bad',fields)),fields);assert.deepEqual(fieldsFromError({issues:[{path:['customer','phone'],message:'private raw input'}]}),{phone:'Nhập số điện thoại 10 số, bắt đầu bằng 0.'});assert.equal(fieldsFromError(new Error('business failure')),undefined);});
