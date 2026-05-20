package handlers

// twenty_sync.go — two-way integration helpers between WarehouseCore and Twenty CRM.
//
//  • syncProductsToTwenty: reads every local product and upserts it to Twenty.
//  • TwentySyncProductsHandler: POST /api/v1/twenty/sync-products.
//  • bootstrapTwentyJobIDs: auto-creates local jobs for new Opportunities.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"warehousecore/internal/repository"
	"warehousecore/internal/services"
)

type whProductRow struct {
	ProductID     int
	Name          string
	CategoryName  string
	DailyRate     float64
	StockQuantity float64
}

type whPackageRow struct {
	PackageID       int
	PackageName     string
	SourceProductID int
}

type whpNode struct {
	ID          string  `json:"id"`
	WarehouseID float64 `json:"warehouseId"`
}

type whpEdge struct {
	Node whpNode `json:"node"`
}

type whpListEdges struct {
	Edges []whpEdge `json:"edges"`
}

type whpkgNode struct {
	ID                 string  `json:"id"`
	WarehousePackageID float64 `json:"warehousePackageId"`
}

type whpkgEdge struct {
	Node whpkgNode `json:"node"`
}

type whpkgListEdges struct {
	Edges []whpkgEdge `json:"edges"`
}

type twentySyncCounters struct {
	ProductCreated int `json:"productCreated"`
	ProductUpdated int `json:"productUpdated"`
	PackageCreated int `json:"packageCreated"`
	PackageUpdated int `json:"packageUpdated"`
}

func (c twentySyncCounters) TotalCreated() int {
	return c.ProductCreated + c.PackageCreated
}

func (c twentySyncCounters) TotalUpdated() int {
	return c.ProductUpdated + c.PackageUpdated
}

// RegisterTwentySyncHook wires the scheduler hook to the product sync logic.
// Call once during startup after DB init.
func RegisterTwentySyncHook() {
	services.TwentyProductSyncHook = func() (int, int, error) {
		counts, err := syncProductsToTwenty(context.Background())
		if err != nil {
			return 0, 0, err
		}
		return counts.TotalCreated(), counts.TotalUpdated(), nil
	}
}

