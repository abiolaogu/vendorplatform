// =============================================================================
// SEARCH SERVICE
// Full-text search with Elasticsearch for vendors, services, and content
// =============================================================================

package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// =============================================================================
// TYPES
// =============================================================================

// SearchRequest for querying
type SearchRequest struct {
	Query      string              `json:"query"`
	Type       SearchType          `json:"type,omitempty"`      // 'vendor', 'service', 'category', 'all'
	Filters    map[string]interface{} `json:"filters,omitempty"`
	Location   *Location           `json:"location,omitempty"`
	RadiusKM   float64             `json:"radius_km,omitempty"`
	Page       int                 `json:"page,omitempty"`
	PageSize   int                 `json:"page_size,omitempty"`
	SortBy     string              `json:"sort_by,omitempty"`   // 'relevance', 'rating', 'distance', 'price'
	SortOrder  string              `json:"sort_order,omitempty"` // 'asc', 'desc'
}

type SearchType string
const (
	TypeVendor   SearchType = "vendor"
	TypeService  SearchType = "service"
	TypeCategory SearchType = "category"
	TypeAll      SearchType = "all"
)

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// SearchResponse from search query
type SearchResponse struct {
	Query       string         `json:"query"`
	Total       int64          `json:"total"`
	Page        int            `json:"page"`
	PageSize    int            `json:"page_size"`
	TotalPages  int            `json:"total_pages"`
	Results     []SearchResult `json:"results"`
	Facets      map[string][]Facet `json:"facets,omitempty"`
	Suggestions []string       `json:"suggestions,omitempty"`
	TookMs      int64          `json:"took_ms"`
}

