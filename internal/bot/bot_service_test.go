package bot

import (
	"testing"
)

func TestBuildHistoryKeyboard(t *testing.T) {
	bs := &BotService{}

	tests := []struct {
		name           string
		currentPage    int
		totalPages     int
		wantNil        bool
		wantPrev       bool
		wantNext       bool
		wantPrevData   string
		wantNextData   string
	}{
		{
			name:        "single page returns nil",
			currentPage: 0,
			totalPages:  1,
			wantNil:     true,
		},
		{
			name:         "first page of many shows only next",
			currentPage:  0,
			totalPages:   3,
			wantNil:      false,
			wantPrev:     false,
			wantNext:     true,
			wantNextData: "1",
		},
		{
			name:         "middle page shows both",
			currentPage:  1,
			totalPages:   3,
			wantNil:      false,
			wantPrev:     true,
			wantNext:     true,
			wantPrevData: "0",
			wantNextData: "2",
		},
		{
			name:         "last page shows only prev",
			currentPage:  2,
			totalPages:   3,
			wantNil:      false,
			wantPrev:     true,
			wantNext:     false,
			wantPrevData: "1",
		},
		{
			name:         "two pages - first page",
			currentPage:  0,
			totalPages:   2,
			wantNil:      false,
			wantPrev:     false,
			wantNext:     true,
			wantNextData: "1",
		},
		{
			name:         "two pages - last page",
			currentPage:  1,
			totalPages:   2,
			wantNil:      false,
			wantPrev:     true,
			wantNext:     false,
			wantPrevData: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyboard := bs.buildHistoryKeyboard(tt.currentPage, tt.totalPages)

			if tt.wantNil {
				if keyboard != nil {
					t.Errorf("expected nil keyboard, got %+v", keyboard)
				}
				return
			}

			if keyboard == nil {
				t.Fatal("expected keyboard, got nil")
			}

			if len(keyboard.InlineKeyboard) != 1 {
				t.Fatalf("expected 1 row, got %d", len(keyboard.InlineKeyboard))
			}

			buttons := keyboard.InlineKeyboard[0]

			// Count expected buttons
			expectedCount := 0
			if tt.wantPrev {
				expectedCount++
			}
			if tt.wantNext {
				expectedCount++
			}

			if len(buttons) != expectedCount {
				t.Fatalf("expected %d buttons, got %d", expectedCount, len(buttons))
			}

			buttonIdx := 0

			if tt.wantPrev {
				if buttons[buttonIdx].Unique != "history_prev" {
					t.Errorf("expected prev button unique 'history_prev', got '%s'", buttons[buttonIdx].Unique)
				}
				if buttons[buttonIdx].Data != tt.wantPrevData {
					t.Errorf("expected prev data '%s', got '%s'", tt.wantPrevData, buttons[buttonIdx].Data)
				}
				buttonIdx++
			}

			if tt.wantNext {
				if buttons[buttonIdx].Unique != "history_next" {
					t.Errorf("expected next button unique 'history_next', got '%s'", buttons[buttonIdx].Unique)
				}
				if buttons[buttonIdx].Data != tt.wantNextData {
					t.Errorf("expected next data '%s', got '%s'", tt.wantNextData, buttons[buttonIdx].Data)
				}
			}
		})
	}
}