// syncProductsToTwenty reads all products from the local database and
// creates or updates corresponding WarehouseCoreProduct records in Twenty.
// Existing records are matched by the warehouseId field.
func syncProductsToTwenty(ctx context.Context) (twentySyncCounters, error) {
	if !twentyConfigured() {
		return twentySyncCounters{}, fmt.Errorf("Twenty not configured")
	}

	counts := twentySyncCounters{}

	db := repository.GetSQLDB()

	// Check whether product_locations table exists; fall back to 0 for accessories/consumables if not.
	var plExists bool
	_ = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = current_schema() AND table_name = 'product_locations'
		)`).Scan(&plExists)

	stockExpr := `COALESCE((SELECT COUNT(*) FROM devices d WHERE d.productid = p.productid), 0)`
	if plExists {
		stockExpr = `CASE
				WHEN COALESCE(p.is_consumable, false) = true OR COALESCE(p.is_accessory, false) = true THEN
					COALESCE((SELECT SUM(pl.quantity) FROM product_locations pl WHERE pl.product_id = p.productid), 0)
				ELSE
					COALESCE((SELECT COUNT(*) FROM devices d WHERE d.productid = p.productid), 0)
			END`
	}

	rows, qErr := db.QueryContext(ctx, `
		SELECT
			p.productid,
			p.name,
			COALESCE(c.name, '') AS category_name,
			COALESCE(p.itemcostperday, 0),
			`+stockExpr+` AS stock_quantity
		FROM products p
		LEFT JOIN categories c ON c.categoryid = p.categoryid
		ORDER BY p.productid
	`)
	if qErr != nil {
		return counts, fmt.Errorf("query products: %w", qErr)
	}
	defer rows.Close()

	var products []whProductRow
	for rows.Next() {
		var pr whProductRow
		if scanErr := rows.Scan(&pr.ProductID, &pr.Name, &pr.CategoryName, &pr.DailyRate, &pr.StockQuantity); scanErr != nil {
			return counts, fmt.Errorf("scan product row: %w", scanErr)
		}
		products = append(products, pr)
	}
	if rowErr := rows.Err(); rowErr != nil {
		return counts, fmt.Errorf("iterate products: %w", rowErr)
	}
	if len(products) == 0 {
		return counts, nil
	}

	nodes, fErr := loadExistingWarehouseCoreProducts(ctx)
	if fErr != nil {
		return counts, fmt.Errorf("fetch existing warehouseCoreProducts: %w", fErr)
	}

	byWarehouseID := make(map[int]string, len(nodes))
	for _, e := range nodes {
		byWarehouseID[int(e.Node.WarehouseID)] = e.Node.ID
	}

	syncedAt := time.Now().UTC().Format(time.RFC3339)
	for _, p := range products {
		twentyID, exists := byWarehouseID[p.ProductID]
		if exists {
			const updateOneQ = `mutation UpdateWHProduct($id: ID!, $productName: String!, $categoryName: String!, $dailyRate: Float!, $stockQuantity: Float!, $lastSyncAt: DateTime!) {
				updateOneWarehouseCoreProduct(id: $id, data: {
					productName: $productName
					categoryName: $categoryName
					dailyRate: $dailyRate
					stockQuantity: $stockQuantity
					lastSyncAt: $lastSyncAt
				}) { id }
			}`
			const updateQ = `mutation UpdateWHProduct($id: ID!, $productName: String!, $categoryName: String!, $dailyRate: Float!, $stockQuantity: Float!, $lastSyncAt: DateTime!) {
				updateWarehouseCoreProduct(id: $id, data: {
					productName: $productName
					categoryName: $categoryName
					dailyRate: $dailyRate
					stockQuantity: $stockQuantity
					lastSyncAt: $lastSyncAt
				}) { id }
			}`

			vars := map[string]interface{}{
				"id":            twentyID,
				"productName":   p.Name,
				"categoryName":  p.CategoryName,
				"dailyRate":     p.DailyRate,
				"stockQuantity": p.StockQuantity,
				"lastSyncAt":    syncedAt,
			}

			uErr := doTwentyGraphQLRoot(ctx, updateOneQ, vars, "updateOneWarehouseCoreProduct", nil)
			if uErr != nil {
				uErr = doTwentyGraphQLRoot(ctx, updateQ, vars, "updateWarehouseCoreProduct", nil)
			}
			if uErr != nil {
				log.Printf("[TWENTY SYNC] update product %d: %v", p.ProductID, uErr)
				continue
			}
			counts.ProductUpdated++
		} else {
			const createOneQ = `mutation CreateWHProduct($warehouseId: Float!, $productName: String!, $categoryName: String!, $dailyRate: Float!, $stockQuantity: Float!, $lastSyncAt: DateTime!) {
				createOneWarehouseCoreProduct(data: {
					warehouseId: $warehouseId
					productName: $productName
					categoryName: $categoryName
					dailyRate: $dailyRate
					stockQuantity: $stockQuantity
					lastSyncAt: $lastSyncAt
				}) { id warehouseId }
			}`
			const createQ = `mutation CreateWHProduct($warehouseId: Float!, $productName: String!, $categoryName: String!, $dailyRate: Float!, $stockQuantity: Float!, $lastSyncAt: DateTime!) {
				createWarehouseCoreProduct(data: {
					warehouseId: $warehouseId
					productName: $productName
					categoryName: $categoryName
					dailyRate: $dailyRate
					stockQuantity: $stockQuantity
					lastSyncAt: $lastSyncAt
				}) { id warehouseId }
			}`

			vars := map[string]interface{}{
				"warehouseId":   float64(p.ProductID),
				"productName":   p.Name,
				"categoryName":  p.CategoryName,
				"dailyRate":     p.DailyRate,
				"stockQuantity": p.StockQuantity,
				"lastSyncAt":    syncedAt,
			}

			cErr := doTwentyGraphQLRoot(ctx, createOneQ, vars, "createOneWarehouseCoreProduct", nil)
			if cErr != nil {
				cErr = doTwentyGraphQLRoot(ctx, createQ, vars, "createWarehouseCoreProduct", nil)
			}
			if cErr != nil {
				log.Printf("[TWENTY SYNC] create product %d: %v", p.ProductID, cErr)
				continue
			}
			counts.ProductCreated++
		}
	}

	pkgCreated, pkgUpdated, pkgErr := syncProductPackagesToTwenty(ctx, byWarehouseID, syncedAt)
	if pkgErr != nil {
		return counts, pkgErr
	}
	counts.PackageCreated = pkgCreated
	counts.PackageUpdated = pkgUpdated

	return counts, nil
}

func syncProductPackagesToTwenty(ctx context.Context, productByWarehouseID map[int]string, syncedAt string) (created, updated int, err error) {
	db := repository.GetSQLDB()
	rows, qErr := db.QueryContext(ctx, `
		SELECT
			COALESCE(pp.package_id, pp.id) AS package_id,
			pp.name,
			COALESCE(pp.product_id, 0) AS source_product_id
		FROM product_packages pp
		ORDER BY COALESCE(pp.package_id, pp.id)
	`)
	if qErr != nil {
		return 0, 0, fmt.Errorf("query product_packages: %w", qErr)
	}
	defer rows.Close()

	var packages []whPackageRow
	for rows.Next() {
		var pr whPackageRow
		if scanErr := rows.Scan(&pr.PackageID, &pr.PackageName, &pr.SourceProductID); scanErr != nil {
			return 0, 0, fmt.Errorf("scan product package row: %w", scanErr)
		}
		packages = append(packages, pr)
	}
	if rowErr := rows.Err(); rowErr != nil {
		return 0, 0, fmt.Errorf("iterate product packages: %w", rowErr)
	}
	if len(packages) == 0 {
		return 0, 0, nil
	}

	existing, loadErr := loadExistingWarehouseCorePackages(ctx)
	if loadErr != nil {
		return 0, 0, fmt.Errorf("fetch existing warehouseCorePackages: %w", loadErr)
	}

	byWarehousePackageID := make(map[int]string, len(existing))
	for _, e := range existing {
		byWarehousePackageID[int(e.Node.WarehousePackageID)] = e.Node.ID
	}

	for _, pkg := range packages {
		productTwentyID := productByWarehouseID[pkg.SourceProductID]
		if productTwentyID == "" {
			log.Printf("[TWENTY SYNC] skip package %d: source product %d was not synced", pkg.PackageID, pkg.SourceProductID)
			continue
		}

		twentyID := byWarehousePackageID[pkg.PackageID]
		if twentyID == "" {
			const createOneWithRelQ = `mutation CreateWHPackage($warehousePackageId: Float!, $packageName: String!, $sourceProductId: Float!, $lastSyncAt: DateTime!, $warehouseProductId: ID!) {
				createOneWarehouseCorePackage(data: {
					warehousePackageId: $warehousePackageId
					packageName: $packageName
					sourceProductId: $sourceProductId
					lastSyncAt: $lastSyncAt
					warehouseProduct: { connect: { id: $warehouseProductId } }
				}) { id }
			}`
			const createWithRelQ = `mutation CreateWHPackage($warehousePackageId: Float!, $packageName: String!, $sourceProductId: Float!, $lastSyncAt: DateTime!, $warehouseProductId: ID!) {
				createWarehouseCorePackage(data: {
					warehousePackageId: $warehousePackageId
					packageName: $packageName
					sourceProductId: $sourceProductId
					lastSyncAt: $lastSyncAt
					warehouseProduct: { connect: { id: $warehouseProductId } }
				}) { id }
			}`
			const createOneWithFKQ = `mutation CreateWHPackage($warehousePackageId: Float!, $packageName: String!, $sourceProductId: Float!, $lastSyncAt: DateTime!, $warehouseProductId: ID!) {
				createOneWarehouseCorePackage(data: {
					warehousePackageId: $warehousePackageId
					packageName: $packageName
					sourceProductId: $sourceProductId
					lastSyncAt: $lastSyncAt
					warehouseProductId: $warehouseProductId
				}) { id }
			}`
			const createWithFKQ = `mutation CreateWHPackage($warehousePackageId: Float!, $packageName: String!, $sourceProductId: Float!, $lastSyncAt: DateTime!, $warehouseProductId: ID!) {
				createWarehouseCorePackage(data: {
					warehousePackageId: $warehousePackageId
					packageName: $packageName
					sourceProductId: $sourceProductId
					lastSyncAt: $lastSyncAt
					warehouseProductId: $warehouseProductId
				}) { id }
			}`

			vars := map[string]interface{}{
				"warehousePackageId": float64(pkg.PackageID),
				"packageName":        pkg.PackageName,
				"sourceProductId":    float64(pkg.SourceProductID),
				"lastSyncAt":         syncedAt,
				"warehouseProductId": productTwentyID,
			}

			cErr := doTwentyGraphQLRoot(ctx, createOneWithRelQ, vars, "createOneWarehouseCorePackage", nil)
			if cErr != nil {
				cErr = doTwentyGraphQLRoot(ctx, createWithRelQ, vars, "createWarehouseCorePackage", nil)
			}
			if cErr != nil {
				cErr = doTwentyGraphQLRoot(ctx, createOneWithFKQ, vars, "createOneWarehouseCorePackage", nil)
			}
			if cErr != nil {
				cErr = doTwentyGraphQLRoot(ctx, createWithFKQ, vars, "createWarehouseCorePackage", nil)
			}
			if cErr != nil {
				log.Printf("[TWENTY SYNC] create package %d: %v", pkg.PackageID, cErr)
				continue
			}
			created++
			continue
		}

		const updateOneWithRelQ = `mutation UpdateWHPackage($id: ID!, $packageName: String!, $sourceProductId: Float!, $lastSyncAt: DateTime!, $warehouseProductId: ID!) {
			updateOneWarehouseCorePackage(id: $id, data: {
				packageName: $packageName
				sourceProductId: $sourceProductId
				lastSyncAt: $lastSyncAt
				warehouseProduct: { connect: { id: $warehouseProductId } }
			}) { id }
		}`
		const updateWithRelQ = `mutation UpdateWHPackage($id: ID!, $packageName: String!, $sourceProductId: Float!, $lastSyncAt: DateTime!, $warehouseProductId: ID!) {
			updateWarehouseCorePackage(id: $id, data: {
				packageName: $packageName
				sourceProductId: $sourceProductId
				lastSyncAt: $lastSyncAt
				warehouseProduct: { connect: { id: $warehouseProductId } }
			}) { id }
		}`
		const updateOneWithFKQ = `mutation UpdateWHPackage($id: ID!, $packageName: String!, $sourceProductId: Float!, $lastSyncAt: DateTime!, $warehouseProductId: ID!) {
			updateOneWarehouseCorePackage(id: $id, data: {
				packageName: $packageName
				sourceProductId: $sourceProductId
				lastSyncAt: $lastSyncAt
				warehouseProductId: $warehouseProductId
			}) { id }
		}`
		const updateWithFKQ = `mutation UpdateWHPackage($id: ID!, $packageName: String!, $sourceProductId: Float!, $lastSyncAt: DateTime!, $warehouseProductId: ID!) {
			updateWarehouseCorePackage(id: $id, data: {
				packageName: $packageName
				sourceProductId: $sourceProductId
				lastSyncAt: $lastSyncAt
				warehouseProductId: $warehouseProductId
			}) { id }
		}`

		vars := map[string]interface{}{
			"id":                 twentyID,
			"packageName":        pkg.PackageName,
			"sourceProductId":    float64(pkg.SourceProductID),
			"lastSyncAt":         syncedAt,
			"warehouseProductId": productTwentyID,
		}

		uErr := doTwentyGraphQLRoot(ctx, updateOneWithRelQ, vars, "updateOneWarehouseCorePackage", nil)
		if uErr != nil {
			uErr = doTwentyGraphQLRoot(ctx, updateWithRelQ, vars, "updateWarehouseCorePackage", nil)
		}
		if uErr != nil {
			uErr = doTwentyGraphQLRoot(ctx, updateOneWithFKQ, vars, "updateOneWarehouseCorePackage", nil)
		}
		if uErr != nil {
			uErr = doTwentyGraphQLRoot(ctx, updateWithFKQ, vars, "updateWarehouseCorePackage", nil)
		}
		if uErr != nil {
			log.Printf("[TWENTY SYNC] update package %d: %v", pkg.PackageID, uErr)
			continue
		}
		updated++
	}

	return created, updated, nil
}

// TwentySyncProductsHandler triggers a full product sync and returns a summary.
func TwentySyncProductsHandler(w http.ResponseWriter, r *http.Request) {
	counts, err := syncProductsToTwenty(r.Context())
	if err != nil {
		log.Printf("[TWENTY SYNC] product sync failed: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"ok":    false,
			"error": fmt.Sprintf("sync failed: %v", err),
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":             true,
		"created":        counts.TotalCreated(),
		"updated":        counts.TotalUpdated(),
		"productCreated": counts.ProductCreated,
		"productUpdated": counts.ProductUpdated,
		"packageCreated": counts.PackageCreated,
		"packageUpdated": counts.PackageUpdated,
	})
}

func loadExistingWarehouseCoreProducts(ctx context.Context) ([]whpEdge, error) {
	// Variant A: direct list shape on older schemas.
	const directFindManyQ = `query {
		findManyWarehouseCoreProducts {
			id
			warehouseId
		}
	}`
	var direct []whpNode
	if err := doTwentyGraphQLRoot(ctx, directFindManyQ, nil, "findManyWarehouseCoreProducts", &direct); err == nil {
		edges := make([]whpEdge, 0, len(direct))
		for _, n := range direct {
			edges = append(edges, whpEdge{Node: n})
		}
		return edges, nil
	}

	// Variant B: connection shape on newer schemas.
	const connectionQ = `query {
		warehouseCoreProducts {
			edges { node { id warehouseId } }
		}
	}`
	var connection whpListEdges
	if err := doTwentyGraphQLRoot(ctx, connectionQ, nil, "warehouseCoreProducts", &connection); err == nil {
		return connection.Edges, nil
	}

	// Variant C: edges/node shape on older schemas.
	const edgesFindManyQ = `query {
		findManyWarehouseCoreProducts {
			edges { node { id warehouseId } }
		}
	}`
	var edgeList whpListEdges
	if err := doTwentyGraphQLRoot(ctx, edgesFindManyQ, nil, "findManyWarehouseCoreProducts", &edgeList); err == nil {
		return edgeList.Edges, nil
	}

	// Prefer a clear hint for the most common failure mode.
	errFindMany := doTwentyGraphQLRoot(ctx, directFindManyQ, nil, "findManyWarehouseCoreProducts", &direct)
	errConnection := doTwentyGraphQLRoot(ctx, connectionQ, nil, "warehouseCoreProducts", &connection)
	if errFindMany != nil && errConnection != nil {
		errMsg := errFindMany.Error() + " | " + errConnection.Error()
		if strings.Contains(errMsg, "Cannot query field \"findManyWarehouseCoreProducts\" on type \"Query\"") &&
			strings.Contains(errMsg, "Cannot query field \"warehouseCoreProducts\" on type \"Query\"") {
			return nil, fmt.Errorf("Twenty object WarehouseCoreProduct is not deployed yet; run 'yarn twenty dev --once' in your Twenty app and verify object sync")
		}
		return nil, errFindMany
	}

	return nil, fmt.Errorf("unable to parse findManyWarehouseCoreProducts response")
}

func loadExistingWarehouseCorePackages(ctx context.Context) ([]whpkgEdge, error) {
	const directFindManyQ = `query {
		findManyWarehouseCorePackages {
			id
			warehousePackageId
		}
	}`
	var direct []whpkgNode
	if err := doTwentyGraphQLRoot(ctx, directFindManyQ, nil, "findManyWarehouseCorePackages", &direct); err == nil {
		edges := make([]whpkgEdge, 0, len(direct))
		for _, n := range direct {
			edges = append(edges, whpkgEdge{Node: n})
		}
		return edges, nil
	}

	const connectionQ = `query {
		warehouseCorePackages {
			edges { node { id warehousePackageId } }
		}
	}`
	var connection whpkgListEdges
	if err := doTwentyGraphQLRoot(ctx, connectionQ, nil, "warehouseCorePackages", &connection); err == nil {
		return connection.Edges, nil
	}

	const edgesFindManyQ = `query {
		findManyWarehouseCorePackages {
			edges { node { id warehousePackageId } }
		}
	}`
	var edgeList whpkgListEdges
	if err := doTwentyGraphQLRoot(ctx, edgesFindManyQ, nil, "findManyWarehouseCorePackages", &edgeList); err == nil {
		return edgeList.Edges, nil
	}

	errFindMany := doTwentyGraphQLRoot(ctx, directFindManyQ, nil, "findManyWarehouseCorePackages", &direct)
	errConnection := doTwentyGraphQLRoot(ctx, connectionQ, nil, "warehouseCorePackages", &connection)
	if errFindMany != nil && errConnection != nil {
		errMsg := errFindMany.Error() + " | " + errConnection.Error()
		if strings.Contains(errMsg, "Cannot query field \"findManyWarehouseCorePackages\" on type \"Query\"") &&
			strings.Contains(errMsg, "Cannot query field \"warehouseCorePackages\" on type \"Query\"") {
			return nil, fmt.Errorf("Twenty object WarehouseCorePackage is not deployed yet; run 'yarn twenty dev --once' in your Twenty app and verify object sync")
		}
		return nil, errFindMany
	}

	return nil, fmt.Errorf("unable to parse findManyWarehouseCorePackages response")
}

// bootstrapTwentyJobIDs finds Twenty Opportunities that have no
// warehouseCoreJobId and auto-assigns one by inserting a row into the local
// jobs table, then writing the integer ID back to Twenty.
func bootstrapTwentyJobIDs(ctx context.Context) {
	cfg, cfgErr := services.GetTwentyConfig()
	if cfgErr != nil {
		log.Printf("[TWENTY BOOTSTRAP] failed to load config: %v", cfgErr)
		return
	}
	if !cfg.EnableJobBootstrap {
		return
	}
	if !twentyConfigured() {
		return
	}

	const findManyQ = `query {
		findManyOpportunities(filter: { warehouseCoreJobId: { is: NULL } }) {
			edges { node { id name jobCode } }
		}
	}`
	const opportunitiesQ = `query {
		opportunities(filter: { warehouseCoreJobId: { is: NULL } }) {
			edges { node { id name jobCode } }
		}
	}`
	type oppNode struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		JobCode string `json:"jobCode"`
	}
	type oppEdge struct {
		Node oppNode `json:"node"`
	}
	type oppList struct {
		Edges []oppEdge `json:"edges"`
	}

	var list oppList
	if err := doTwentyGraphQLRoot(ctx, findManyQ, nil, "findManyOpportunities", &list); err != nil {
		if err2 := doTwentyGraphQLRoot(ctx, opportunitiesQ, nil, "opportunities", &list); err2 != nil {
			log.Printf("[TWENTY BOOTSTRAP] fetch opportunities: %v", err2)
			return
		}
	}
	if len(list.Edges) == 0 {
		return
	}

	db := repository.GetSQLDB()
	for _, e := range list.Edges {
		opp := e.Node
		jobCode := opp.JobCode
		if jobCode == "" {
			jobCode = opp.Name
		}

		var localID int64
		insErr := db.QueryRowContext(ctx,
			`INSERT INTO jobs (job_code, status) VALUES ($1, 'open') RETURNING id`,
			jobCode,
		).Scan(&localID)
		if insErr != nil {
			log.Printf("[TWENTY BOOTSTRAP] insert job for opportunity %s: %v", opp.ID, insErr)
			continue
		}

		const updateOneQ = `mutation BootstrapJobID($id: ID!, $jobId: Float!) {
			updateOneOpportunity(id: $id, data: { warehouseCoreJobId: $jobId }) {
				id warehouseCoreJobId
			}
		}`
		const updateQ = `mutation BootstrapJobID($id: ID!, $jobId: Float!) {
			updateOpportunity(id: $id, data: { warehouseCoreJobId: $jobId }) {
				id warehouseCoreJobId
			}
		}`

		vars := map[string]interface{}{
			"id":    opp.ID,
			"jobId": float64(localID),
		}

		mutErr := doTwentyGraphQLRoot(ctx, updateOneQ, vars, "updateOneOpportunity", nil)
		if mutErr != nil {
			mutErr = doTwentyGraphQLRoot(ctx, updateQ, vars, "updateOpportunity", nil)
		}
		if mutErr != nil {
			log.Printf("[TWENTY BOOTSTRAP] write-back job ID %d for opportunity %s: %v", localID, opp.ID, mutErr)
			if _, delErr := db.ExecContext(ctx, `DELETE FROM jobs WHERE id = $1`, localID); delErr != nil {
				log.Printf("[TWENTY BOOTSTRAP] rollback job row %d: %v", localID, delErr)
			}
			continue
		}

		log.Printf("[TWENTY BOOTSTRAP] linked opportunity %s -> job ID %d", opp.ID, localID)
	}
}
