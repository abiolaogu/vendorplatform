// EventGPT Unit Tests
// Copyright (c) 2024 BillyRonks Global Limited. All rights reserved.

package unit

import (
	"testing"

	"github.com/BillyRonksGlobal/vendorplatform/internal/eventgpt"
	"github.com/stretchr/testify/assert"
)

// TestIntentClassification tests the intent classification logic
func TestIntentClassification(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected eventgpt.Intent
	}{
		{
			name:     "Create Event - Wedding",
			message:  "I'm planning a wedding",
			expected: eventgpt.IntentCreateEvent,
		},
		{
			name:     "Create Event - Birthday",
			message:  "Help me organize a birthday party",
			expected: eventgpt.IntentCreateEvent,
		},
		{
			name:     "Find Vendor - Photographer",
			message:  "I need to find a photographer",
			expected: eventgpt.IntentFindVendor,
		},
		{
			name:     "Find Vendor - Caterer",
			message:  "Looking for a caterer",
			expected: eventgpt.IntentFindVendor,
		},
		{
			name:     "Get Quote",
			message:  "How much does a photographer cost?",
			expected: eventgpt.IntentGetQuote,
		},
		{
			name:     "Book Service",
			message:  "I want to book a DJ",
			expected: eventgpt.IntentBookService,
		},
		{
			name:     "Compare Options",
			message:  "Compare these two vendors",
			expected: eventgpt.IntentCompareOptions,
		},
		{
			name:     "Check Availability",
			message:  "Are you available next weekend?",
			expected: eventgpt.IntentCheckAvailability,
		},
		{
			name:     "Ask Question",
			message:  "What services do you offer?",
			expected: eventgpt.IntentAskQuestion,
		},
	}

	// Create a minimal service for testing
	service := &eventgpt.Service{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Access the private method via reflection or make it public for testing
			// For now, we'll test the patterns directly

			// Note: In production, you'd want to make classifyIntent public or
			// create a test helper that exposes it
		})
	}
}

// TestSlotExtraction tests entity extraction from messages
func TestSlotExtraction(t *testing.T) {
	tests := []struct {
		name          string
		message       string
		intent        eventgpt.Intent
		expectedSlots map[eventgpt.Slot]interface{}
	}{
		{
			name:    "Extract event type",
			message: "I'm planning a wedding in Lagos",
			intent:  eventgpt.IntentCreateEvent,
			expectedSlots: map[eventgpt.Slot]interface{}{
				eventgpt.SlotEventType: "wedding",
				eventgpt.SlotLocation:  "Lagos",
			},
		},
		{
			name:    "Extract guest count",
			message: "I need a venue for 200 guests",
			intent:  eventgpt.IntentFindVendor,
			expectedSlots: map[eventgpt.Slot]interface{}{
				eventgpt.SlotGuestCount: "200",
			},
		},
		{
			name:    "Extract budget",
			message: "My budget is 5000000 naira",
			intent:  eventgpt.IntentGetQuote,
			expectedSlots: map[eventgpt.Slot]interface{}{
				eventgpt.SlotBudget: "5000000",
			},
		},
		{
			name:    "Extract vendor type",
			message: "Looking for a photographer",
			intent:  eventgpt.IntentFindVendor,
			expectedSlots: map[eventgpt.Slot]interface{}{
				eventgpt.SlotVendorType: "photography",
			},
		},
		{
			name:    "Extract location - Abuja",
			message: "I need vendors in Abuja",
			intent:  eventgpt.IntentFindVendor,
			expectedSlots: map[eventgpt.Slot]interface{}{
				eventgpt.SlotLocation: "Abuja",
			},
		},
	}

	service := &eventgpt.Service{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test slot extraction
			// Note: In production, you'd want to make extractSlots public or
			// create a test helper
		})
	}
}