// SearchResult represents a single search hit
type SearchResult struct {
	ID          uuid.UUID              `json:"id"`
	Type        SearchType             `json:"type"`
	Score       float64                `json:"score"`
	Title       string                 `json:"title"`
	Description string                 `json:"description,omitempty"`
	Image       string                 `json:"image,omitempty"`
	Rating      float64                `json:"rating,omitempty"`
	ReviewCount int                    `json:"review_count,omitempty"`
	Location    *Location              `json:"location,omitempty"`
	Distance    float64                `json:"distance_km,omitempty"`
	PriceRange  string                 `json:"price_range,omitempty"`
	Categories  []string               `json:"categories,omitempty"`
	Highlights  map[string][]string    `json:"highlights,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// Facet for aggregations
type Facet struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

// VendorDocument for indexing
type VendorDocument struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Categories   []string  `json:"categories"`
	Tags         []string  `json:"tags"`
	Location     *Location `json:"location,omitempty"`
	Address      string    `json:"address"`
	City         string    `json:"city"`
	State        string    `json:"state"`
	Rating       float64   `json:"rating"`
	ReviewCount  int       `json:"review_count"`
	PriceLevel   int       `json:"price_level"` // 1-5
	IsVerified   bool      `json:"is_verified"`
	IsAvailable  bool      `json:"is_available"`
	ResponseTime int       `json:"response_time_hours"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ServiceDocument for indexing
type ServiceDocument struct {
	ID          uuid.UUID `json:"id"`
	VendorID    uuid.UUID `json:"vendor_id"`
	VendorName  string    `json:"vendor_name"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Subcategory string    `json:"subcategory"`
	Tags        []string  `json:"tags"`
	Price       int64     `json:"price"`
	Currency    string    `json:"currency"`
	PriceUnit   string    `json:"price_unit"` // 'fixed', 'hourly', 'daily'
	Duration    int       `json:"duration_minutes"`
	Rating      float64   `json:"rating"`
	BookingCount int      `json:"booking_count"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// =============================================================================
// SERVICE
// =============================================================================

// Config for search service
type Config struct {
	ElasticsearchURL string
	IndexPrefix      string
	CacheTTL         time.Duration
}

// Service handles search operations
type Service struct {
	db     *pgxpool.Pool
	cache  *redis.Client
	config *Config
	http   *http.Client
}

// NewService creates a new search service
func NewService(db *pgxpool.Pool, cache *redis.Client, config *Config) *Service {
	return &Service{
		db:     db,
		cache:  cache,
		config: config,
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

// =============================================================================
// SEARCH OPERATIONS
// =============================================================================

// Search performs a search query
func (s *Service) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	start := time.Now()
	
	// Set defaults
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Type == "" {
		req.Type = TypeAll
	}
	
	// Check cache for common queries
	cacheKey := s.buildCacheKey(req)
	if cached, err := s.cache.Get(ctx, cacheKey).Result(); err == nil {
		var resp SearchResponse
		if json.Unmarshal([]byte(cached), &resp) == nil {
			resp.TookMs = time.Since(start).Milliseconds()
			return &resp, nil
		}
	}
	
	// Build Elasticsearch query
	esQuery := s.buildElasticsearchQuery(req)
	
	// Determine indices to search
	indices := s.getIndices(req.Type)
	
	// Execute search
	resp, err := s.executeSearch(ctx, indices, esQuery)
	if err != nil {
		return nil, err
	}
	
	resp.Query = req.Query
	resp.Page = req.Page
	resp.PageSize = req.PageSize
	resp.TotalPages = int((resp.Total + int64(req.PageSize) - 1) / int64(req.PageSize))
	resp.TookMs = time.Since(start).Milliseconds()
	
	// Cache result
	respJSON, _ := json.Marshal(resp)
	s.cache.Set(ctx, cacheKey, respJSON, s.config.CacheTTL)
	
	return resp, nil
}

func (s *Service) buildCacheKey(req SearchRequest) string {
	return fmt.Sprintf("search:%s:%s:%d:%d", req.Type, req.Query, req.Page, req.PageSize)
}

func (s *Service) getIndices(searchType SearchType) string {
	prefix := s.config.IndexPrefix
	switch searchType {
	case TypeVendor:
		return prefix + "vendors"
	case TypeService:
		return prefix + "services"
	case TypeCategory:
		return prefix + "categories"
	default:
		return prefix + "vendors," + prefix + "services"
	}
}

func (s *Service) buildElasticsearchQuery(req SearchRequest) map[string]interface{} {
	query := map[string]interface{}{
		"from": (req.Page - 1) * req.PageSize,
		"size": req.PageSize,
	}
	
	// Build bool query
	must := []map[string]interface{}{}
	filter := []map[string]interface{}{}
	should := []map[string]interface{}{}
	
	// Main text search
	if req.Query != "" {
		must = append(must, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  req.Query,
				"fields": []string{"name^3", "description^2", "categories", "tags"},
				"type":   "best_fields",
				"fuzziness": "AUTO",
			},
		})
	}
	
	// Location filter
	if req.Location != nil && req.RadiusKM > 0 {
		filter = append(filter, map[string]interface{}{
			"geo_distance": map[string]interface{}{
				"distance": fmt.Sprintf("%fkm", req.RadiusKM),
				"location": map[string]float64{
					"lat": req.Location.Lat,
					"lon": req.Location.Lon,
				},
			},
		})
	}
	
	// Apply filters
	for key, value := range req.Filters {
		switch key {
		case "category":
			filter = append(filter, map[string]interface{}{
				"term": map[string]interface{}{"categories": value},
			})
		case "is_verified":
			filter = append(filter, map[string]interface{}{
				"term": map[string]interface{}{"is_verified": value},
			})
		case "min_rating":
			filter = append(filter, map[string]interface{}{
				"range": map[string]interface{}{
					"rating": map[string]interface{}{"gte": value},
				},
			})
		case "price_level":
			filter = append(filter, map[string]interface{}{
				"term": map[string]interface{}{"price_level": value},
			})
		case "city":
			filter = append(filter, map[string]interface{}{
				"term": map[string]interface{}{"city": value},
			})
		}
	}
	
	// Build bool query
	boolQuery := map[string]interface{}{}
	if len(must) > 0 {
		boolQuery["must"] = must
	}
	if len(filter) > 0 {
		boolQuery["filter"] = filter
	}
	if len(should) > 0 {
		boolQuery["should"] = should
	}
	
	if len(boolQuery) > 0 {
		query["query"] = map[string]interface{}{"bool": boolQuery}
	} else {
		query["query"] = map[string]interface{}{"match_all": map[string]interface{}{}}
	}
	
	// Sorting
	sort := []map[string]interface{}{}
	switch req.SortBy {
	case "rating":
		order := "desc"
		if req.SortOrder == "asc" {
			order = "asc"
		}
		sort = append(sort, map[string]interface{}{
			"rating": map[string]string{"order": order},
		})
	case "distance":
		if req.Location != nil {
			sort = append(sort, map[string]interface{}{
				"_geo_distance": map[string]interface{}{
					"location": map[string]float64{
						"lat": req.Location.Lat,
						"lon": req.Location.Lon,
					},
					"order": "asc",
					"unit":  "km",
				},
			})
		}
	case "price":
		order := "asc"
		if req.SortOrder == "desc" {
			order = "desc"
		}
		sort = append(sort, map[string]interface{}{
			"price_level": map[string]string{"order": order},
		})
	default: // relevance
		sort = append(sort, map[string]interface{}{
			"_score": map[string]string{"order": "desc"},
		})
	}
	query["sort"] = sort
	
	// Aggregations for facets
	query["aggs"] = map[string]interface{}{
		"categories": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "categories",
				"size":  20,
			},
		},
		"cities": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "city",
				"size":  10,
			},
		},
		"price_levels": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "price_level",
				"size":  5,
			},
		},
	}
	
	// Highlighting
	query["highlight"] = map[string]interface{}{
		"fields": map[string]interface{}{
			"name":        map[string]interface{}{},
			"description": map[string]interface{}{},
		},
		"pre_tags":  []string{"<em>"},
		"post_tags": []string{"</em>"},
	}
	
	return query
}

