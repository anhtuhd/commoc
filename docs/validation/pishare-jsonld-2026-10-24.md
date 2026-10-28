# JSON-LD production validation — 24/10/2026 (Asia/Saigon)

Source before fix: d75046d, https://pishare.cloud. Read-only comparison of published CMS data with server-rendered JSON-LD on 12 routes: homepage, all seven products, story, articles, shipping and returns policies. All matched; no references to the previous Sites domain. No production content was modified for tests.

## External validators

- Schema.org Validator, URL mode, `/san-pham/com-tuoi`: 0 errors, 0 warnings; Product (with Brand), BreadcrumbList.
- Schema.org Validator, URL mode, `/`: 0 errors, 0 warnings; WebSite with Organization and Brand.
- Google Rich Results Test initially rejected URL mode with “Log in and try again”. Code mode succeeded using the exact JSON-LD extracted from the live page, wrapped in a minimal HTML document. This checks markup eligibility, not Google's ability to crawl/render the whole URL.
- Product: 3 valid items (Product snippets, Merchant listings, Breadcrumbs). Result: https://search.google.com/test/rich-results/result?id=iRmX8RtABlX7jb1bfPOf5Q
- Optional product warnings: `review`, `aggregateRating`. Optional merchant warnings: `hasMerchantReturnPolicy`, `shippingDetails`. The published CMS has no verified business/shipping/return-policy fields and no reviews. These fields remain omitted rather than populated with sample data.
- Homepage: “No items detected” in Google's rich-results tool; no claim that every Schema.org type is a Google rich-result feature. Schema.org validation above confirms the Brand/Organization/WebSite graph is valid. Result: https://search.google.com/test/rich-results/result?id=eCZre91oLBEdBAH3_0HFlA

## Corrected CMS synchronization issue

A custom published `journalTitle` changed the article heading but left the visible/JSON-LD breadcrumb, footer link and SEO title at “Chuyện bên chén trà”. They now use one shared published label; blank values fall back to the existing default. Draft content is not exported. Regression test first failed before the fix, then passed.

Checks: 29 unit tests passed; TypeScript passed; Next.js production webpack build passed. Existing tests cover changed brand/logo, product price/stock, quote-only products, unpublished content, private routes, script escaping, verified/unverified addresses and policies, free-shipping thresholds, refundType and draft isolation.

Renamed-journal regression fixture (not published to production): Google Code mode detects 1 valid Breadcrumb item. Result: https://search.google.com/test/rich-results/result?id=REn6zg0VPQLorrpekswQag . Generic vs Google-specific validation distinction: https://developers.google.com/search/docs/appearance/structured-data .
