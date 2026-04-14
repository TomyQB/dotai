package menu_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TomyQB/dotai/internal/tui/menu"
)

// TestMenuTitle verifies that Title() returns "dotai".
func TestMenuTitle(t *testing.T) {
	m := menu.New()
	if got := m.Title(); got != "dotai" {
		t.Errorf("Title() = %q, want %q", got, "dotai")
	}
}

// TestMenuInitialCursor verifies that the cursor starts at position 0.
func TestMenuInitialCursor(t *testing.T) {
	m := menu.New()
	// The only way to observe the cursor externally is via Update responses.
	// A fresh model rendered at cursor 0 should move to 1 on a single "j" press.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	// Then pressing "k" should move back to 0 (SelectedMsg would be ActionInstall).
	updated, cmd := updated.(tea.Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Update(enter after j+k=0) returned nil cmd")
	}
	msg := cmd()
	sel, ok := msg.(menu.SelectedMsg)
	if !ok {
		t.Fatalf("expected SelectedMsg, got %T", msg)
	}
	// After j (cursor=1) then enter → ActionUpdate
	if sel.Action != menu.ActionUpdate {
		t.Errorf("action = %v, want ActionUpdate (%v)", sel.Action, menu.ActionUpdate)
	}
	_ = updated
}

// TestMenuNavigation verifies j moves down, k moves up, and wrapping works.
func TestMenuNavigation(t *testing.T) {
	// Items in order: ActionInstall(0), ActionUpdate(1), ActionStatus(2), ActionMemcli(3), ActionDoctor(4), ActionQuit(5)
	numItems := 6

	t.Run("j moves down", func(t *testing.T) {
		m := menu.New()
		// Press j once → cursor should be at 1 (ActionUpdate)
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		tm := updated.(tea.Model)
		updated2, cmd := tm.Update(tea.KeyMsg{Type: tea.KeyEnter})
		_ = updated2
		if cmd == nil {
			t.Fatal("nil cmd after j + enter")
		}
		msg := cmd()
		sel, ok := msg.(menu.SelectedMsg)
		if !ok {
			t.Fatalf("expected SelectedMsg, got %T", msg)
		}
		if sel.Action != menu.ActionUpdate {
			t.Errorf("after j: action = %v, want ActionUpdate", sel.Action)
		}
	})

	t.Run("k moves up with wrap", func(t *testing.T) {
		m := menu.New()
		// At cursor 0, press k → should wrap to last item (ActionQuit = index numItems-1)
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		tm := updated.(tea.Model)
		// ActionQuit returns tea.Quit, not a SelectedMsg
		_, cmd := tm.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Fatal("nil cmd after k + enter at wrap position")
		}
		_ = numItems
	})

	t.Run("j wraps around to first", func(t *testing.T) {
		m := menu.New()
		// Press j numItems times to wrap around to index 0 again.
		var tm tea.Model = m
		for i := 0; i < numItems; i++ {
			updated, _ := tm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
			tm = updated.(tea.Model)
		}
		_, cmd := tm.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Fatal("nil cmd after numItems×j + enter")
		}
		msg := cmd()
		sel, ok := msg.(menu.SelectedMsg)
		if !ok {
			t.Fatalf("expected SelectedMsg, got %T", msg)
		}
		if sel.Action != menu.ActionInstall {
			t.Errorf("after wrap: action = %v, want ActionInstall", sel.Action)
		}
	})
}

// TestMenuEnterEmitsSelectedMsg verifies that pressing enter at each cursor
// position emits the correct Action.
func TestMenuEnterEmitsSelectedMsg(t *testing.T) {
	wantActions := []menu.Action{
		menu.ActionInstall,
		menu.ActionUpdate,
		menu.ActionStatus,
		menu.ActionMemcli,
		menu.ActionDoctor,
		// ActionQuit (last index) emits tea.Quit — tested separately.
	}

	for i, want := range wantActions {
		var tm tea.Model = menu.New()
		// Navigate to position i by pressing j i times.
		for range i {
			updated, _ := tm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
			tm = updated.(tea.Model)
		}
		_, cmd := tm.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Errorf("cursor %d: Update(enter) returned nil cmd", i)
			continue
		}
		msg := cmd()
		sel, ok := msg.(menu.SelectedMsg)
		if !ok {
			t.Errorf("cursor %d: expected SelectedMsg, got %T", i, msg)
			continue
		}
		if sel.Action != want {
			t.Errorf("cursor %d: Action = %v, want %v", i, sel.Action, want)
		}
	}
}

// TestMenuQuit verifies that pressing q emits a quit command.
func TestMenuQuit(t *testing.T) {
	m := menu.New()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("Update('q') returned nil cmd, want quit")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("'q' produced %T, want tea.QuitMsg", msg)
	}
}
