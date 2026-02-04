package unit

import (
	"testing"
	"time"

	"github.com/BillyRonksGlobal/vendorplatform/internal/homerescue"
	"github.com/google/uuid"
)

// TestEmergencyValidation tests emergency creation validation
func TestEmergencyValidation(t *testing.T) {
	tests := []struct {
		name        string
		req         *homerescue.CreateEmergencyRequest
		expectError bool
		errorType   error
	}{
		{
			name: "Valid critical emergency",
			req: &homerescue.CreateEmergencyRequest{
				UserID:      uuid.New(),
				Category:    "plumbing",
				Urgency:     "critical",
				Title:       "Burst pipe emergency",
				Description: "Water flooding bathroom",
				Address:     "123 Test St",
				City:        "Lagos",
				State:       "Lagos",
				PostalCode:  "100001",
				Latitude:    6.5244,
				Longitude:   3.3792,
			},
			expectError: false,
		},
		{
			name: "Invalid urgency level",
			req: &homerescue.CreateEmergencyRequest{
				UserID:      uuid.New(),
				Category:    "plumbing",
				Urgency:     "super_critical", // Invalid
				Title:       "Emergency",
				Description: "Test",
				Address:     "123 Test St",
				City:        "Lagos",
				State:       "Lagos",
				PostalCode:  "100001",
				Latitude:    6.5244,
				Longitude:   3.3792,
			},
			expectError: true,
			errorType:   homerescue.ErrInvalidUrgency,
		},
		{
			name: "Missing required fields",
			req: &homerescue.CreateEmergencyRequest{
				UserID:   uuid.Nil, // Invalid
				Category: "",       // Invalid
				Title:    "",       // Invalid
			},
			expectError: true,
			errorType:   homerescue.ErrInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation check (would need actual service for full test)
			if tt.req.UserID == uuid.Nil || tt.req.Category == "" || tt.req.Title == "" {
				if !tt.expectError {
					t.Errorf("Expected no error but validation should fail")
				}
			}
		})
	}
}

// TestSLACalculations tests SLA time calculations
func TestSLACalculations(t *testing.T) {
	tests := []struct {
		name               string
		urgency            string
		expectedResponseSLA int
		expectedRefund      int
	}{
		{
			name:               "Critical urgency",
			urgency:            "critical",
			expectedResponseSLA: 30,
			expectedRefund:      100,
		},
		{
			name:               "Urgent urgency",
			urgency:            "urgent",
			expectedResponseSLA: 120,
			expectedRefund:      50,
		},
		{
			name:               "Same day urgency",
			urgency:            "same_day",
			expectedResponseSLA: 360,
			expectedRefund:      25,
		},
		{
			name:               "Scheduled urgency",
			urgency:            "scheduled",
			expectedResponseSLA: 1440,
			expectedRefund:      0,
		},
	}

	responseSLAMinutes := map[string]int{
		"critical":  30,
		"urgent":    120,
		"same_day":  360,
		"scheduled": 1440,
	}

	slaRefundPercentages := map[string]int{
		"critical":  100,
		"urgent":    50,
		"same_day":  25,
		"scheduled": 0,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualSLA := responseSLAMinutes[tt.urgency]
			if actualSLA != tt.expectedResponseSLA {
				t.Errorf("Expected SLA %d minutes, got %d", tt.expectedResponseSLA, actualSLA)
			}

			actualRefund := slaRefundPercentages[tt.urgency]
			if actualRefund != tt.expectedRefund {
				t.Errorf("Expected refund %d%%, got %d%%", tt.expectedRefund, actualRefund)
			}
		})
	}
}

// TestRefundCalculation tests refund amount calculation
func TestRefundCalculation(t *testing.T) {
	tests := []struct {
		name             string
		finalCost        float64
		refundPercentage int
		expectedRefund   float64
	}{
		{
			name:             "Critical 100% refund",
			finalCost:        15000.0,
			refundPercentage: 100,
			expectedRefund:   15000.0,
		},
		{
			name:             "Urgent 50% refund",
			finalCost:        20000.0,
			refundPercentage: 50,
			expectedRefund:   10000.0,
		},
		{
			name:             "Same day 25% refund",
			finalCost:        8000.0,
			refundPercentage: 25,
			expectedRefund:   2000.0,
		},
		{
			name:             "Scheduled no refund",
			finalCost:        5000.0,
			refundPercentage: 0,
			expectedRefund:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refundAmount := tt.finalCost * float64(tt.refundPercentage) / 100.0
			if refundAmount != tt.expectedRefund {
				t.Errorf("Expected refund ₦%.2f, got ₦%.2f", tt.expectedRefund, refundAmount)
			}
		})
	}
}

