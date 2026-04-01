package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gorilla/mux"
)

func registerDynamicDocs(router *mux.Router) {
	router.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(buildMuxOpenAPISpec(router))
	}).Methods("GET")

	router.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(swaggerUIHTML("/openapi.json")))
	}).Methods("GET")
}

func buildMuxOpenAPISpec(router *mux.Router) map[string]interface{} {
	type routeOp struct {
		path   string
		method string
	}

	ops := make([]routeOp, 0, 128)
	_ = router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err != nil || path == "" {
			return nil
		}

		methods, err := route.GetMethods()
		if err != nil || len(methods) == 0 {
			methods = []string{"GET"}
		}

		for _, method := range methods {
			ops = append(ops, routeOp{path: path, method: method})
		}
		return nil
	})

	sort.Slice(ops, func(i, j int) bool {
		if ops[i].path == ops[j].path {
			return ops[i].method < ops[j].method
		}
		return ops[i].path < ops[j].path
	})

	paths := map[string]map[string]interface{}{}
	for _, op := range ops {
		if _, ok := paths[op.path]; !ok {
			paths[op.path] = map[string]interface{}{}
		}

		paths[op.path][strings.ToLower(op.method)] = map[string]interface{}{
			"summary":     fmt.Sprintf("%s %s", op.method, op.path),
			"operationId": sanitizeOperationID(op.method + "_" + op.path),
			"responses": map[string]interface{}{
				"200": map[string]string{"description": "Success"},
			},
		}
	}

	return map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]string{
			"title":   "WarehouseCore API",
			"version": "dynamic",
		},
		"paths": paths,
	}
}

func sanitizeOperationID(s string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"{", "",
		"}", "",
		"-", "_",
		".", "_",
	)
	return strings.Trim(replacer.Replace(strings.ToLower(s)), "_")
}

func swaggerUIHTML(specURL string) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>WarehouseCore API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: '%s',
      dom_id: '#swagger-ui',
      deepLinking: true
    });
  </script>
</body>
</html>`, specURL)
}
