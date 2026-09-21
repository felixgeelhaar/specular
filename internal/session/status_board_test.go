package session

import "testing"

func TestBuildStatusBoard(t *testing.T) {
	t.Parallel()
	list := []Record{
		{ID: "a", Status: StatusWorking},
		{ID: "b", Status: StatusQueued},
		{ID: "c", Status: StatusCompleted},
		{ID: "d", Status: StatusFailed},
		{ID: "e", Status: StatusStopped},
		{ID: "f", Status: StatusWaiting},
		{ID: "g", Status: "mystery"},
	}
	board := BuildStatusBoard(list)
	if board.Summary.Total != 7 {
		t.Fatalf("total=%d", board.Summary.Total)
	}
	if board.Summary.Working != 2 { // working + waiting
		t.Fatalf("working=%d", board.Summary.Working)
	}
	if board.Summary.Queued != 1 || board.Summary.Completed != 1 || board.Summary.Failed != 1 || board.Summary.Stopped != 1 {
		t.Fatalf("%+v", board.Summary)
	}
	if board.Summary.Other != 1 {
		t.Fatalf("other=%d", board.Summary.Other)
	}
	if len(board.Sessions) != 7 {
		t.Fatalf("sessions=%d", len(board.Sessions))
	}
	// Defensive copy: mutating input shouldn't change board snapshot length contract.
	list[0].Status = StatusFailed
	if board.Sessions[0].Status != StatusWorking {
		t.Fatal("expected board to own a copy of session records")
	}
}
