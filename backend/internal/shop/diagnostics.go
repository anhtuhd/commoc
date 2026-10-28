package shop

import (
	"math"
	"net/http"
	"sort"
	"strings"
)

func (a *App) diagnostics(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if r.Header.Get("Origin") != a.Site {
			http.Error(w, "Forbidden", 403)
			return
		}
		b, e := body(w, r, 2048)
		name, path, value, device := str(b["name"]), str(b["path"]), num(b["value"]), str(b["device"])
		if e != nil || !match(`^[\w-]{1,100}$`, str(b["id"])) || !contains([]string{"LCP", "INP", "CLS", "FCP", "TTFB", "NAV", "API"}, name) || value < 0 || value > 120000 || (name == "CLS" && value > 10) || !contains([]string{"mobile", "desktop"}, device) || len(path) > 200 || !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "?#") || !metricPath(path) {
			http.Error(w, "Invalid metric", 400)
			return
		}
		if e = a.rate(r.Context(), "metric:"+clientKey(r), 1200); e != nil {
			http.Error(w, "Too many requests", 429)
			return
		}
		_, e = a.DB.Pool.Exec(r.Context(), `INSERT INTO performance_samples(id,name,path,value,device) VALUES($1,$2,$3,$4,$5) ON CONFLICT(id) DO UPDATE SET value=excluded.value WHERE performance_samples.name=excluded.name AND performance_samples.path=excluded.path AND performance_samples.device=excluded.device`, b["id"], name, path, value, device)
		if e != nil {
			http.Error(w, "Unavailable", 503)
			return
		}
		w.WriteHeader(204)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	s, e := a.DB.Read(r.Context())
	if e != nil || requireRole(s, a.user(r), "owner") != nil {
		http.Error(w, "Forbidden", 403)
		return
	}
	samples, e := rowsJSON(r.Context(), a.DB.Pool, `SELECT row_to_json(t) FROM (SELECT id,name,path,value,device,extract(epoch FROM at)*1000 AS "at" FROM performance_samples WHERE at>=now()-interval '7 days' ORDER BY at DESC LIMIT 10000) t`)
	if e != nil {
		http.Error(w, "Unavailable", 503)
		return
	}
	buckets := map[string][]Object{}
	for _, v := range samples {
		key := str(v["name"]) + "|" + str(v["path"]) + "|" + str(v["device"])
		buckets[key] = append(buckets[key], v)
	}
	limits := map[string][2]float64{"LCP": {2500, 4000}, "INP": {200, 500}, "CLS": {.1, .25}, "FCP": {1800, 3000}, "TTFB": {800, 1800}, "NAV": {1000, 2500}, "API": {500, 1500}}
	groups := []Object{}
	for _, rows := range buckets {
		sort.Slice(rows, func(i, j int) bool { return num(rows[i]["value"]) < num(rows[j]["value"]) })
		p75 := num(rows[int(math.Ceil(float64(len(rows))*.75))-1]["value"])
		v := rows[0]
		l := limits[str(v["name"])]
		rating := "poor"
		if p75 <= l[0] {
			rating = "good"
		} else if p75 <= l[1] {
			rating = "needs-improvement"
		}
		groups = append(groups, Object{"name": v["name"], "path": v["path"], "device": v["device"], "count": len(rows), "p75": p75, "rating": rating})
	}
	sort.Slice(groups, func(i, j int) bool { return num(groups[i]["count"]) > num(groups[j]["count"]) })
	recent := samples
	if len(recent) > 30 {
		recent = recent[:30]
	}
	jsonReply(w, Object{"days": 7, "total": len(samples), "groups": groups, "recent": recent, "retentionDays": 30, "limit": 20000}, 200)
}

func metricPath(path string) bool {
	return contains([]string{"/", "/san-pham", "/bo-suu-tap", "/cau-chuyen", "/bai-viet", "/lien-he", "/gio-hang", "/thanh-toan", "/tra-cuu", "/dat-hang-thanh-cong", "/chinh-sach/giao-hang", "/chinh-sach/doi-tra", "/chinh-sach/bao-mat", "/san-pham/:id", "/bai-viet/:id"}, path)
}
