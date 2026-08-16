package securefs

import "testing"

func TestPrivateDescriptorAllowed(t *testing.T) {
	approved := []privateAccessRule{
		{allow: true, trustee: "current"},
		{allow: true, trustee: "system"},
		{allow: true, trustee: "administrators"},
		{allow: false, trustee: "other"},
	}
	tests := map[string]struct {
		owner     string
		protected bool
		rules     []privateAccessRule
		want      bool
	}{
		"approved protected DACL": {owner: "current", protected: true, rules: approved, want: true},
		"unprotected":             {owner: "current", protected: false, rules: approved},
		"wrong owner":             {owner: "other", protected: true, rules: approved},
		"missing current allow":   {owner: "current", protected: true, rules: []privateAccessRule{{allow: true, trustee: "system"}}},
		"additional allow":        {owner: "current", protected: true, rules: append(append([]privateAccessRule(nil), approved...), privateAccessRule{allow: true, trustee: "other"})},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := privateDescriptorAllowed(test.owner, "current", "system", "administrators", test.protected, test.rules); got != test.want {
				t.Fatalf("privateDescriptorAllowed()=%v want=%v", got, test.want)
			}
		})
	}
}

func TestPrivateAllowRulesAllowed(t *testing.T) {
	approved := []privateAccessRule{
		{allow: true, trustee: "current"},
		{allow: true, trustee: "system"},
		{allow: true, trustee: "administrators"},
		{allow: false, trustee: "other"},
	}
	if !privateAllowRulesAllowed("current", "system", "administrators", approved) {
		t.Fatal("approved inherited allow trustees were rejected")
	}
	withEveryone := append(append([]privateAccessRule(nil), approved...), privateAccessRule{allow: true, trustee: "everyone"})
	if privateAllowRulesAllowed("current", "system", "administrators", withEveryone) {
		t.Fatal("additional inherited allow trustee was accepted")
	}
}