func (s *Service) executeSearch(ctx context.Context, indices string, query map[string]interface{}) (*SearchResponse, error) {
	body, _ := json.Marshal(query)
	
	url := fmt.Sprintf("%s/%s/_search", s.config.ElasticsearchURL, indices)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("elasticsearch error: %s", string(bodyBytes))
	}
	
	var esResp struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID        string                 `json:"_id"`
				Index     string                 `json:"_index"`
				Score     float64                `json:"_score"`
				Source    map[string]interface{} `json:"_source"`
				Highlight map[string][]string    `json:"highlight,omitempty"`
				Sort      []interface{}          `json:"sort,omitempty"`
			} `json:"hits"`
		} `json:"hits"`
		Aggregations map[string]struct {
			Buckets []struct {
				Key      string `json:"key"`
				DocCount int64  `json:"doc_count"`
			} `json:"buckets"`
		} `json:"aggregations"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&esResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	// Convert to SearchResponse
	results := make([]SearchResult, 0, len(esResp.Hits.Hits))
	for _, hit := range esResp.Hits.Hits {
		result := SearchResult{
			Score:      hit.Score,
			Highlights: hit.Highlight,
			Data:       hit.Source,
		}
		
		// Parse ID
		if id, err := uuid.Parse(hit.ID); err == nil {
			result.ID = id
		}
		
		// Determine type from index
		if strings.Contains(hit.Index, "vendors") {
			result.Type = TypeVendor
		} else if strings.Contains(hit.Index, "services") {
			result.Type = TypeService
		}
		
		// Extract common fields
		if name, ok := hit.Source["name"].(string); ok {
			result.Title = name
		}
		if desc, ok := hit.Source["description"].(string); ok {
			result.Description = desc
		}
		if rating, ok := hit.Source["rating"].(float64); ok {
			result.Rating = rating
		}
		if reviewCount, ok := hit.Source["review_count"].(float64); ok {
			result.ReviewCount = int(reviewCount)
		}
		if cats, ok := hit.Source["categories"].([]interface{}); ok {
			for _, c := range cats {
				if cat, ok := c.(string); ok {
					result.Categories = append(result.Categories, cat)
				}
			}
		}
		
		// Extract distance if geo sorted
		if len(hit.Sort) > 0 {
			if dist, ok := hit.Sort[0].(float64); ok {
				result.Distance = dist
			}
		}
		
		results = append(results, result)
	}
	
	// Parse facets
	facets := make(map[string][]Facet)
	for name, agg := range esResp.Aggregations {
		for _, bucket := range agg.Buckets {
			facets[name] = append(facets[name], Facet{
				Value: bucket.Key,
				Count: bucket.DocCount,
			})
		}
	}
	
	return &SearchResponse{
		Total:   esResp.Hits.Total.Value,
		Results: results,
		Facets:  facets,
	}, nil
}

// =============================================================================
// INDEXING OPERATIONS
// =============================================================================

// IndexVendor indexes or updates a vendor document
func (s *Service) IndexVendor(ctx context.Context, vendor *VendorDocument) error {
	index := s.config.IndexPrefix + "vendors"
	return s.indexDocument(ctx, index, vendor.ID.String(), vendor)
}

// IndexService indexes or updates a service document
func (s *Service) IndexService(ctx context.Context, service *ServiceDocument) error {
	index := s.config.IndexPrefix + "services"
	return s.indexDocument(ctx, index, service.ID.String(), service)
}

func (s *Service) indexDocument(ctx context.Context, index, id string, doc interface{}) error {
	body, _ := json.Marshal(doc)
	
	url := fmt.Sprintf("%s/%s/_doc/%s", s.config.ElasticsearchURL, index, id)
	req, _ := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("indexing failed: %s", string(bodyBytes))
	}
	
	return nil
}

// DeleteDocument removes a document from the index
func (s *Service) DeleteDocument(ctx context.Context, searchType SearchType, id string) error {
	index := s.getIndices(searchType)
	
	url := fmt.Sprintf("%s/%s/_doc/%s", s.config.ElasticsearchURL, index, id)
	req, _ := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	return nil
}

// =============================================================================
// INDEX MANAGEMENT
// =============================================================================

// CreateIndices creates the required Elasticsearch indices
func (s *Service) CreateIndices(ctx context.Context) error {
	// Vendor index mapping
	vendorMapping := map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"name":          map[string]string{"type": "text", "analyzer": "standard"},
				"description":   map[string]string{"type": "text", "analyzer": "standard"},
				"categories":    map[string]string{"type": "keyword"},
				"tags":          map[string]string{"type": "keyword"},
				"location":      map[string]string{"type": "geo_point"},
				"address":       map[string]string{"type": "text"},
				"city":          map[string]string{"type": "keyword"},
				"state":         map[string]string{"type": "keyword"},
				"rating":        map[string]string{"type": "float"},
				"review_count":  map[string]string{"type": "integer"},
				"price_level":   map[string]string{"type": "integer"},
				"is_verified":   map[string]string{"type": "boolean"},
				"is_available":  map[string]string{"type": "boolean"},
				"created_at":    map[string]string{"type": "date"},
				"updated_at":    map[string]string{"type": "date"},
			},
		},
		"settings": map[string]interface{}{
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
	}
	
	if err := s.createIndex(ctx, s.config.IndexPrefix+"vendors", vendorMapping); err != nil {
		return err
	}
	
	// Service index mapping
	serviceMapping := map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"name":          map[string]string{"type": "text", "analyzer": "standard"},
				"description":   map[string]string{"type": "text", "analyzer": "standard"},
				"vendor_id":     map[string]string{"type": "keyword"},
				"vendor_name":   map[string]string{"type": "text"},
				"category":      map[string]string{"type": "keyword"},
				"subcategory":   map[string]string{"type": "keyword"},
				"tags":          map[string]string{"type": "keyword"},
				"price":         map[string]string{"type": "long"},
				"currency":      map[string]string{"type": "keyword"},
				"rating":        map[string]string{"type": "float"},
				"booking_count": map[string]string{"type": "integer"},
				"is_active":     map[string]string{"type": "boolean"},
				"created_at":    map[string]string{"type": "date"},
			},
		},
	}
	
	return s.createIndex(ctx, s.config.IndexPrefix+"services", serviceMapping)
}

func (s *Service) createIndex(ctx context.Context, name string, mapping map[string]interface{}) error {
	body, _ := json.Marshal(mapping)
	
	url := fmt.Sprintf("%s/%s", s.config.ElasticsearchURL, name)
	req, _ := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// 400 means index already exists, which is fine
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create index: %s", string(bodyBytes))
	}
	
	return nil
}

// =============================================================================
// AUTOCOMPLETE / SUGGESTIONS
// =============================================================================

// Suggest returns autocomplete suggestions
func (s *Service) Suggest(ctx context.Context, prefix string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	
	// Check cache first
	cacheKey := fmt.Sprintf("suggest:%s", prefix)
	if cached, err := s.cache.Get(ctx, cacheKey).Result(); err == nil {
		var suggestions []string
		if json.Unmarshal([]byte(cached), &suggestions) == nil {
			return suggestions, nil
		}
	}
	
	// Query from database (more reliable for autocomplete)
	query := `
		SELECT DISTINCT name FROM (
			SELECT name FROM vendors WHERE name ILIKE $1 AND status = 'active'
			UNION
			SELECT name FROM services WHERE name ILIKE $1 AND is_active = TRUE
			UNION
			SELECT name FROM service_categories WHERE name ILIKE $1
		) AS suggestions
		ORDER BY name
		LIMIT $2
	`
	
	rows, err := s.db.Query(ctx, query, prefix+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var suggestions []string
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			suggestions = append(suggestions, name)
		}
	}
	
	// Cache for 5 minutes
	suggestionsJSON, _ := json.Marshal(suggestions)
	s.cache.Set(ctx, cacheKey, suggestionsJSON, 5*time.Minute)
	
	return suggestions, nil
}

// =============================================================================
// REINDEXING
// =============================================================================

// ReindexVendors reindexes all vendors from the database
func (s *Service) ReindexVendors(ctx context.Context) error {
	rows, err := s.db.Query(ctx, `
		SELECT v.id, v.business_name, v.description, v.categories, v.tags,
		       ST_X(v.location::geometry) as lon, ST_Y(v.location::geometry) as lat,
		       v.address, v.city, v.state, v.rating, v.review_count, v.price_level,
		       v.is_verified, v.is_available, v.created_at, v.updated_at
		FROM vendors v
		WHERE v.status = 'active'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var doc VendorDocument
		var lon, lat *float64
		var categories, tags []string
		
		err := rows.Scan(
			&doc.ID, &doc.Name, &doc.Description, &categories, &tags,
			&lon, &lat, &doc.Address, &doc.City, &doc.State,
			&doc.Rating, &doc.ReviewCount, &doc.PriceLevel,
			&doc.IsVerified, &doc.IsAvailable, &doc.CreatedAt, &doc.UpdatedAt,
		)
		if err != nil {
			continue
		}
		
		doc.Categories = categories
		doc.Tags = tags
		if lon != nil && lat != nil {
			doc.Location = &Location{Lat: *lat, Lon: *lon}
		}
		
		s.IndexVendor(ctx, &doc)
	}
	
	return nil
}

// ReindexServices reindexes all services from the database
func (s *Service) ReindexServices(ctx context.Context) error {
	rows, err := s.db.Query(ctx, `
		SELECT s.id, s.vendor_id, v.business_name, s.name, s.description,
		       s.category, s.subcategory, s.tags, s.price, s.currency,
		       s.rating, s.booking_count, s.is_active, s.created_at
		FROM services s
		JOIN vendors v ON v.id = s.vendor_id
		WHERE s.is_active = TRUE AND v.status = 'active'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var doc ServiceDocument
		var tags []string
		
		err := rows.Scan(
			&doc.ID, &doc.VendorID, &doc.VendorName, &doc.Name, &doc.Description,
			&doc.Category, &doc.Subcategory, &tags, &doc.Price, &doc.Currency,
			&doc.Rating, &doc.BookingCount, &doc.IsActive, &doc.CreatedAt,
		)
		if err != nil {
			continue
		}
		
		doc.Tags = tags
		s.IndexService(ctx, &doc)
	}
	
	return nil
}
