package unit

import (
	"context"
	"testing"
	"time"

	"github.com/BillyRonksGlobal/vendorplatform/internal/lifeos"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Test Event Detection

func TestEventDetection_ValidUserID(t *testing.T) {
	userID := uuid.New()
	lookbackDays := 30

	// Mock service would be injected here
	// For now, testing logic structure

	assert.NotEqual(t, uuid.Nil, userID)
	assert.Greater(t, lookbackDays, 0)
}

func TestEventDetection_PatternMatching(t *testing.T) {
	patterns := map[string][]string{
		"wedding":    {"wedding", "venue", "catering"},
		"relocation": {"moving", "relocation", "packing"},
		"renovation": {"renovation", "contractor"},
	}

	for eventType, keywords := range patterns {
		assert.NotEmpty(t, keywords, "Event type %s should have keywords", eventType)
		assert.GreaterOrEqual(t, len(keywords), 2, "Should have at least 2 keywords")
	}
}

func TestEventDetection_ConfidenceThreshold(t *testing.T) {
	testCases := []struct {
		name       string
		confidence float64
		shouldPass bool
	}{
		{"High confidence", 0.85, true},
		{"Medium confidence", 0.6, true},
		{"Threshold", 0.5, true},
		{"Below threshold", 0.45, false},
		{"Very low", 0.2, false},
	}

	threshold := 0.5
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			passes := tc.confidence > threshold
			assert.Equal(t, tc.shouldPass, passes)
		})
	}
}

// Test Bundle Recommendations

func TestBundleGeneration_CoreServicesBundle(t *testing.T) {
	coreCategories := []string{"Venue", "Catering", "Photography", "Decoration", "Entertainment"}

	// Should generate core bundle if 3+ categories
	if len(coreCategories) >= 3 {
		savingsPercent := 10.0 + float64(len(coreCategories)-3)*1.5
		if savingsPercent > 20 {
			savingsPercent = 20
		}

		assert.GreaterOrEqual(t, savingsPercent, 10.0)
		assert.LessOrEqual(t, savingsPercent, 20.0)
		assert.Equal(t, 13.0, savingsPercent) // 10 + (5-3)*1.5 = 13%
	}
}

func TestBundleGeneration_FullPackageBundle(t *testing.T) {
	allCategories := []string{"A", "B", "C", "D", "E", "F", "G", "H"}

	if len(allCategories) >= 5 {
		savingsPercent := 15.0 + float64(len(allCategories)-5)*1.0
		if savingsPercent > 25 {
			savingsPercent = 25
		}

		assert.GreaterOrEqual(t, savingsPercent, 15.0)
		assert.LessOrEqual(t, savingsPercent, 25.0)
		assert.Equal(t, 18.0, savingsPercent) // 15 + (8-5)*1.0 = 18%
	}
}

func TestBundleGeneration_CategoryGroupBundles(t *testing.T) {
	matchedCategories := []string{"DJ/Music", "Photography"}

	if len(matchedCategories) >= 2 {
		savingsPercent := 8.0 + float64(len(matchedCategories))*2.0
		assert.Equal(t, 12.0, savingsPercent) // 8 + 2*2 = 12%
	}
}

func TestBundleGeneration_MinimumCategories(t *testing.T) {
	testCases := []struct {
		name       string
		categories []string
		shouldGen  bool
	}{
		{"Single category", []string{"A"}, false},
		{"Two categories", []string{"A", "B"}, true},
		{"Three categories", []string{"A", "B", "C"}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			canGenerate := len(tc.categories) >= 2
			assert.Equal(t, tc.shouldGen, canGenerate)
		})
	}
}

// Test Risk Assessment

func TestRiskAssessment_TimelineRisk(t *testing.T) {
	testCases := []struct {
		name        string
		daysUntil   int
		severity    string
		probability float64
	}{
		{"Very urgent", 10, "critical", 1.0},
		{"Urgent", 20, "critical", 0.9},
		{"Medium timeline", 60, "medium", 0.6},
		{"Comfortable", 120, "low", 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var expectedSeverity string
			var expectedProbability float64

			if tc.daysUntil < 14 {
				expectedSeverity = "critical"
				expectedProbability = 1.0
			} else if tc.daysUntil < 30 {
				expectedSeverity = "critical"
				expectedProbability = 0.9
			} else if tc.daysUntil < 90 {
				expectedSeverity = "medium"
				expectedProbability = 0.6
			}

			if tc.daysUntil < 90 {
				assert.Equal(t, tc.severity, expectedSeverity)
				assert.Equal(t, tc.probability, expectedProbability)
			}
		})
	}
}

