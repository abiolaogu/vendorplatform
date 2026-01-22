// Package recommendation provides a high-performance, production-ready
// recommendation engine for the Vendor & Artisans Platform.
// It implements adjacency-based, collaborative filtering, and contextual recommendations.
package recommendation

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// =============================================================================
// CORE TYPES & INTERFACES
// =============================================================================

// RecommendationType defines the kind of recommendation
type RecommendationType string

const (
	AdjacentService    RecommendationType = "adjacent_service"
	SimilarVendor      RecommendationType = "similar_vendor"
	BundleSuggestion   RecommendationType = "bundle"
	TrendingService    RecommendationType = "trending"
	PersonalizedPick   RecommendationType = "personalized"
	ContextualUpsell   RecommendationType = "contextual_upsell"
	EventBasedSuggest  RecommendationType = "event_based"
	CollaborativeFilter RecommendationType = "collaborative"
)

// EntityType defines what's being recommended
type EntityType string

const (
	EntityVendor   EntityType = "vendor"
	EntityService  EntityType = "service"
	EntityCategory EntityType = "category"
	EntityBundle   EntityType = "bundle"
)

// Recommendation represents a single recommendation
type Recommendation struct {
	ID               uuid.UUID          `json:"id"`
	Type             RecommendationType `json:"type"`
	EntityType       EntityType         `json:"entity_type"`
	EntityID         uuid.UUID          `json:"entity_id"`
	Score            float64            `json:"score"`
	RelevanceScore   float64            `json:"relevance_score"`
	DiversityScore   float64            `json:"diversity_score"`
	ExplanationCopy  string             `json:"explanation_copy"`
	Position         int                `json:"position"`
	Metadata         map[string]any     `json:"metadata"`
	SourceContext    *SourceContext     `json:"source_context,omitempty"`
}

// SourceContext provides context for why a recommendation was made
type SourceContext struct {
	TriggerType      string    `json:"trigger_type"`
	TriggerEntityID  uuid.UUID `json:"trigger_entity_id,omitempty"`
	EventType        string    `json:"event_type,omitempty"`
	ProjectID        uuid.UUID `json:"project_id,omitempty"`
	SearchQuery      string    `json:"search_query,omitempty"`
}

// RecommendationRequest encapsulates a recommendation query
type RecommendationRequest struct {
	UserID          uuid.UUID          `json:"user_id,omitempty"`
	SessionID       uuid.UUID          `json:"session_id,omitempty"`
	ProjectID       uuid.UUID          `json:"project_id,omitempty"`
	CurrentEntityID uuid.UUID          `json:"current_entity_id,omitempty"`
	CurrentEntityType EntityType       `json:"current_entity_type,omitempty"`
	EventType       string             `json:"event_type,omitempty"`
	Location        *GeoPoint          `json:"location,omitempty"`
	Budget          *BudgetRange       `json:"budget,omitempty"`
	RequestedTypes  []RecommendationType `json:"requested_types,omitempty"`
	Limit           int                `json:"limit"`
	ExcludeIDs      []uuid.UUID        `json:"exclude_ids,omitempty"`
	DiversityFactor float64            `json:"diversity_factor"` // 0-1, higher = more diverse
}

// GeoPoint represents a geographic location
type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// BudgetRange represents min/max budget
type BudgetRange struct {
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Currency string  `json:"currency"`
}

// RecommendationResponse contains the recommendation results
type RecommendationResponse struct {
	Recommendations []Recommendation `json:"recommendations"`
	TotalCandidates int              `json:"total_candidates"`
	AlgorithmVersion string          `json:"algorithm_version"`
	ProcessingTimeMs int64           `json:"processing_time_ms"`
	ExperimentID    uuid.UUID        `json:"experiment_id,omitempty"`
	Variant         string           `json:"variant,omitempty"`
}

// =============================================================================
// ENGINE CORE
// =============================================================================

// Engine is the main recommendation engine
type Engine struct {
	db              *pgxpool.Pool
	cache           *redis.Client
	config          *Config
	adjacencyGraph  *AdjacencyGraph
	userProfiler    *UserProfiler
	eventDetector   *EventDetector
	trendingService *TrendingService
	scorer          *Scorer
	ranker          *Ranker
	diversifier     *Diversifier
	mu              sync.RWMutex
}

// Config holds engine configuration
type Config struct {
	// Caching
	CacheTTL              time.Duration
	AdjacencyRefreshRate  time.Duration
	
	// Scoring weights
	AdjacencyWeight       float64
	CollaborativeWeight   float64
	TrendingWeight        float64
	PersonalizationWeight float64
	LocationWeight        float64
	RecencyWeight         float64
	
	// Diversity
	MinDiversityScore     float64
	CategoryDiversityBonus float64
	
	// Performance
	MaxCandidates         int
	ParallelScoring       bool
	ScoringWorkers        int
	
	// A/B Testing
	EnableExperiments     bool
	DefaultVariant        string
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	return &Config{
		CacheTTL:              5 * time.Minute,
		AdjacencyRefreshRate:  1 * time.Hour,
		AdjacencyWeight:       0.35,
		CollaborativeWeight:   0.25,
		TrendingWeight:        0.15,
		PersonalizationWeight: 0.20,
		LocationWeight:        0.05,
		RecencyWeight:         0.10,
		MinDiversityScore:     0.3,
		CategoryDiversityBonus: 0.1,
		MaxCandidates:         500,
		ParallelScoring:       true,
		ScoringWorkers:        4,
		EnableExperiments:     true,
		DefaultVariant:        "control",
	}
}