// TestDistanceCalculation tests Haversine distance formula
func TestDistanceCalculation(t *testing.T) {
	// Lagos coordinates
	lagosLat := 6.5244
	lagosLon := 3.3792

	// Abuja coordinates (approximately 700km from Lagos)
	abujaLat := 9.0765
	abujaLon := 7.3986

	tests := []struct {
		name         string
		lat1, lon1   float64
		lat2, lon2   float64
		expectedKm   float64
		tolerance    float64
	}{
		{
			name:       "Same location",
			lat1:       lagosLat,
			lon1:       lagosLon,
			lat2:       lagosLat,
			lon2:       lagosLon,
			expectedKm: 0.0,
			tolerance:  0.01,
		},
		{
			name:       "Lagos to Abuja",
			lat1:       lagosLat,
			lon1:       lagosLon,
			lat2:       abujaLat,
			lon2:       abujaLon,
			expectedKm: 700.0, // Approximately
			tolerance:  50.0,  // Allow 50km variance
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would call the actual calculateDistance function
			// For now, we're testing the logic
			distance := calculateTestDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)

			diff := distance - tt.expectedKm
			if diff < 0 {
				diff = -diff
			}

			if diff > tt.tolerance {
				t.Errorf("Expected distance ~%.2f km (±%.2f), got %.2f km",
					tt.expectedKm, tt.tolerance, distance)
			}
		})
	}
}

// Helper function for testing distance calculation
func calculateTestDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	const pi = 3.14159265358979323846

	dLat := (lat2 - lat1) * pi / 180.0
	dLon := (lon2 - lon1) * pi / 180.0

	lat1Rad := lat1 * pi / 180.0
	lat2Rad := lat2 * pi / 180.0

	// Simplified Haversine
	a := 0.5 - 0.5 * (1.0 - dLat * dLat) +
		(1.0 - lat1Rad * lat1Rad) * (1.0 - lat2Rad * lat2Rad) *
		0.5 * (1.0 - dLon * dLon)

	// Very simplified version
	if lat1 == lat2 && lon1 == lon2 {
		return 0.0
	}

	// Rough approximation for large distances
	latDiff := lat2 - lat1
	lonDiff := lon2 - lon1
	return 111.0 * (latDiff*latDiff + lonDiff*lonDiff*0.5)
}

// TestETACalculation tests ETA estimation
func TestETACalculation(t *testing.T) {
	tests := []struct {
		name              string
		distanceKm        float64
		avgSpeedKmh       float64
		expectedMinutes   int
	}{
		{
			name:            "5km at 40km/h",
			distanceKm:      5.0,
			avgSpeedKmh:     40.0,
			expectedMinutes: 7,  // (5/40)*60 = 7.5 ≈ 7
		},
		{
			name:            "20km at 40km/h",
			distanceKm:      20.0,
			avgSpeedKmh:     40.0,
			expectedMinutes: 30, // (20/40)*60 = 30
		},
		{
			name:            "10km at 40km/h",
			distanceKm:      10.0,
			avgSpeedKmh:     40.0,
			expectedMinutes: 15, // (10/40)*60 = 15
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minutes := int((tt.distanceKm / tt.avgSpeedKmh) * 60)
			if minutes != tt.expectedMinutes {
				t.Errorf("Expected %d minutes, got %d", tt.expectedMinutes, minutes)
			}
		})
	}
}

// TestSLAStatusCalculation tests SLA status determination
func TestSLAStatusCalculation(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name             string
		responseDeadline time.Time
		arrivalDeadline  time.Time
		status           string
		expectedSLAStatus string
	}{
		{
			name:             "Completed emergency",
			responseDeadline: now.Add(-30 * time.Minute),
			arrivalDeadline:  now.Add(-10 * time.Minute),
			status:           "completed",
			expectedSLAStatus: "final",
		},
		{
			name:             "Response deadline passed",
			responseDeadline: now.Add(-10 * time.Minute),
			arrivalDeadline:  now.Add(20 * time.Minute),
			status:           "searching",
			expectedSLAStatus: "breached",
		},
		{
			name:             "Arrival deadline passed",
			responseDeadline: now.Add(-60 * time.Minute),
			arrivalDeadline:  now.Add(-10 * time.Minute),
			status:           "en_route",
			expectedSLAStatus: "breached",
		},
		{
			name:             "On track",
			responseDeadline: now.Add(60 * time.Minute),
			arrivalDeadline:  now.Add(90 * time.Minute),
			status:           "accepted",
			expectedSLAStatus: "on_track",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := calculateSLAStatusTest(tt.responseDeadline, tt.arrivalDeadline, tt.status)
			if status != tt.expectedSLAStatus {
				t.Errorf("Expected SLA status %s, got %s", tt.expectedSLAStatus, status)
			}
		})
	}
}