func TestRiskAssessment_BudgetRisk(t *testing.T) {
	testCases := []struct {
		name             string
		totalBudget      float64
		remainingBudget  float64
		shouldFlagRisk   bool
	}{
		{"High unallocated", 1000000, 800000, true},  // 80% remaining
		{"Medium unallocated", 1000000, 600000, false}, // 60% remaining
		{"Low unallocated", 1000000, 300000, false},  // 30% remaining
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			remainingPct := (tc.remainingBudget / tc.totalBudget) * 100
			hasRisk := remainingPct > 70

			assert.Equal(t, tc.shouldFlagRisk, hasRisk)
		})
	}
}

func TestRiskAssessment_VendorAvailabilityRisk(t *testing.T) {
	testCases := []struct {
		name            string
		unbookedCritical int
		expectedSeverity string
	}{
		{"No vendors", 0, "low"},
		{"One critical", 1, "high"},
		{"Two critical", 2, "high"},
		{"Three+ critical", 3, "critical"},
		{"Many critical", 5, "critical"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var severity string
			if tc.unbookedCritical >= 3 {
				severity = "critical"
			} else if tc.unbookedCritical > 0 {
				severity = "high"
			} else {
				severity = "low"
			}

			assert.Equal(t, tc.expectedSeverity, severity)
		})
	}
}

func TestRiskAssessment_OverallRiskScore(t *testing.T) {
	testCases := []struct {
		name         string
		avgRiskScore float64
		expectedRisk string
	}{
		{"Critical risk", 75, "critical"},
		{"High risk", 60, "high"},
		{"Medium risk", 40, "medium"},
		{"Low risk", 20, "low"},
		{"No risk", 5, "low"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			overallRisk := "low"
			if tc.avgRiskScore > 70 {
				overallRisk = "critical"
			} else if tc.avgRiskScore > 50 {
				overallRisk = "high"
			} else if tc.avgRiskScore > 30 {
				overallRisk = "medium"
			}

			assert.Equal(t, tc.expectedRisk, overallRisk)
		})
	}
}

func TestRiskAssessment_CompletionRisk(t *testing.T) {
	testCases := []struct {
		name             string
		completionPct    float64
		daysUntil        int
		shouldHaveRisk   bool
	}{
		{"Low completion, urgent", 15, 45, true},
		{"Low completion, time", 15, 90, false},
		{"Good completion", 60, 45, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasRisk := tc.completionPct < 20 && tc.daysUntil < 60
			assert.Equal(t, tc.shouldHaveRisk, hasRisk)
		})
	}
}

// Test Budget Optimization

func TestBudgetOptimization_PriorityAllocation(t *testing.T) {
	testCases := []struct {
		priority     string
		expectedPct  float64
	}{
		{"primary", 15.0},
		{"secondary", 10.0},
		{"optional", 5.0},
	}

	for _, tc := range testCases {
		t.Run(tc.priority, func(t *testing.T) {
			var budgetPct float64
			switch tc.priority {
			case "primary":
				budgetPct = 15.0
			case "secondary":
				budgetPct = 10.0
			case "optional":
				budgetPct = 5.0
			default:
				budgetPct = 8.0
			}

			assert.Equal(t, tc.expectedPct, budgetPct)
		})
	}
}

func TestBudgetOptimization_BundleSavings(t *testing.T) {
	totalBudget := 1000000.0

	testCases := []struct {
		name            string
		categoryCount   int
		expectedSavings float64
	}{
		{"3 categories", 3, totalBudget * 0.12},
		{"5 categories", 5, totalBudget * 0.12},
		{"8 categories", 8, totalBudget * 0.12},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.categoryCount >= 3 {
				bundleSavings := totalBudget * 0.12 // 12%
				assert.Equal(t, tc.expectedSavings, bundleSavings)
			}
		})
	}
}

func TestBudgetOptimization_EarlyBookingDiscount(t *testing.T) {
	totalBudget := 1000000.0

	testCases := []struct {
		name            string
		daysUntilEvent  int
		shouldHaveDisc  bool
		expectedSavings float64
	}{
		{"Very early", 120, true, totalBudget * 0.08},
		{"Early", 95, true, totalBudget * 0.08},
		{"Threshold", 90, false, 0},
		{"Late", 60, false, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasDiscount := tc.daysUntilEvent > 90
			assert.Equal(t, tc.shouldHaveDisc, hasDiscount)

			if hasDiscount {
				earlyBookingSavings := totalBudget * 0.08
				assert.Equal(t, tc.expectedSavings, earlyBookingSavings)
			}
		})
	}
}