// NewEngine creates a new recommendation engine
func NewEngine(db *pgxpool.Pool, cache *redis.Client, config *Config) (*Engine, error) {
	if config == nil {
		config = DefaultConfig()
	}
	
	engine := &Engine{
		db:     db,
		cache:  cache,
		config: config,
	}
	
	// Initialize components
	engine.adjacencyGraph = NewAdjacencyGraph(db, cache)
	engine.userProfiler = NewUserProfiler(db, cache)
	engine.eventDetector = NewEventDetector(db)
	engine.trendingService = NewTrendingService(db, cache)
	engine.scorer = NewScorer(config)
	engine.ranker = NewRanker(config)
	engine.diversifier = NewDiversifier(config)
	
	// Load adjacency graph into memory
	if err := engine.adjacencyGraph.Load(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to load adjacency graph: %w", err)
	}
	
	// Start background refresh
	go engine.backgroundRefresh()
	
	return engine, nil
}

// GetRecommendations is the main entry point for getting recommendations
func (e *Engine) GetRecommendations(ctx context.Context, req *RecommendationRequest) (*RecommendationResponse, error) {
	startTime := time.Now()
	
	// Validate request
	if err := e.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}
	
	// Set defaults
	if req.Limit == 0 {
		req.Limit = 10
	}
	if req.DiversityFactor == 0 {
		req.DiversityFactor = 0.3
	}
	
	// Build user context
	userCtx, err := e.buildUserContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to build user context: %w", err)
	}
	
	// Generate candidates from multiple sources
	candidates, err := e.generateCandidates(ctx, req, userCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate candidates: %w", err)
	}
	
	// Score candidates
	scoredCandidates := e.scorer.ScoreAll(ctx, candidates, req, userCtx)
	
	// Rank and diversify
	ranked := e.ranker.Rank(scoredCandidates)
	diversified := e.diversifier.Diversify(ranked, req.Limit, req.DiversityFactor)
	
	// Build response
	response := &RecommendationResponse{
		Recommendations:   diversified,
		TotalCandidates:   len(candidates),
		AlgorithmVersion:  "v2.1.0",
		ProcessingTimeMs:  time.Since(startTime).Milliseconds(),
	}
	
	// Add experiment info if enabled
	if e.config.EnableExperiments {
		response.ExperimentID = uuid.New() // Would come from experiment service
		response.Variant = e.config.DefaultVariant
	}
	
	// Log recommendations for analytics (async)
	go e.logRecommendations(ctx, req, response)
	
	return response, nil
}

// =============================================================================
// CANDIDATE GENERATION
// =============================================================================

// Candidate represents a potential recommendation before scoring
type Candidate struct {
	EntityType    EntityType
	EntityID      uuid.UUID
	CategoryID    uuid.UUID
	Source        RecommendationType
	BaseScore     float64
	Metadata      map[string]any
}

func (e *Engine) generateCandidates(ctx context.Context, req *RecommendationRequest, userCtx *UserContext) ([]Candidate, error) {
	var allCandidates []Candidate
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	// Determine which generators to use
	generators := e.selectGenerators(req)
	
	for _, gen := range generators {
		wg.Add(1)
		go func(g CandidateGenerator) {
			defer wg.Done()
			candidates, err := g.Generate(ctx, req, userCtx)
			if err != nil {
				// Log error but don't fail
				return
			}
			mu.Lock()
			allCandidates = append(allCandidates, candidates...)
			mu.Unlock()
		}(gen)
	}
	
	wg.Wait()
	
	// Deduplicate
	return e.deduplicateCandidates(allCandidates), nil
}

// CandidateGenerator interface for different recommendation sources
type CandidateGenerator interface {
	Generate(ctx context.Context, req *RecommendationRequest, userCtx *UserContext) ([]Candidate, error)
}

// =============================================================================
// ADJACENCY-BASED GENERATOR
// =============================================================================

// AdjacencyGenerator generates recommendations based on service adjacencies
type AdjacencyGenerator struct {
	graph *AdjacencyGraph
	db    *pgxpool.Pool
}

func (g *AdjacencyGenerator) Generate(ctx context.Context, req *RecommendationRequest, userCtx *UserContext) ([]Candidate, error) {
	if req.CurrentEntityID == uuid.Nil {
		return nil, nil
	}
	
	// Get the category of the current entity
	var categoryID uuid.UUID
	switch req.CurrentEntityType {
	case EntityService:
		categoryID = g.getCategoryForService(ctx, req.CurrentEntityID)
	case EntityCategory:
		categoryID = req.CurrentEntityID
	case EntityVendor:
		categoryID = g.getPrimaryCategoryForVendor(ctx, req.CurrentEntityID)
	default:
		return nil, nil
	}
	
	if categoryID == uuid.Nil {
		return nil, nil
	}
	
	// Get adjacent categories from the graph
	adjacentCategories := g.graph.GetAdjacent(categoryID, req.EventType, 20)
	
	var candidates []Candidate
	for _, adj := range adjacentCategories {
		// Get top vendors/services for each adjacent category
		services := g.getTopServicesForCategory(ctx, adj.TargetCategoryID, req.Location, 5)
		
		for _, svc := range services {
			candidates = append(candidates, Candidate{
				EntityType: EntityService,
				EntityID:   svc.ID,
				CategoryID: adj.TargetCategoryID,
				Source:     AdjacentService,
				BaseScore:  adj.Score,
				Metadata: map[string]any{
					"adjacency_type":      adj.AdjacencyType,
					"recommendation_copy": adj.RecommendationCopy,
					"source_category":     categoryID,
					"target_category":     adj.TargetCategoryID,
				},
			})
		}
	}
	
	return candidates, nil
}