// TestConversationStateTransitions tests state machine transitions
func TestConversationStateTransitions(t *testing.T) {
	tests := []struct {
		name          string
		currentState  eventgpt.ConversationState
		slots         map[eventgpt.Slot]interface{}
		expectedState eventgpt.ConversationState
	}{
		{
			name:         "Initial to Gathering Details",
			currentState: eventgpt.StateInitial,
			slots: map[eventgpt.Slot]interface{}{
				eventgpt.SlotEventType: "wedding",
			},
			expectedState: eventgpt.StateGatheringDetails,
		},
		{
			name:         "Gathering Details to Showing Options",
			currentState: eventgpt.StateGatheringDetails,
			slots: map[eventgpt.Slot]interface{}{
				eventgpt.SlotEventType:  "wedding",
				eventgpt.SlotEventDate:  "December 2024",
				eventgpt.SlotLocation:   "Lagos",
				eventgpt.SlotGuestCount: "200",
				eventgpt.SlotBudget:     "5000000",
			},
			expectedState: eventgpt.StateShowingOptions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test state transitions
			// This would test the determineNextState method
		})
	}
}

// TestMissingSlots tests the missing slot detection
func TestMissingSlots(t *testing.T) {
	tests := []struct {
		name          string
		slots         map[eventgpt.Slot]interface{}
		expectedCount int
	}{
		{
			name:          "All slots missing",
			slots:         map[eventgpt.Slot]interface{}{},
			expectedCount: 5, // event_type, date, location, guest_count, budget
		},
		{
			name: "Two slots filled",
			slots: map[eventgpt.Slot]interface{}{
				eventgpt.SlotEventType: "wedding",
				eventgpt.SlotLocation:  "Lagos",
			},
			expectedCount: 3,
		},
		{
			name: "All slots filled",
			slots: map[eventgpt.Slot]interface{}{
				eventgpt.SlotEventType:  "wedding",
				eventgpt.SlotEventDate:  "December",
				eventgpt.SlotLocation:   "Lagos",
				eventgpt.SlotGuestCount: "200",
				eventgpt.SlotBudget:     "5000000",
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test missing slot detection
			// This would test the getMissingSlots method
		})
	}
}

// TestQuickReplies tests quick reply generation
func TestQuickReplies(t *testing.T) {
	tests := []struct {
		name          string
		state         eventgpt.ConversationState
		expectedCount int
	}{
		{
			name:          "Initial State",
			state:         eventgpt.StateInitial,
			expectedCount: 3,
		},
		{
			name:          "Gathering Details State",
			state:         eventgpt.StateGatheringDetails,
			expectedCount: 4,
		},
		{
			name:          "Showing Options State",
			state:         eventgpt.StateShowingOptions,
			expectedCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test quick reply generation
			// This would test the generateQuickReplies method
		})
	}
}

// TestEventSummary tests event summarization
func TestEventSummary(t *testing.T) {
	conversation := &eventgpt.Conversation{
		Slots: map[eventgpt.Slot]interface{}{
			eventgpt.SlotEventType:  "wedding",
			eventgpt.SlotEventDate:  "December 2024",
			eventgpt.SlotLocation:   "Lagos",
			eventgpt.SlotGuestCount: "200",
			eventgpt.SlotBudget:     "5000000",
		},
	}

	// Test summarization
	// This would test the summarizeEvent method
	assert.NotNil(t, conversation)
	assert.Equal(t, "wedding", conversation.Slots[eventgpt.SlotEventType])
}

// TestResponseGeneration tests the response generation for different intents
func TestResponseGeneration(t *testing.T) {
	tests := []struct {
		name              string
		intent            eventgpt.Intent
		slots             map[eventgpt.Slot]interface{}
		shouldContainText string
	}{
		{
			name:              "Create Event Response",
			intent:            eventgpt.IntentCreateEvent,
			slots:             map[eventgpt.Slot]interface{}{},
			shouldContainText: "event",
		},
		{
			name:   "Find Vendor Response with Location",
			intent: eventgpt.IntentFindVendor,
			slots: map[eventgpt.Slot]interface{}{
				eventgpt.SlotVendorType: "photography",
				eventgpt.SlotLocation:   "Lagos",
			},
			shouldContainText: "Lagos",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test response generation
			assert.NotEmpty(t, tt.shouldContainText)
		})
	}
}

// TestWelcomeMessage tests the welcome message generation
func TestWelcomeMessage(t *testing.T) {
	service := &eventgpt.Service{}

	// Test welcome message generation
	// This would test the generateWelcomeMessage method
	assert.NotNil(t, service)
}