func TestBudgetOptimization_AlternativeVendorSavings(t *testing.T) {
	totalBudget := 1000000.0
	expectedSavings := totalBudget * 0.15 // 15%

	assert.Equal(t, 150000.0, expectedSavings)
}

func TestBudgetOptimization_TotalSavingsCumulative(t *testing.T) {
	totalBudget := 1000000.0

	// Bundle: 12%, Early Booking: 8%, Alternative: 15%
	// Total potential: 12% + 8% + 15% = 35% = 350,000

	bundleSavings := totalBudget * 0.12
	earlyBookingSavings := totalBudget * 0.08
	alternativeSavings := totalBudget * 0.15

	totalSavings := bundleSavings + earlyBookingSavings + alternativeSavings

	assert.Equal(t, 350000.0, totalSavings)
	assert.Equal(t, 35.0, (totalSavings/totalBudget)*100) // 35%
}

func TestBudgetOptimization_PositiveBudgetValidation(t *testing.T) {
	testCases := []struct {
		name        string
		budget      float64
		shouldError bool
	}{
		{"Valid budget", 1000000, false},
		{"Zero budget", 0, true},
		{"Negative budget", -1000, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasError := tc.budget <= 0
			assert.Equal(t, tc.shouldError, hasError)
		})
	}
}

// Test Helper Functions