func (g *AdjacencyGenerator) getCategoryForService(ctx context.Context, serviceID uuid.UUID) uuid.UUID {
	var categoryID uuid.UUID
	g.db.QueryRow(ctx, "SELECT category_id FROM services WHERE id = $1", serviceID).Scan(&categoryID)
	return categoryID
}

func (g *AdjacencyGenerator) getPrimaryCategoryForVendor(ctx context.Context, vendorID uuid.UUID) uuid.UUID {
	var categoryID uuid.UUID
	g.db.QueryRow(ctx, `
		SELECT s.category_id 
		FROM services s 
		WHERE s.vendor_id = $1 
		ORDER BY s.booking_count DESC 
		LIMIT 1
	`, vendorID).Scan(&categoryID)
	return categoryID
}

type ServiceInfo struct {
	ID         uuid.UUID
	VendorID   uuid.UUID
	Rating     float64
	BookingCount int
}

func (g *AdjacencyGenerator) getTopServicesForCategory(ctx context.Context, categoryID uuid.UUID, loc *GeoPoint, limit int) []ServiceInfo {
	query := `
		SELECT s.id, s.vendor_id, s.rating_average, s.booking_count
		FROM services s
		JOIN vendors v ON v.id = s.vendor_id
		WHERE s.category_id = $1
		  AND s.is_available = TRUE
		  AND v.is_active = TRUE
	`
	args := []any{categoryID}
	
	if loc != nil {
		query += ` AND ST_DWithin(v.service_location, ST_MakePoint($2, $3)::geography, v.service_radius_km * 1000)`
		args = append(args, loc.Longitude, loc.Latitude)
	}
	
	query += ` ORDER BY s.rating_average DESC, s.booking_count DESC LIMIT $` + fmt.Sprintf("%d", len(args)+1)
	args = append(args, limit)
	
	rows, err := g.db.Query(ctx, query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	
	var services []ServiceInfo
	for rows.Next() {
		var s ServiceInfo
		if err := rows.Scan(&s.ID, &s.VendorID, &s.Rating, &s.BookingCount); err != nil {
			continue
		}
		services = append(services, s)
	}
	
	return services
}

// =============================================================================
// EVENT-BASED GENERATOR
// =============================================================================

// EventBasedGenerator generates recommendations based on detected life events
type EventBasedGenerator struct {
	db            *pgxpool.Pool
	eventDetector *EventDetector
}

func (g *EventBasedGenerator) Generate(ctx context.Context, req *RecommendationRequest, userCtx *UserContext) ([]Candidate, error) {
	// If event type is provided, use it directly
	eventType := req.EventType
	
	// Otherwise, try to detect from user context
	if eventType == "" && userCtx.DetectedEvents != nil && len(userCtx.DetectedEvents) > 0 {
		eventType = userCtx.DetectedEvents[0].EventType
	}
	
	if eventType == "" {
		return nil, nil
	}
	
	// Get required categories for this event
	categories, err := g.getCategoriesForEvent(ctx, eventType, userCtx.AlreadyBookedCategories)
	if err != nil {
		return nil, err
	}
	
	var candidates []Candidate
	for _, cat := range categories {
		// Get services for each needed category
		services := g.getTopServicesForCategory(ctx, cat.CategoryID, req.Location, 3)
		
		for _, svc := range services {
			candidates = append(candidates, Candidate{
				EntityType: EntityService,
				EntityID:   svc.ID,
				CategoryID: cat.CategoryID,
				Source:     EventBasedSuggest,
				BaseScore:  cat.NecessityScore * cat.PopularityScore,
				Metadata: map[string]any{
					"event_type":       eventType,
					"role_type":        cat.RoleType,
					"phase":            cat.Phase,
					"necessity_score":  cat.NecessityScore,
					"budget_percentage": cat.BudgetPercentage,
				},
			})
		}
	}
	
	return candidates, nil
}

type EventCategory struct {
	CategoryID       uuid.UUID
	RoleType         string
	Phase            string
	NecessityScore   float64
	PopularityScore  float64
	BudgetPercentage float64
}

func (g *EventBasedGenerator) getCategoriesForEvent(ctx context.Context, eventType string, alreadyBooked []uuid.UUID) ([]EventCategory, error) {
	query := `
		SELECT ecm.category_id, ecm.role_type, ecm.phase, 
		       ecm.necessity_score, ecm.popularity_score, ecm.typical_budget_percentage
		FROM event_category_mappings ecm
		JOIN life_event_triggers let ON let.id = ecm.event_trigger_id
		WHERE let.slug = $1
		  AND ecm.is_active = TRUE
		  AND ecm.category_id != ALL($2)
		ORDER BY ecm.necessity_score DESC, ecm.popularity_score DESC
	`
	
	rows, err := g.db.Query(ctx, query, eventType, alreadyBooked)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var categories []EventCategory
	for rows.Next() {
		var c EventCategory
		if err := rows.Scan(&c.CategoryID, &c.RoleType, &c.Phase, 
			&c.NecessityScore, &c.PopularityScore, &c.BudgetPercentage); err != nil {
			continue
		}
		categories = append(categories, c)
	}
	
	return categories, nil
}

func (g *EventBasedGenerator) getTopServicesForCategory(ctx context.Context, categoryID uuid.UUID, loc *GeoPoint, limit int) []ServiceInfo {
	// Implementation similar to AdjacencyGenerator
	query := `
		SELECT s.id, s.vendor_id, s.rating_average, s.booking_count
		FROM services s
		JOIN vendors v ON v.id = s.vendor_id
		WHERE s.category_id = $1 AND s.is_available = TRUE AND v.is_active = TRUE
		ORDER BY s.rating_average DESC, s.booking_count DESC
		LIMIT $2
	`
	
	rows, err := g.db.Query(ctx, query, categoryID, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	
	var services []ServiceInfo
	for rows.Next() {
		var s ServiceInfo
		if err := rows.Scan(&s.ID, &s.VendorID, &s.Rating, &s.BookingCount); err != nil {
			continue
		}
		services = append(services, s)
	}
	
	return services
}

// =============================================================================
// COLLABORATIVE FILTERING GENERATOR
// =============================================================================

// CollaborativeGenerator uses user behavior patterns for recommendations
type CollaborativeGenerator struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

func (g *CollaborativeGenerator) Generate(ctx context.Context, req *RecommendationRequest, userCtx *UserContext) ([]Candidate, error) {
	if req.UserID == uuid.Nil {
		return nil, nil
	}
	
	// Find similar users based on booking patterns
	similarUserIDs, err := g.findSimilarUsers(ctx, req.UserID, 50)
	if err != nil {
		return nil, err
	}
	
	if len(similarUserIDs) == 0 {
		return nil, nil
	}
	
	// Get popular items among similar users that current user hasn't booked
	popularItems, err := g.getPopularAmongSimilar(ctx, similarUserIDs, userCtx.BookedServiceIDs, 20)
	if err != nil {
		return nil, err
	}
	
	var candidates []Candidate
	for _, item := range popularItems {
		candidates = append(candidates, Candidate{
			EntityType: EntityService,
			EntityID:   item.ServiceID,
			CategoryID: item.CategoryID,
			Source:     CollaborativeFilter,
			BaseScore:  item.Score,
			Metadata: map[string]any{
				"similar_user_count": item.SimilarUserCount,
				"booking_frequency":  item.BookingFrequency,
			},
		})
	}
	
	return candidates, nil
}

func (g *CollaborativeGenerator) findSimilarUsers(ctx context.Context, userID uuid.UUID, limit int) ([]uuid.UUID, error) {
	// Find users with similar booking patterns using Jaccard similarity
	query := `
		WITH user_categories AS (
			SELECT DISTINCT s.category_id
			FROM bookings b
			JOIN services s ON s.id = b.service_id
			WHERE b.user_id = $1 AND b.status IN ('completed', 'confirmed')
		),
		other_user_categories AS (
			SELECT b.user_id, ARRAY_AGG(DISTINCT s.category_id) as categories
			FROM bookings b
			JOIN services s ON s.id = b.service_id
			WHERE b.user_id != $1 AND b.status IN ('completed', 'confirmed')
			GROUP BY b.user_id
		),
		similarity AS (
			SELECT 
				ouc.user_id,
				CARDINALITY(ARRAY(SELECT UNNEST(ouc.categories) INTERSECT SELECT category_id FROM user_categories)) * 1.0 /
				CARDINALITY(ARRAY(SELECT UNNEST(ouc.categories) UNION SELECT category_id FROM user_categories)) as jaccard
			FROM other_user_categories ouc
		)
		SELECT user_id FROM similarity WHERE jaccard > 0.2 ORDER BY jaccard DESC LIMIT $2
	`
	
	rows, err := g.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var userIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			continue
		}
		userIDs = append(userIDs, id)
	}
	
	return userIDs, nil
}

