package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"warehousecore/internal/services"
)

type twentyGraphQLResponse struct {
	Data   map[string]json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type twentyOpportunity struct {
	ID                 string                   `json:"id"`
	Name               string                   `json:"name"`
	JobCode            string                   `json:"jobCode"`
	WarehouseCoreJobID *float64                 `json:"warehouseCoreJobId"`
	JobStartDate       *string                  `json:"jobStartDate"`
	JobEndDate         *string                  `json:"jobEndDate"`
	CloseDate          *string                  `json:"closeDate"`
	Stage              interface{}              `json:"stage"`
	Company            *twentyCompany           `json:"company"`
	JobRequirements    twentyJobRequirementList `json:"jobProductRequirements"`
}

type twentyCompany struct {
	Name string `json:"name"`
}

type twentyJobRequirementLine struct {
	Name                   string   `json:"name"`
	Quantity               *float64 `json:"quantity"`
	WarehouseCoreProductID *float64 `json:"warehouseCoreProductId"`
	WarehouseProduct       *struct {
		WarehouseID *float64 `json:"warehouseId"`
		ProductName string   `json:"productName"`
	} `json:"warehouseProduct"`
}

// twentyJobRequirementList supports both legacy array responses and
// connection-style payloads returned by newer Twenty schemas.
type twentyJobRequirementList []twentyJobRequirementLine

func (l *twentyJobRequirementList) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*l = nil
		return nil
	}

	if strings.HasPrefix(trimmed, "[") {
		var arr []twentyJobRequirementLine
		if err := json.Unmarshal(data, &arr); err != nil {
			return err
		}
		*l = arr
		return nil
	}

	type edge struct {
		Node twentyJobRequirementLine `json:"node"`
	}
	type connection struct {
		Edges []edge                     `json:"edges"`
		Nodes []twentyJobRequirementLine `json:"nodes"`
	}

	var conn connection
	if err := json.Unmarshal(data, &conn); err != nil {
		return err
	}

	if len(conn.Nodes) > 0 {
		*l = conn.Nodes
		return nil
	}

	rows := make([]twentyJobRequirementLine, 0, len(conn.Edges))
	for _, e := range conn.Edges {
		rows = append(rows, e.Node)
	}
	*l = rows
	return nil
}

func requirementProductID(req twentyJobRequirementLine) int {
	if req.WarehouseCoreProductID != nil {
		id := int(*req.WarehouseCoreProductID)
		if id > 0 {
			return id
		}
	}
	if req.WarehouseProduct != nil && req.WarehouseProduct.WarehouseID != nil {
		id := int(*req.WarehouseProduct.WarehouseID)
		if id > 0 {
			return id
		}
	}
	return 0
}

func requirementName(req twentyJobRequirementLine) string {
	if name := strings.TrimSpace(req.Name); name != "" {
		return name
	}
	if req.WarehouseProduct != nil {
		if name := strings.TrimSpace(req.WarehouseProduct.ProductName); name != "" {
			return name
		}
	}
	return ""
}

func requirementQty(req twentyJobRequirementLine) int {
	if req.Quantity != nil {
		qty := int(*req.Quantity)
		if qty > 0 {
			return qty
		}
	}
	if requirementProductID(req) > 0 {
		// If a product is linked but quantity is omitted, treat it as 1 by default.
		return 1
	}
	return 0
}

func twentyBaseURL() string {
	base := ""
	for _, key := range []string{"TWENTY_BASE_URL", "TWENTY_URL", "TWENTY_SERVER_URL", "TWENTY_GRAPHQL_URL"} {
		if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
			base = strings.TrimRight(raw, "/")
			break
		}
	}
	if base == "" {
		if cfg, err := services.GetTwentyConfig(); err == nil {
			base = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
		}
	}
	if strings.HasSuffix(strings.ToLower(base), "/graphql") {
		base = strings.TrimSuffix(base, "/graphql")
	}
	if base == "" {
		return ""
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}

	// Inside containers, localhost resolves to the container itself, not the host.
	// Rewrite localhost-style Twenty URLs to host.docker.internal for local dev.
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if isRunningInDocker() && (host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "0.0.0.0") {
		port := parsed.Port()
		if port == "" {
			parsed.Host = "host.docker.internal"
		} else {
			parsed.Host = "host.docker.internal:" + port
		}
		return strings.TrimRight(parsed.String(), "/")
	}

	return base
}

