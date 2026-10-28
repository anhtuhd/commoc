import {test} from 'node:test';
import assert from 'node:assert/strict';
import {initialState} from '../lib/shop/seed.ts';
import {cartKey,priceRange} from '../lib/shop/product-options.ts';
import {productSchema} from '../lib/shop/validation.ts';

test('cart identity normalizes selection order and distinguishes per-item notes',()=>{
 const a={id:'p',quantity:1,optionIds:['b','a'],note:'  ít ngọt '};
 assert.equal(cartKey(a),cartKey({...a,optionIds:['a','b'],note:'ít ngọt',quantity:2}));
 assert.notEqual(cartKey(a),cartKey({...a,note:'không đường'}));
});
test('configured price range reflects attainable required and optional choices',()=>{
 const p=initialState().products[0];p.optionGroups=[{id:'size',name:'Cỡ',type:'single',min:1,max:1,active:true,choices:[{id:'sm',name:'Nhỏ',priceDelta:-10000,active:true},{id:'lg',name:'Lớn',priceDelta:10000,active:true}]},{id:'topping',name:'Thêm',type:'multiple',min:0,max:1,active:true,choices:[{id:'nut',name:'Hạt',priceDelta:5000,active:true}]}];
 assert.deepEqual(priceRange(p),{min:p.price-10000,max:p.price+15000});assert.equal(productSchema.safeParse(p).success,true);
 p.optionGroups[1].choices[0].id='sm';assert.equal(productSchema.safeParse(p).success,false);
});