type PopularItem struct {
	ServiceID        uuid.UUID
	CategoryID       uuid.UUID
	Score            float64
	SimilarUserCount int
	BookingFrequency float64
}

func (g *CollaborativeGenerator) getPopularAmongSimilar(ctx context.Context, similarUserIDs []uuid.UUID, excludeServices []uuid.UUID, limit int) ([]PopularItem, error) {
	query := `
		SELECT s.id, s.category_id, 
		       COUNT(DISTINCT b.user_id) as similar_user_count,
		       COUNT(b.id) as booking_count
		FROM bookings b
		JOIN services s ON s.id = b.service_id
		WHERE b.user_id = ANY($1)
		  AND b.status IN ('completed', 'confirmed')
		  AND s.id != ALL($2)
		  AND s.is_available = TRUE
		GROUP BY s.id, s.category_id
		ORDER BY similar_user_count DESC, booking_count DESC
		LIMIT $3
	`
	
	rows, err := g.db.Query(ctx, query, similarUserIDs, excludeServices, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var items []PopularItem
	maxCount := 0
	for rows.Next() {
		var item PopularItem
		var bookingCount int
		if err := rows.Scan(&item.ServiceID, &item.CategoryID, &item.SimilarUserCount, &bookingCount); err != nil {
			continue
		}
		if item.SimilarUserCount > maxCount {
			maxCount = item.SimilarUserCount
		}
		item.BookingFrequency = float64(bookingCount) / float64(len(similarUserIDs))
		items = append(items, item)
	}
	
	// Normalize scores
	for i := range items {
		items[i].Score = float64(items[i].SimilarUserCount) / float64(maxCount)
	}
	
	return items, nil
}

// =============================================================================
// TRENDING GENERATOR
// =============================================================================

// TrendingGenerator recommends currently popular services
type TrendingGenerator struct {
	service *TrendingService
}

func (g *TrendingGenerator) Generate(ctx context.Context, req *RecommendationRequest, userCtx *UserContext) ([]Candidate, error) {
	trending := g.service.GetTrending(ctx, req.Location, 20)
	
	var candidates []Candidate
	for _, item := range trending {
		candidates = append(candidates, Candidate{
			EntityType: EntityService,
			EntityID:   item.ServiceID,
			CategoryID: item.CategoryID,
			Source:     TrendingService,
			BaseScore:  item.TrendScore,
			Metadata: map[string]any{
				"view_count_7d":    item.ViewCount7D,
				"booking_count_7d": item.BookingCount7D,
				"growth_rate":      item.GrowthRate,
			},
		})
	}
	
	return candidates, nil
}

// =============================================================================
// ADJACENCY GRAPH (In-Memory)
// =============================================================================

// AdjacencyGraph maintains the service adjacency relationships in memory
type AdjacencyGraph struct {
	db       *pgxpool.Pool
	cache    *redis.Client
	mu       sync.RWMutex
	edges    map[uuid.UUID][]AdjacencyEdge // source -> targets
	contexts map[string]map[uuid.UUID][]AdjacencyEdge // context -> source -> targets
	lastLoad time.Time
}

// AdjacencyEdge represents a connection between categories
type AdjacencyEdge struct {
	SourceCategoryID   uuid.UUID
	TargetCategoryID   uuid.UUID
	AdjacencyType      string
	Score              float64
	RecommendationCopy string
	TriggerContext     string
}

func NewAdjacencyGraph(db *pgxpool.Pool, cache *redis.Client) *AdjacencyGraph {
	return &AdjacencyGraph{
		db:       db,
		cache:    cache,
		edges:    make(map[uuid.UUID][]AdjacencyEdge),
		contexts: make(map[string]map[uuid.UUID][]AdjacencyEdge),
	}
}

// Load loads the adjacency graph from database
func (g *AdjacencyGraph) Load(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	query := `
		SELECT source_category_id, target_category_id, adjacency_type,
		       computed_score, recommendation_copy, COALESCE(trigger_context, '')
		FROM service_adjacencies
		WHERE is_active = TRUE
		ORDER BY computed_score DESC
	`
	
	rows, err := g.db.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	// Reset
	g.edges = make(map[uuid.UUID][]AdjacencyEdge)
	g.contexts = make(map[string]map[uuid.UUID][]AdjacencyEdge)
	
	for rows.Next() {
		var edge AdjacencyEdge
		if err := rows.Scan(&edge.SourceCategoryID, &edge.TargetCategoryID,
			&edge.AdjacencyType, &edge.Score, &edge.RecommendationCopy,
			&edge.TriggerContext); err != nil {
			continue
		}
		
		// Add to general edges
		g.edges[edge.SourceCategoryID] = append(g.edges[edge.SourceCategoryID], edge)
		
		// Add to context-specific map
		if edge.TriggerContext != "" {
			if g.contexts[edge.TriggerContext] == nil {
				g.contexts[edge.TriggerContext] = make(map[uuid.UUID][]AdjacencyEdge)
			}
			g.contexts[edge.TriggerContext][edge.SourceCategoryID] = append(
				g.contexts[edge.TriggerContext][edge.SourceCategoryID], edge)
		}
	}
	
	g.lastLoad = time.Now()
	return nil
}

// GetAdjacent returns adjacent categories for a given source
func (g *AdjacencyGraph) GetAdjacent(sourceID uuid.UUID, context string, limit int) []AdjacencyEdge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	var edges []AdjacencyEdge
	
	// Try context-specific first
	if context != "" {
		if contextEdges, ok := g.contexts[context]; ok {
			if srcEdges, ok := contextEdges[sourceID]; ok {
				edges = append(edges, srcEdges...)
			}
		}
	}
	
	// Fallback to general edges
	if len(edges) == 0 {
		edges = g.edges[sourceID]
	}
	
	// Sort by score
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Score > edges[j].Score
	})
	
	if limit > 0 && len(edges) > limit {
		edges = edges[:limit]
	}
	
	return edges
}