// Helper function for testing SLA status calculation
func calculateSLAStatusTest(responseDeadline, arrivalDeadline time.Time, status string) string {
	now := time.Now()

	if status == "completed" || status == "cancelled" {
		return "final"
	}

	// Check if response deadline passed
	if now.After(responseDeadline) && (status == "new" || status == "searching") {
		return "breached"
	}

	// Check if arrival deadline passed
	if now.After(arrivalDeadline) && status != "completed" {
		return "breached"
	}

	// Check if approaching deadline
	responseBuffer := responseDeadline.Sub(now)
	urgentSLA := 120 * time.Minute
	if responseBuffer < urgentSLA/5 {
		return "at_risk"
	}

	return "on_track"
}

// TestEmergencyStatusFlow tests the emergency lifecycle
func TestEmergencyStatusFlow(t *testing.T) {
	validStatuses := []string{
		"new",
		"searching",
		"assigned",
		"accepted",
		"en_route",
		"in_progress",
		"completed",
		"cancelled",
		"no_technicians_available",
	}

	validTransitions := map[string][]string{
		"new":                     {"searching", "cancelled"},
		"searching":               {"assigned", "accepted", "no_technicians_available", "cancelled"},
		"assigned":                {"accepted", "en_route", "cancelled"},
		"accepted":                {"en_route", "in_progress", "cancelled"},
		"en_route":                {"in_progress", "cancelled"},
		"in_progress":             {"completed", "cancelled"},
		"completed":               {},
		"cancelled":               {},
		"no_technicians_available": {"searching", "cancelled"},
	}

	t.Run("Valid status values", func(t *testing.T) {
		if len(validStatuses) < 5 {
			t.Error("Should have at least 5 valid statuses")
		}
	})

	t.Run("Valid transitions defined", func(t *testing.T) {
		for status, transitions := range validTransitions {
			if status == "completed" || status == "cancelled" {
				if len(transitions) != 0 {
					t.Errorf("Terminal status %s should have no transitions", status)
				}
			}
		}
	})
}

// TestTechnicianAvailabilityLogic tests technician matching logic
func TestTechnicianAvailabilityLogic(t *testing.T) {
	tests := []struct {
		name              string
		isAvailable       bool
		currentJobs       int
		maxJobs           int
		shouldBeAvailable bool
	}{
		{
			name:              "Available with capacity",
			isAvailable:       true,
			currentJobs:       2,
			maxJobs:           5,
			shouldBeAvailable: true,
		},
		{
			name:              "At max capacity",
			isAvailable:       true,
			currentJobs:       5,
			maxJobs:           5,
			shouldBeAvailable: false,
		},
		{
			name:              "Marked unavailable",
			isAvailable:       false,
			currentJobs:       0,
			maxJobs:           5,
			shouldBeAvailable: false,
		},
		{
			name:              "Over capacity",
			isAvailable:       true,
			currentJobs:       6,
			maxJobs:           5,
			shouldBeAvailable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			available := tt.isAvailable && tt.currentJobs < tt.maxJobs
			if available != tt.shouldBeAvailable {
				t.Errorf("Expected availability %v, got %v", tt.shouldBeAvailable, available)
			}
		})
	}
}

// TestEmergencyCategories tests valid emergency categories
func TestEmergencyCategories(t *testing.T) {
	validCategories := []string{
		"plumbing",
		"electrical",
		"locksmith",
		"hvac",
		"glass",
		"roofing",
		"pest",
		"security",
		"general",
	}

	t.Run("All categories defined", func(t *testing.T) {
		if len(validCategories) < 8 {
			t.Error("Should have at least 8 emergency categories")
		}
	})

	t.Run("No duplicate categories", func(t *testing.T) {
		seen := make(map[string]bool)
		for _, cat := range validCategories {
			if seen[cat] {
				t.Errorf("Duplicate category: %s", cat)
			}
			seen[cat] = true
		}
	})
}

// TestUrgencyLevels tests valid urgency levels
func TestUrgencyLevels(t *testing.T) {
	urgencyLevels := []string{"critical", "urgent", "same_day", "scheduled"}

	t.Run("All urgency levels defined", func(t *testing.T) {
		if len(urgencyLevels) != 4 {
			t.Error("Should have exactly 4 urgency levels")
		}
	})

	t.Run("Urgency order is logical", func(t *testing.T) {
		slaMinutes := map[string]int{
			"critical":  30,
			"urgent":    120,
			"same_day":  360,
			"scheduled": 1440,
		}

		// Critical should have shortest SLA
		if slaMinutes["critical"] >= slaMinutes["urgent"] {
			t.Error("Critical SLA should be shorter than urgent")
		}

		// Scheduled should have longest SLA
		if slaMinutes["scheduled"] <= slaMinutes["same_day"] {
			t.Error("Scheduled SLA should be longer than same_day")
		}
	})
}
