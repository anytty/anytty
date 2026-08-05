package action

import "testing"

func TestPaneCTASurfaceGroupsContainOnlyRegisteredCanonicalActions(t *testing.T) {
	groups := map[string][]ID{
		"empty":        EmptyPaneCTAActions(),
		"exited":       ExitedPaneCTAActions(),
		"disconnected": DisconnectedPaneCTAActions(),
	}
	for name, actions := range groups {
		if len(actions) == 0 {
			t.Fatalf("%s CTA group must not be empty", name)
		}
		for _, id := range actions {
			if _, ok := SpecByID(id); !ok {
				t.Fatalf("%s CTA group contains unregistered action %q", name, id)
			}
		}
	}

	first := EmptyPaneCTAActions()
	first[0] = "invalid.test_action"
	if got := EmptyPaneCTAActions()[0]; got != ActionEmptyAttach {
		t.Fatalf("callers must not mutate action-domain CTA order, got %q", got)
	}
}