// =============================================================================
// USER CONTEXT & PROFILING
// =============================================================================

// UserContext contains enriched user information for personalization
type UserContext struct {
	UserID                  uuid.UUID
	IsAuthenticated         bool
	LifeStage               string
	Interests               []string
	PreferredCategories     []uuid.UUID
	BookedServiceIDs        []uuid.UUID
	AlreadyBookedCategories []uuid.UUID
	ViewedServiceIDs        []uuid.UUID
	LocationPreferences     *GeoPoint
	BudgetRange             *BudgetRange
	DetectedEvents          []DetectedEvent
	RecentSearches          []string
	SessionHistory          []SessionAction
}

// DetectedEvent represents a detected life event for the user
type DetectedEvent struct {
	EventType   string
	Confidence  float64
	DetectedAt  time.Time
	TriggerData map[string]any
}

// SessionAction represents a user action in the current session
type SessionAction struct {
	ActionType string
	EntityID   uuid.UUID
	Timestamp  time.Time
}

// UserProfiler builds user context from various sources
type UserProfiler struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

func NewUserProfiler(db *pgxpool.Pool, cache *redis.Client) *UserProfiler {
	return &UserProfiler{db: db, cache: cache}
}

func (p *UserProfiler) BuildContext(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (*UserContext, error) {
	uc := &UserContext{
		UserID:          userID,
		IsAuthenticated: userID != uuid.Nil,
	}
	
	if userID == uuid.Nil {
		return uc, nil
	}
	
	// Get user profile
	if err := p.loadUserProfile(ctx, uc); err != nil {
		return nil, err
	}
	
	// Get booking history
	if err := p.loadBookingHistory(ctx, uc); err != nil {
		return nil, err
	}
	
	// Get view history
	if err := p.loadViewHistory(ctx, uc); err != nil {
		return nil, err
	}
	
	// Get recent searches
	if err := p.loadSearchHistory(ctx, uc); err != nil {
		return nil, err
	}
	
	return uc, nil
}

func (p *UserProfiler) loadUserProfile(ctx context.Context, uc *UserContext) error {
	query := `
		SELECT life_stage, interests, 
		       ST_Y(current_location::geometry), ST_X(current_location::geometry)
		FROM users WHERE id = $1
	`
	
	var lat, lon *float64
	var interests []string
	var lifeStage *string
	
	err := p.db.QueryRow(ctx, query, uc.UserID).Scan(&lifeStage, &interests, &lat, &lon)
	if err != nil {
		return nil // User might not exist
	}
	
	if lifeStage != nil {
		uc.LifeStage = *lifeStage
	}
	uc.Interests = interests
	
	if lat != nil && lon != nil {
		uc.LocationPreferences = &GeoPoint{Latitude: *lat, Longitude: *lon}
	}
	
	return nil
}

func (p *UserProfiler) loadBookingHistory(ctx context.Context, uc *UserContext) error {
	query := `
		SELECT DISTINCT s.id, s.category_id
		FROM bookings b
		JOIN services s ON s.id = b.service_id
		WHERE b.user_id = $1 AND b.status IN ('completed', 'confirmed')
		ORDER BY b.created_at DESC
		LIMIT 100
	`
	
	rows, err := p.db.Query(ctx, query, uc.UserID)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	categorySet := make(map[uuid.UUID]bool)
	for rows.Next() {
		var serviceID, categoryID uuid.UUID
		if err := rows.Scan(&serviceID, &categoryID); err != nil {
			continue
		}
		uc.BookedServiceIDs = append(uc.BookedServiceIDs, serviceID)
		if !categorySet[categoryID] {
			uc.AlreadyBookedCategories = append(uc.AlreadyBookedCategories, categoryID)
			categorySet[categoryID] = true
		}
	}
	
	return nil
}

func (p *UserProfiler) loadViewHistory(ctx context.Context, uc *UserContext) error {
	query := `
		SELECT DISTINCT entity_id
		FROM user_interactions
		WHERE user_id = $1 
		  AND entity_type = 'service' 
		  AND interaction_type = 'view'
		  AND created_at > NOW() - INTERVAL '7 days'
		LIMIT 50
	`
	
	rows, err := p.db.Query(ctx, query, uc.UserID)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			continue
		}
		uc.ViewedServiceIDs = append(uc.ViewedServiceIDs, id)
	}
	
	return nil
}

