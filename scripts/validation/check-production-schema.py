import subprocess,json,re,concurrent.futures
base='https://pishare.cloud'
def get(path):
 return subprocess.check_output(['curl','-fsS','--max-time','30',base+path],text=True)
data=json.loads(get('/api/store/public'))
def check(path):
 html=get(path);graphs=[]
 for raw in re.findall(r'<script[^>]*type="application/ld\+json"[^>]*>(.*?)</script>',html,re.S):
  value=json.loads(raw);graphs.extend(value.get('@graph',[value]))
 if path in ['/quan-tri','/thanh-toan','/tra-cuu']:
  assert not graphs,path
 elif path.startswith('/san-pham/'):
  p=next(p for p in data['products'] if path.endswith('/'+p['id']))
  prod=next((x for x in graphs if x['@type']=='Product'),None)
  if p['mode']=='quote':
   assert prod is None and any(x['@type']=='BreadcrumbList' for x in graphs)
   return {'path':path,'types':[x['@type'] for x in graphs],'ok':True}
  assert prod['name']==p['name']
  brand=next(x for x in graphs if x.get('@id')==prod['brand']['@id']);assert brand['name']==data['brand']['name']
  if p['mode']=='fixed':
   assert float(prod['offers']['price'])==p['price'];assert prod['offers']['priceCurrency']=='VND'
   assert prod['offers']['availability'].endswith('InStock' if p['stock']>0 else 'OutOfStock')
  else:assert 'offers' not in prod
  reviews=[x for x in data['reviews'] if x['productId']==p['id']]
  if not reviews:assert 'aggregateRating' not in prod and 'review' not in prod
  crumbs=next(x for x in graphs if x['@type']=='BreadcrumbList')['itemListElement']
  assert [x['position'] for x in crumbs]==list(range(1,len(crumbs)+1))
  assert crumbs[-1]['item']==base+path
 elif path=='/':
  assert next(x for x in graphs if x['@type']=='Organization')['name']==data['brand']['name']
 assert 'moc-vi-com-viet-231026.spunky-dune-5060.chatgpt.site' not in ''.join(json.dumps(g) for g in graphs)
 return {'path':path,'types':[x['@type'] for x in graphs],'ok':True}
paths=['/']+['/san-pham/'+p['id'] for p in data['products'] if p['active']]+['/quan-tri','/thanh-toan','/tra-cuu']
with concurrent.futures.ThreadPoolExecutor(max_workers=3) as pool:results=list(pool.map(check,paths))
print(json.dumps(results,ensure_ascii=False,indent=2))
