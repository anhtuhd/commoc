'use client';
import {useState} from 'react';
import type {Product} from '../../lib/shop/types';
import {cartKey,configurable,type CartItem} from '../../lib/shop/product-options';
import {useShop} from './context';

export function CartOptionsEditor({product,item}:{product:Product;item:CartItem}){
 const {setCart}=useShop(),[open,setOpen]=useState(false),[chosen,setChosen]=useState(item.optionIds||[]),[note,setNote]=useState(item.note||''),[error,setError]=useState('');
 if(!configurable(product))return null;
 const groups=(product.optionGroups||[]).filter(g=>g.active);
 function save(){
  if(!groups.every(g=>{const n=g.choices.filter(c=>c.active&&chosen.includes(c.id)).length;return n>=g.min&&n<=g.max})){setError('Vui lòng kiểm tra số lượng lựa chọn của từng nhóm.');return}
  const validIds=new Set(groups.flatMap(g=>g.choices.filter(c=>c.active).map(c=>c.id)));const revised={...item,optionIds:chosen.filter(id=>validIds.has(id)).sort(),note:product.noteEnabled?note.trim():''};
  setCart(old=>{const source=old.find(x=>cartKey(x)===cartKey(item));if(!source)return old;const other=old.find(x=>cartKey(x)===cartKey(revised)&&cartKey(x)!==cartKey(item));return other?old.filter(x=>cartKey(x)!==cartKey(item)).map(x=>cartKey(x)===cartKey(other)?{...x,quantity:x.quantity+source.quantity}:x):old.map(x=>cartKey(x)===cartKey(item)?revised:x)});setError('');setOpen(false);
 }
 return <div><button type="button" className="text-link" onClick={()=>{setChosen(item.optionIds||[]);setNote(item.note||'');setOpen(!open)}}>{open?'Đóng':'Sửa lựa chọn'}</button>{open&&<div className="panel stack">{groups.map(g=><fieldset key={g.id}><legend>{g.name} · Chọn {g.min}–{g.max}</legend>{g.choices.filter(c=>c.active).map(c=><label className="option-choice" key={c.id}><input type={g.type==='single'?'radio':'checkbox'} name={'cart-'+cartKey(item)+'-'+g.id} checked={chosen.includes(c.id)} onChange={e=>setChosen(old=>{const next=g.type==='single'?old.filter(id=>!g.choices.some(x=>x.id===id)):old.filter(id=>id!==c.id);return e.target.checked?[...next,c.id]:next})}/>{c.name}</label>)}{g.type==='single'&&g.min===0&&<button type="button" className="text-link" onClick={()=>setChosen(old=>old.filter(id=>!g.choices.some(c=>c.id===id)))}>Bỏ lựa chọn</button>}</fieldset>)}{product.noteEnabled&&<label>Ghi chú cho món<textarea maxLength={500} value={note} onChange={e=>setNote(e.target.value)}/></label>}{error&&<p role="alert" className="error">{error}</p>}<button type="button" className="button small" onClick={save}>Lưu lựa chọn</button></div>}</div>
}