func (p *UserProfiler) loadSearchHistory(ctx context.Context, uc *UserContext) error {
	query := `
		SELECT normalized_query
		FROM search_history
		WHERE user_id = $1 AND created_at > NOW() - INTERVAL '24 hours'
		ORDER BY created_at DESC
		LIMIT 10
	`
	
	rows, err := p.db.Query(ctx, query, uc.UserID)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var query string
		if err := rows.Scan(&query); err != nil {
			continue
		}
		uc.RecentSearches = append(uc.RecentSearches, query)
	}
	
	return nil
}

// =============================================================================
// SCORING
// =============================================================================

// Scorer calculates the final score for each candidate
type Scorer struct {
	config *Config
}

func NewScorer(config *Config) *Scorer {
	return &Scorer{config: config}
}

func (s *Scorer) ScoreAll(ctx context.Context, candidates []Candidate, req *RecommendationRequest, userCtx *UserContext) []Recommendation {
	recs := make([]Recommendation, 0, len(candidates))
	
	for _, c := range candidates {
		rec := s.scoreCandidate(c, req, userCtx)
		recs = append(recs, rec)
	}
	
	return recs
}

func (s *Scorer) scoreCandidate(c Candidate, req *RecommendationRequest, userCtx *UserContext) Recommendation {
	// Start with base score from the source
	baseScore := c.BaseScore
	
	// Apply weight based on source type
	sourceWeight := s.getSourceWeight(c.Source)
	weightedBase := baseScore * sourceWeight
	
	// Calculate personalization boost
	personalizationBoost := s.calculatePersonalizationBoost(c, userCtx)
	
	// Calculate relevance score
	relevanceScore := s.calculateRelevance(c, req, userCtx)
	
	// Calculate recency boost (if applicable)
	recencyBoost := 0.0
	if c.Metadata != nil {
		if growth, ok := c.Metadata["growth_rate"].(float64); ok {
			recencyBoost = math.Min(growth*0.1, 0.2) // Cap at 0.2
		}
	}
	
	// Final score
	finalScore := weightedBase + 
		(personalizationBoost * s.config.PersonalizationWeight) +
		(relevanceScore * 0.2) +
		(recencyBoost * s.config.RecencyWeight)
	
	// Normalize to 0-1
	finalScore = math.Min(1.0, math.Max(0.0, finalScore))
	
	// Build explanation
	explanation := s.buildExplanation(c, userCtx)
	
	return Recommendation{
		ID:              uuid.New(),
		Type:            c.Source,
		EntityType:      c.EntityType,
		EntityID:        c.EntityID,
		Score:           finalScore,
		RelevanceScore:  relevanceScore,
		ExplanationCopy: explanation,
		Metadata:        c.Metadata,
	}
}