func isRunningInDocker() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

func twentyToken() (string, string) {
	if key := strings.TrimSpace(os.Getenv("TWENTY_API_KEY")); key != "" {
		return key, "TWENTY_API_KEY"
	}
	if key := strings.TrimSpace(os.Getenv("TWENTY_ACCESS_TOKEN")); key != "" {
		return key, "TWENTY_ACCESS_TOKEN"
	}
	if key := strings.TrimSpace(os.Getenv("TWENTY_TOKEN")); key != "" {
		return key, "TWENTY_TOKEN"
	}
	if cfg, err := services.GetTwentyConfig(); err == nil {
		if key := strings.TrimSpace(cfg.APIKey); key != "" {
			return key, "twenty.config.api_key"
		}
	}
	return "", ""
}

func twentyGraphQLEndpoint() string {
	base := twentyBaseURL()
	if base == "" {
		return ""
	}
	if strings.HasSuffix(base, "/graphql") {
		return base
	}
	return base + "/graphql"
}

func twentyConfigured() bool {
	return twentyGraphQLEndpoint() != ""
}

func doTwentyGraphQL(ctx context.Context, query string, variables map[string]interface{}, out interface{}) error {
	endpoint := twentyGraphQLEndpoint()
	if endpoint == "" {
		return fmt.Errorf("TWENTY_BASE_URL not configured")
	}

	payload := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		payload["variables"] = variables
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	token, _ := twentyToken()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-API-Key", token)
	}

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("twenty graphql request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var gql twentyGraphQLResponse
	if err := json.Unmarshal(respBody, &gql); err != nil {
		return err
	}

	if len(gql.Errors) > 0 {
		return fmt.Errorf("twenty graphql error: %s", gql.Errors[0].Message)
	}

	if out == nil {
		return nil
	}

	rootField := strings.TrimSpace(os.Getenv("TWENTY_OPPORTUNITY_QUERY_ROOT"))
	if rootField == "" {
		rootField = "findManyOpportunities"
	}

	raw, ok := gql.Data[rootField]
	if !ok {
		for _, v := range gql.Data {
			raw = v
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("twenty graphql response missing root field")
	}

	return json.Unmarshal(raw, out)
}

// doTwentyGraphQLRoot is like doTwentyGraphQL but uses an explicit rootField
// instead of the TWENTY_OPPORTUNITY_QUERY_ROOT env var. Use this for mutations
// and queries where the root key is known at the call site.
func doTwentyGraphQLRoot(ctx context.Context, query string, variables map[string]interface{}, rootField string, out interface{}) error {
	endpoint := twentyGraphQLEndpoint()
	if endpoint == "" {
		return fmt.Errorf("TWENTY_BASE_URL not configured")
	}

	payload := map[string]interface{}{"query": query}
	if variables != nil {
		payload["variables"] = variables
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token, _ := twentyToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-API-Key", token)
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("twenty request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var gql twentyGraphQLResponse
	if err := json.Unmarshal(respBody, &gql); err != nil {
		return err
	}
	if len(gql.Errors) > 0 {
		return fmt.Errorf("twenty graphql error: %s", gql.Errors[0].Message)
	}
	if out == nil {
		return nil
	}
	raw, ok := gql.Data[rootField]
	if !ok {
		return fmt.Errorf("twenty response missing field %q", rootField)
	}
	return json.Unmarshal(raw, out)
}

func parseTwentyStatus(value interface{}) string {
	if value == nil {
		return "open"
	}
	if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s)
	}
	if m, ok := value.(map[string]interface{}); ok {
		for _, key := range []string{"name", "value", "label"} {
			if v, exists := m[key]; exists {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		}
	}
	return "open"
}

func matchesJobStatusFilter(statusFilter, actual string) bool {
	if strings.TrimSpace(statusFilter) == "" {
		return true
	}
	if strings.EqualFold(statusFilter, "open") {
		lower := strings.ToLower(strings.TrimSpace(actual))
		return lower != "completed" && lower != "invoiced" && lower != "cancelled"
	}
	return strings.EqualFold(statusFilter, actual)
}

func parseDateValue(primary *string, fallback *string) *string {
	if primary != nil && strings.TrimSpace(*primary) != "" {
		v := strings.TrimSpace(*primary)
		return &v
	}
	if fallback != nil && strings.TrimSpace(*fallback) != "" {
		v := strings.TrimSpace(*fallback)
		return &v
	}
	return nil
}

func loadTwentyOpportunities(ctx context.Context) ([]twentyOpportunity, error) {
	query := strings.TrimSpace(os.Getenv("TWENTY_OPPORTUNITIES_QUERY"))
	if query != "" {
		var custom []twentyOpportunity
		if err := doTwentyGraphQL(ctx, query, nil, &custom); err != nil {
			return nil, err
		}

		// Auto-assign warehouseCoreJobId to any new Opportunities that don't have one.
		go bootstrapTwentyJobIDs(context.Background())

		return custom, nil
	}

	const findManyQWithRelation = `
	query WarehousecoreJobs {
	  findManyOpportunities {
	    id
	    name
	    jobCode
	    warehouseCoreJobId
	    jobStartDate
	    jobEndDate
	    closeDate
	    stage
	    company {
	      name
	    }
	    jobProductRequirements {
	      name
	      quantity
	      warehouseCoreProductId
	      warehouseProduct {
	        warehouseId
	        productName
	      }
	    }
	  }
	}
	`
	const findManyQLegacy = `
	query WarehousecoreJobs {
	  findManyOpportunities {
	    id
	    name
	    jobCode
	    warehouseCoreJobId
	    jobStartDate
	    jobEndDate
	    closeDate
	    stage
	    company {
	      name
	    }
	    jobProductRequirements {
	      name
	      quantity
	      warehouseCoreProductId
	    }
	  }
	}
	`

	var opportunities []twentyOpportunity
	if err := doTwentyGraphQLRoot(ctx, findManyQWithRelation, nil, "findManyOpportunities", &opportunities); err != nil {
		if errLegacy := doTwentyGraphQLRoot(ctx, findManyQLegacy, nil, "findManyOpportunities", &opportunities); errLegacy != nil {
			const connectionQWithRelation = `
		query WarehousecoreJobs {
		  opportunities {
		    edges {
		      node {
		        id
		        name
		        jobCode
		        warehouseCoreJobId
		        jobStartDate
		        jobEndDate
		        closeDate
		        stage
		        company {
		          name
		        }
		        jobProductRequirements {
		          name
		          quantity
		          warehouseCoreProductId
		          warehouseProduct {
		            warehouseId
		            productName
		          }
		        }
		      }
		    }
		  }
		}
		`
			const connectionQLegacy = `
			query WarehousecoreJobs {
			  opportunities {
			    edges {
			      node {
			        id
			        name
			        jobCode
			        warehouseCoreJobId
			        jobStartDate
			        jobEndDate
			        closeDate
			        stage
			        company {
			          name
			        }
			        jobProductRequirements {
			          name
			          quantity
			          warehouseCoreProductId
			        }
			      }
			    }
			  }
			}
			`
			type oppEdge struct {
				Node twentyOpportunity `json:"node"`
			}
			type oppConnection struct {
				Edges []oppEdge `json:"edges"`
			}
			var conn oppConnection
			if err2 := doTwentyGraphQLRoot(ctx, connectionQWithRelation, nil, "opportunities", &conn); err2 != nil {
				if err2Legacy := doTwentyGraphQLRoot(ctx, connectionQLegacy, nil, "opportunities", &conn); err2Legacy != nil {
					return nil, err2Legacy
				}
			}
			opportunities = make([]twentyOpportunity, 0, len(conn.Edges))
			for _, e := range conn.Edges {
				opportunities = append(opportunities, e.Node)
			}
		}
	}

	// Auto-assign warehouseCoreJobId to any new Opportunities that don't have one.
	go bootstrapTwentyJobIDs(context.Background())

	return opportunities, nil
}

func twentyJobsResponse(statusFilter string, opportunities []twentyOpportunity) []map[string]interface{} {
	jobs := make([]map[string]interface{}, 0)

	for _, opp := range opportunities {
		if opp.WarehouseCoreJobID == nil {
			continue
		}
		jobID := int(*opp.WarehouseCoreJobID)
		if jobID <= 0 {
			continue
		}

		status := parseTwentyStatus(opp.Stage)
		if !matchesJobStatusFilter(statusFilter, status) {
			continue
		}

		requirementsCount := 0
		hasLinkedRequirement := false
		for _, req := range opp.JobRequirements {
			if requirementProductID(req) > 0 {
				hasLinkedRequirement = true
				requirementsCount += requirementQty(req)
			}
		}

		// Only expose opportunities that are actually linked to at least one
		// WarehouseCore product requirement.
		if !hasLinkedRequirement {
			continue
		}

		jobCode := strings.TrimSpace(opp.JobCode)
		if jobCode == "" {
			jobCode = fmt.Sprintf("JOB%06d", jobID)
		}

		row := map[string]interface{}{
			"job_id":             jobID,
			"job_code":           jobCode,
			"description":        strings.TrimSpace(opp.Name),
			"status":             status,
			"device_count":       0,
			"requirements_count": requirementsCount,
		}

		if d := parseDateValue(opp.JobStartDate, nil); d != nil {
			row["start_date"] = *d
		}
		if d := parseDateValue(opp.JobEndDate, opp.CloseDate); d != nil {
			row["end_date"] = *d
		}

		if opp.Company != nil {
			companyName := strings.TrimSpace(opp.Company.Name)
			if companyName != "" {
				row["customer_first_name"] = companyName
				row["customer_last_name"] = ""
			}
		}

		jobs = append(jobs, row)
	}

	return jobs
}

func twentyJobSummaryResponse(jobID int, opportunities []twentyOpportunity) (map[string]interface{}, bool) {
	for _, opp := range opportunities {
		if opp.WarehouseCoreJobID == nil || int(*opp.WarehouseCoreJobID) != jobID {
			continue
		}

		jobCode := strings.TrimSpace(opp.JobCode)
		if jobCode == "" {
			jobCode = fmt.Sprintf("JOB%06d", jobID)
		}

		summary := map[string]interface{}{
			"job_id":               jobID,
			"job_code":             jobCode,
			"description":          strings.TrimSpace(opp.Name),
			"status":               parseTwentyStatus(opp.Stage),
			"devices":              []map[string]interface{}{},
			"product_requirements": []map[string]interface{}{},
		}

		if d := parseDateValue(opp.JobStartDate, nil); d != nil {
			summary["start_date"] = *d
		}
		if d := parseDateValue(opp.JobEndDate, opp.CloseDate); d != nil {
			summary["end_date"] = *d
		}
		if opp.Company != nil {
			companyName := strings.TrimSpace(opp.Company.Name)
			if companyName != "" {
				summary["customer_first_name"] = companyName
				summary["customer_last_name"] = ""
			}
		}

		productReqs := make([]map[string]interface{}, 0, len(opp.JobRequirements))
		for _, req := range opp.JobRequirements {
			requiredQty := requirementQty(req)
			productID := requirementProductID(req)
			name := requirementName(req)

			productReqs = append(productReqs, map[string]interface{}{
				"product_id":   productID,
				"product_name": name,
				"required":     requiredQty,
				"assigned":     0,
			})
		}
		summary["product_requirements"] = productReqs

		return summary, true
	}
	return nil, false
}

func proxyTwentyJobs(w http.ResponseWriter, r *http.Request) bool {
	if !twentyConfigured() {
		return false
	}
	opportunities, err := loadTwentyOpportunities(r.Context())
	if err != nil {
		log.Printf("Twenty jobs adapter unavailable, falling back: %v", err)
		return false
	}

	jobs := twentyJobsResponse(strings.TrimSpace(r.URL.Query().Get("status")), opportunities)
	respondJSON(w, http.StatusOK, jobs)
	return true
}

func proxyTwentyJobSummaryWithLocalDevices(w http.ResponseWriter, r *http.Request, jobID int) bool {
	if !twentyConfigured() {
		return false
	}
	opportunities, err := loadTwentyOpportunities(r.Context())
	if err != nil {
		log.Printf("Twenty job summary adapter unavailable, falling back: %v", err)
		return false
	}

	summary, found := twentyJobSummaryResponse(jobID, opportunities)
	if !found {
		return false
	}

	body, err := json.Marshal(summary)
	if err != nil {
		log.Printf("Failed to marshal Twenty summary for job %d: %v", jobID, err)
		return false
	}

	merged, err := mergeLocalJobDevicesIntoSummary(body, jobID)
	if err != nil {
		log.Printf("Failed to merge local job devices into Twenty summary job %d: %v", jobID, err)
		return false
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(merged)
	return true
}