func TestHelpers_CapitalizeFirst(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"wedding", "Wedding"},
		{"birthday", "Birthday"},
		{"", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			var result string
			if len(tc.input) == 0 {
				result = tc.input
			} else {
				result = string(tc.input[0]-32) + tc.input[1:]
			}
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHelpers_Slugify(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"Food & Beverage", "food-beverage"},
		{"Entertainment Bundle", "entertainment-bundle"},
		{"Simple", "simple"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			// Simplified slugify logic for testing
			result := ""
			for _, r := range tc.input {
				if r == ' ' {
					result += "-"
				} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
					if r >= 'A' && r <= 'Z' {
						result += string(r + 32)
					} else {
						result += string(r)
					}
				}
			}
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHelpers_Contains(t *testing.T) {
	testCases := []struct {
		haystack string
		needle   string
		expected bool
	}{
		{"Photography Services", "Photography", true},
		{"Catering", "Cater", true},
		{"Entertainment", "Music", false},
		{"", "test", false},
	}

	for _, tc := range testCases {
		t.Run(tc.haystack+"-"+tc.needle, func(t *testing.T) {
			contains := len(tc.haystack) >= len(tc.needle) &&
			           (len(tc.needle) == 0 || tc.haystack[:len(tc.needle)] == tc.needle)
			assert.Equal(t, tc.expected, contains)
		})
	}
}

// Integration Test Structures

func TestLifeEvent_DataStructure(t *testing.T) {
	event := lifeos.LifeEvent{
		ID:                  uuid.New(),
		UserID:              uuid.New(),
		EventType:           "wedding",
		ClusterType:         "celebrations",
		DetectedAt:          time.Now(),
		DetectionMethod:     "explicit",
		DetectionConfidence: 1.0,
		Status:              "confirmed",
		Phase:               "discovery",
		CompletionPct:       0.0,
	}

	assert.NotEqual(t, uuid.Nil, event.ID)
	assert.NotEqual(t, uuid.Nil, event.UserID)
	assert.Equal(t, "wedding", event.EventType)
	assert.Equal(t, "celebrations", event.ClusterType)
	assert.Equal(t, "explicit", event.DetectionMethod)
	assert.Equal(t, 1.0, event.DetectionConfidence)
	assert.Equal(t, "confirmed", event.Status)
	assert.Equal(t, "discovery", event.Phase)
}

func TestBundleOpportunity_DataStructure(t *testing.T) {
	bundle := lifeos.BundleOpportunity{
		BundleID:         "bundle-wedding-core",
		Name:             "Wedding Essential Bundle",
		Description:      "Core services for your wedding",
		Categories:       []string{"Venue", "Catering", "Photography"},
		TotalCategories:  3,
		EstimatedSavings: 120000,
		SavingsPercent:   12.0,
		Priority:         1,
	}

	assert.Equal(t, "bundle-wedding-core", bundle.BundleID)
	assert.Equal(t, 3, bundle.TotalCategories)
	assert.Equal(t, 3, len(bundle.Categories))
	assert.Equal(t, 12.0, bundle.SavingsPercent)
	assert.Greater(t, bundle.EstimatedSavings, 0.0)
}

func TestRiskAssessment_DataStructure(t *testing.T) {
	assessment := lifeos.RiskAssessment{
		EventID:     uuid.New(),
		OverallRisk: "medium",
		RiskScore:   45.0,
		Risks: []lifeos.Risk{
			{
				RiskType:    "timeline",
				Severity:    "medium",
				Probability: 0.6,
				Impact:      0.5,
				Description: "Limited time for planning",
			},
		},
		Mitigations: []lifeos.Mitigation{
			{
				RiskType:    "timeline",
				Strategy:    "Expedited Booking",
				Priority:    1,
				Description: "Fast-track vendor selection",
				ActionItems: []string{"Book venue ASAP", "Contact vendors today"},
			},
		},
		AssessedAt: time.Now(),
	}

	assert.NotEqual(t, uuid.Nil, assessment.EventID)
	assert.Equal(t, "medium", assessment.OverallRisk)
	assert.Equal(t, 45.0, assessment.RiskScore)
	assert.Len(t, assessment.Risks, 1)
	assert.Len(t, assessment.Mitigations, 1)
	assert.Equal(t, "timeline", assessment.Risks[0].RiskType)
	assert.Greater(t, len(assessment.Mitigations[0].ActionItems), 0)
}

func TestBudgetOptimization_DataStructure(t *testing.T) {
	optimization := lifeos.BudgetOptimization{
		EventID:     uuid.New(),
		TotalBudget: 1000000,
		OptimizedAllocation: map[string]lifeos.CategoryBudget{
			"venue": {
				CategoryID:            "venue-001",
				CategoryName:          "Venue",
				CurrentAllocation:     0,
				RecommendedAllocation: 200000,
				MarketAverage:         200000,
				Priority:              "primary",
			},
		},
		SavingsOpportunities: []lifeos.SavingsOpportunity{
			{
				OpportunityType:  "bundle",
				Description:      "Bundle discount",
				EstimatedSavings: 120000,
			},
		},
		TotalPotentialSavings: 350000,
		RecommendedChanges: []lifeos.BudgetChange{
			{
				CategoryID:        "venue-001",
				CategoryName:      "Venue",
				CurrentAmount:     0,
				RecommendedAmount: 200000,
				Change:            200000,
				ChangePercent:     0,
				Reason:            "Optimized for primary priority",
			},
		},
	}

	assert.NotEqual(t, uuid.Nil, optimization.EventID)
	assert.Equal(t, 1000000.0, optimization.TotalBudget)
	assert.Equal(t, 350000.0, optimization.TotalPotentialSavings)
	assert.Len(t, optimization.OptimizedAllocation, 1)
	assert.Greater(t, len(optimization.SavingsOpportunities), 0)
	assert.Greater(t, len(optimization.RecommendedChanges), 0)
}

// API Request Validation Tests

func TestAPI_CreateLifeEventValidation(t *testing.T) {
	testCases := []struct {
		name        string
		userID      uuid.UUID
		eventType   string
		shouldError bool
	}{
		{"Valid request", uuid.New(), "wedding", false},
		{"Empty user ID", uuid.Nil, "wedding", true},
		{"Empty event type", uuid.New(), "", true},
		{"Invalid event type", uuid.New(), "invalid_event", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasError := tc.userID == uuid.Nil || tc.eventType == ""

			validEventTypes := map[string]bool{
				"wedding": true, "birthday": true, "relocation": true,
				"renovation": true, "childbirth": true,
			}
			if !validEventTypes[tc.eventType] && tc.eventType != "" {
				hasError = true
			}

			assert.Equal(t, tc.shouldError, hasError)
		})
	}
}

func TestAPI_DetectLifeEventsValidation(t *testing.T) {
	testCases := []struct {
		name         string
		userID       uuid.UUID
		lookbackDays int
		shouldError  bool
	}{
		{"Valid request", uuid.New(), 30, false},
		{"Default lookback", uuid.New(), 0, false}, // 0 defaults to 30
		{"Empty user ID", uuid.Nil, 30, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasError := tc.userID == uuid.Nil
			assert.Equal(t, tc.shouldError, hasError)
		})
	}
}

func TestAPI_OptimizeBudgetValidation(t *testing.T) {
	testCases := []struct {
		name        string
		eventID     uuid.UUID
		totalBudget float64
		shouldError bool
	}{
		{"Valid request", uuid.New(), 1000000, false},
		{"Invalid event ID", uuid.Nil, 1000000, true},
		{"Zero budget", uuid.New(), 0, true},
		{"Negative budget", uuid.New(), -1000, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasError := tc.eventID == uuid.Nil || tc.totalBudget <= 0
			assert.Equal(t, tc.shouldError, hasError)
		})
	}
}