func (s *Scorer) getSourceWeight(source RecommendationType) float64 {
	switch source {
	case AdjacentService:
		return s.config.AdjacencyWeight
	case CollaborativeFilter:
		return s.config.CollaborativeWeight
	case TrendingService:
		return s.config.TrendingWeight
	case EventBasedSuggest:
		return 0.4 // High weight for event-based
	default:
		return 0.2
	}
}

func (s *Scorer) calculatePersonalizationBoost(c Candidate, userCtx *UserContext) float64 {
	boost := 0.0
	
	// Boost if category matches user interests
	for _, interest := range userCtx.Interests {
		// Would need category name lookup
		_ = interest
		boost += 0.05
	}
	
	// Boost if similar to previously booked categories
	for _, bookedCat := range userCtx.PreferredCategories {
		if bookedCat == c.CategoryID {
			boost += 0.15
			break
		}
	}
	
	// Negative boost if already viewed but not booked (might indicate disinterest)
	for _, viewedID := range userCtx.ViewedServiceIDs {
		if viewedID == c.EntityID {
			boost -= 0.05
			break
		}
	}
	
	return math.Min(0.3, boost) // Cap boost
}

func (s *Scorer) calculateRelevance(c Candidate, req *RecommendationRequest, userCtx *UserContext) float64 {
	relevance := 0.5 // Base relevance
	
	// Boost for event match
	if req.EventType != "" {
		if ctx, ok := c.Metadata["event_type"].(string); ok && ctx == req.EventType {
			relevance += 0.3
		}
	}
	
	// Boost for budget match
	// Would need service price lookup
	
	return math.Min(1.0, relevance)
}

func (s *Scorer) buildExplanation(c Candidate, userCtx *UserContext) string {
	// Use pre-built explanation if available
	if copy, ok := c.Metadata["recommendation_copy"].(string); ok && copy != "" {
		return copy
	}
	
	switch c.Source {
	case AdjacentService:
		return "Frequently booked together with your selection"
	case CollaborativeFilter:
		return "Popular among users with similar preferences"
	case TrendingService:
		return "Trending in your area"
	case EventBasedSuggest:
		return "Recommended for your event"
	default:
		return "Recommended for you"
	}
}

// =============================================================================
// RANKING & DIVERSIFICATION
// =============================================================================

// Ranker sorts recommendations by score
type Ranker struct {
	config *Config
}

func NewRanker(config *Config) *Ranker {
	return &Ranker{config: config}
}

func (r *Ranker) Rank(recs []Recommendation) []Recommendation {
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].Score > recs[j].Score
	})
	return recs
}

// Diversifier ensures variety in recommendations
type Diversifier struct {
	config *Config
}

func NewDiversifier(config *Config) *Diversifier {
	return &Diversifier{config: config}
}

func (d *Diversifier) Diversify(recs []Recommendation, limit int, diversityFactor float64) []Recommendation {
	if len(recs) <= limit {
		return d.assignPositions(recs)
	}
	
	// Use Maximal Marginal Relevance (MMR) for diversification
	selected := make([]Recommendation, 0, limit)
	remaining := make([]Recommendation, len(recs))
	copy(remaining, recs)
	
	// Always add the top item
	selected = append(selected, remaining[0])
	remaining = remaining[1:]
	
	for len(selected) < limit && len(remaining) > 0 {
		bestIdx := 0
		bestMMR := -1.0
		
		for i, candidate := range remaining {
			// Calculate similarity to already selected
			maxSim := 0.0
			for _, sel := range selected {
				sim := d.calculateSimilarity(candidate, sel)
				if sim > maxSim {
					maxSim = sim
				}
			}
			
			// MMR = λ * Relevance - (1-λ) * MaxSimilarity
			mmr := diversityFactor*candidate.Score - (1-diversityFactor)*maxSim
			
			if mmr > bestMMR {
				bestMMR = mmr
				bestIdx = i
			}
		}
		
		selected = append(selected, remaining[bestIdx])
		remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
	}
	
	return d.assignPositions(selected)
}

func (d *Diversifier) calculateSimilarity(a, b Recommendation) float64 {
	sim := 0.0
	
	// Same category = high similarity
	if aCat, ok := a.Metadata["category_id"].(uuid.UUID); ok {
		if bCat, ok := b.Metadata["category_id"].(uuid.UUID); ok {
			if aCat == bCat {
				sim += 0.5
			}
		}
	}
	
	// Same source type = some similarity
	if a.Type == b.Type {
		sim += 0.3
	}
	
	return sim
}

func (d *Diversifier) assignPositions(recs []Recommendation) []Recommendation {
	for i := range recs {
		recs[i].Position = i + 1
		recs[i].DiversityScore = 1.0 - float64(i)/float64(len(recs))
	}
	return recs
}

// =============================================================================
// SUPPORTING COMPONENTS
// =============================================================================

// EventDetector detects life events from user behavior
type EventDetector struct {
	db *pgxpool.Pool
}

func NewEventDetector(db *pgxpool.Pool) *EventDetector {
	return &EventDetector{db: db}
}

// TrendingService tracks trending items
type TrendingService struct {
	db    *pgxpool.Pool
	cache *redis.Client
}

type TrendingItem struct {
	ServiceID      uuid.UUID
	CategoryID     uuid.UUID
	TrendScore     float64
	ViewCount7D    int
	BookingCount7D int
	GrowthRate     float64
}

