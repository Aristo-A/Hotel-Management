package models

import (
	"math"
	"testing"
	"time"
)

// TestGSTCalculation verifies that subtotal, 12% GST, and balances are accurately computed
func TestGSTCalculation(t *testing.T) {
	nights := 3
	ratePerNight := 5500.00 // Deluxe room in INR

	subtotal := float64(nights) * ratePerNight
	if subtotal != 16500.00 {
		t.Fatalf("expected subtotal 16500.00, got: %f", subtotal)
	}

	taxRate := 0.12 // 12% Indian GST
	taxAmount := math.Round((subtotal*taxRate)*100) / 100
	if taxAmount != 1980.00 {
		t.Fatalf("expected taxAmount 1980.00 (12%% of 16500), got: %f", taxAmount)
	}

	discount := 500.00
	totalAmount := subtotal + taxAmount - discount
	if totalAmount != 17980.00 {
		t.Fatalf("expected totalAmount 17980.00, got: %f", totalAmount)
	}

	// Partial payment verification
	amountPaid := 10000.00
	balance := totalAmount - amountPaid
	if balance != 7980.00 {
		t.Fatalf("expected balance 7980.00, got: %f", balance)
	}
}

// TestDateOverlapLogic tests the mathematical overlap condition:
// checkInExisting < checkOutNew AND checkOutExisting > checkInNew
func TestDateOverlapLogic(t *testing.T) {
	// Existing reservation: Dec 1 to Dec 5 (Inclusive Dec 1, Exclusive Dec 5)
	existingIn := time.Date(2024, 12, 1, 14, 0, 0, 0, time.UTC)
	existingOut := time.Date(2024, 12, 5, 11, 0, 0, 0, time.UTC)

	isOverlap := func(newIn, newOut time.Time) bool {
		return existingIn.Before(newOut) && existingOut.After(newIn)
	}

	// Case 1: Overlapping in the middle (Dec 2 to Dec 4) -> MUST OVERLAP
	overlap1 := isOverlap(
		time.Date(2024, 12, 2, 14, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 4, 11, 0, 0, 0, time.UTC),
	)
	if !overlap1 {
		t.Errorf("expected overlap for Dec 2 - Dec 4 inside Dec 1 - Dec 5")
	}

	// Case 2: Overlapping start (Nov 28 to Dec 2) -> MUST OVERLAP
	overlap2 := isOverlap(
		time.Date(2024, 11, 28, 14, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 2, 11, 0, 0, 0, time.UTC),
	)
	if !overlap2 {
		t.Errorf("expected overlap for Nov 28 - Dec 2 overlapping Dec 1")
	}

	// Case 3: Completely before (Nov 25 to Nov 30) -> NO OVERLAP
	overlap3 := isOverlap(
		time.Date(2024, 11, 25, 14, 0, 0, 0, time.UTC),
		time.Date(2024, 11, 30, 11, 0, 0, 0, time.UTC),
	)
	if overlap3 {
		t.Errorf("expected NO overlap for Nov 25 - Nov 30")
	}

	// Case 4: Back-to-back check-in on departure day (Dec 5 to Dec 8) -> NO OVERLAP
	// Guest A leaves Dec 5 at 11:00 AM; Guest B arrives Dec 5 at 14:00 PM
	overlap4 := isOverlap(
		time.Date(2024, 12, 5, 14, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 8, 11, 0, 0, 0, time.UTC),
	)
	if overlap4 {
		t.Errorf("expected NO overlap for back-to-back booking starting on checkout date Dec 5")
	}
}

// TestRoomStatusStateTransitions verifies allowed physical room statuses
func TestRoomStatusStateTransitions(t *testing.T) {
	validStatuses := map[string]bool{
		"available":      true,
		"reserved":       true,
		"occupied":       true,
		"dirty":          true,
		"cleaning":       true,
		"out_of_service": true,
	}

	room := Room{
		RoomNumber: "201",
		Floor:      2,
		Rate:       4500.00,
		Status:     "available",
	}

	if !validStatuses[room.Status] {
		t.Errorf("room status %s should be valid", room.Status)
	}

	// Transition: Booking confirmed -> reserved
	room.Status = "reserved"
	if !validStatuses[room.Status] {
		t.Errorf("status transition to reserved should be valid")
	}

	// Transition: Check-in -> occupied
	room.Status = "occupied"
	if !validStatuses[room.Status] {
		t.Errorf("status transition to occupied should be valid")
	}

	// Transition: Check-out -> dirty
	room.Status = "dirty"
	if !validStatuses[room.Status] {
		t.Errorf("status transition to dirty should be valid")
	}
}
