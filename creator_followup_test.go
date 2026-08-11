package main

import (
	"testing"
	"time"
)

func TestDecideFollowupWaitsForDeliveredActiveSubscriber(t *testing.T) {
	got := decideFollowup(followupInput{Delivered: true, SubscriberActive: true, ProcessingHours: 6})
	if got.Action != "process-content" || got.Delay != 6*time.Hour {
		t.Fatalf("got %+v, want process-content after 6h", got)
	}

	got = decideFollowup(followupInput{Delivered: true, SubscriberActive: false, ProcessingHours: 6})
	if got.Action != "hold" || got.Delay != 0 {
		t.Fatalf("got %+v, want hold with no delay", got)
	}
}