func NewTrendingService(db *pgxpool.Pool, cache *redis.Client) *TrendingService {
	return &TrendingService{db: db, cache: cache}
}

func (t *TrendingService) GetTrending(ctx context.Context, loc *GeoPoint, limit int) []TrendingItem {
	query := `
		WITH recent_activity AS (
			SELECT 
				ui.entity_id as service_id,
				COUNT(CASE WHEN ui.interaction_type = 'view' THEN 1 END) as views,
				COUNT(CASE WHEN ui.interaction_type = 'book' THEN 1 END) as bookings
			FROM user_interactions ui
			WHERE ui.entity_type = 'service'
			  AND ui.created_at > NOW() - INTERVAL '7 days'
			GROUP BY ui.entity_id
		),
		prev_activity AS (
			SELECT 
				ui.entity_id as service_id,
				COUNT(*) as prev_interactions
			FROM user_interactions ui
			WHERE ui.entity_type = 'service'
			  AND ui.created_at BETWEEN NOW() - INTERVAL '14 days' AND NOW() - INTERVAL '7 days'
			GROUP BY ui.entity_id
		)
		SELECT 
			s.id,
			s.category_id,
			ra.views,
			ra.bookings,
			CASE WHEN COALESCE(pa.prev_interactions, 0) = 0 THEN 1.0
			     ELSE (ra.views + ra.bookings * 5.0) / pa.prev_interactions
			END as growth_rate
		FROM services s
		JOIN recent_activity ra ON ra.service_id = s.id
		LEFT JOIN prev_activity pa ON pa.service_id = s.id
		WHERE s.is_available = TRUE
		ORDER BY (ra.bookings * 5 + ra.views) * 
		         CASE WHEN COALESCE(pa.prev_interactions, 0) = 0 THEN 2.0
		              ELSE (ra.views + ra.bookings * 5.0) / pa.prev_interactions
		         END DESC
		LIMIT $1
	`
	
	rows, err := t.db.Query(ctx, query, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	
	var items []TrendingItem
	maxScore := 0.0
	for rows.Next() {
		var item TrendingItem
		if err := rows.Scan(&item.ServiceID, &item.CategoryID, 
			&item.ViewCount7D, &item.BookingCount7D, &item.GrowthRate); err != nil {
			continue
		}
		rawScore := float64(item.BookingCount7D*5+item.ViewCount7D) * item.GrowthRate
		if rawScore > maxScore {
			maxScore = rawScore
		}
		items = append(items, item)
	}
	
	// Normalize scores
	for i := range items {
		rawScore := float64(items[i].BookingCount7D*5+items[i].ViewCount7D) * items[i].GrowthRate
		items[i].TrendScore = rawScore / maxScore
	}
	
	return items
}

// =============================================================================
// ENGINE HELPER METHODS
// =============================================================================

func (e *Engine) validateRequest(req *RecommendationRequest) error {
	if req.Limit < 0 || req.Limit > 100 {
		return fmt.Errorf("limit must be between 0 and 100")
	}
	if req.DiversityFactor < 0 || req.DiversityFactor > 1 {
		return fmt.Errorf("diversity factor must be between 0 and 1")
	}
	return nil
}

func (e *Engine) buildUserContext(ctx context.Context, req *RecommendationRequest) (*UserContext, error) {
	return e.userProfiler.BuildContext(ctx, req.UserID, req.SessionID)
}

func (e *Engine) selectGenerators(req *RecommendationRequest) []CandidateGenerator {
	generators := []CandidateGenerator{
		&AdjacencyGenerator{graph: e.adjacencyGraph, db: e.db},
		&EventBasedGenerator{db: e.db, eventDetector: e.eventDetector},
		&CollaborativeGenerator{db: e.db, cache: e.cache},
		&TrendingGenerator{service: e.trendingService},
	}
	
	// Could filter based on req.RequestedTypes
	return generators
}

func (e *Engine) deduplicateCandidates(candidates []Candidate) []Candidate {
	seen := make(map[uuid.UUID]bool)
	result := make([]Candidate, 0, len(candidates))
	
	for _, c := range candidates {
		if !seen[c.EntityID] {
			seen[c.EntityID] = true
			result = append(result, c)
		}
	}
	
	return result
}

func (e *Engine) logRecommendations(ctx context.Context, req *RecommendationRequest, resp *RecommendationResponse) {
	// Insert recommendation events for analytics
	for _, rec := range resp.Recommendations {
		_, _ = e.db.Exec(ctx, `
			INSERT INTO recommendation_events 
			(user_id, session_id, recommendation_type, algorithm_version,
			 recommended_entity_type, recommended_entity_id,
			 source_entity_type, source_entity_id, position, total_recommendations,
			 relevance_score, diversity_score, experiment_id, variant)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		`,
			req.UserID, req.SessionID, rec.Type, resp.AlgorithmVersion,
			rec.EntityType, rec.EntityID,
			req.CurrentEntityType, req.CurrentEntityID, rec.Position, len(resp.Recommendations),
			rec.RelevanceScore, rec.DiversityScore, resp.ExperimentID, resp.Variant,
		)
	}
}

func (e *Engine) backgroundRefresh() {
	ticker := time.NewTicker(e.config.AdjacencyRefreshRate)
	defer ticker.Stop()
	
	for range ticker.C {
		ctx := context.Background()
		if err := e.adjacencyGraph.Load(ctx); err != nil {
			// Log error
			continue
		}
	}
}
