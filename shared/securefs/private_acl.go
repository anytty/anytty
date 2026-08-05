package securefs

type privateAccessRule struct {
	allow   bool
	trustee string
}

func privateDescriptorAllowed(owner, current, system, administrators string, protected bool, rules []privateAccessRule) bool {
	if owner == "" || current == "" || owner != current || !protected {
		return false
	}
	return privateAllowRulesAllowed(current, system, administrators, rules)
}

func privateAllowRulesAllowed(current, system, administrators string, rules []privateAccessRule) bool {
	if current == "" {
		return false
	}
	currentAllowed := false
	for _, rule := range rules {
		if !rule.allow {
			continue
		}
		switch rule.trustee {
		case current:
			currentAllowed = true
		case system, administrators:
		default:
			return false
		}
	}
	return currentAllowed
}
