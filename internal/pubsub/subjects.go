package pubsub

import "fmt"

func SignalPublishedSubject(domain string, kind string) string {
	return fmt.Sprintf("signals.published.%s.%s", domain, kind)
}

func SignalCounteredSubject(domain string) string {
	return fmt.Sprintf("signals.countered.%s", domain)
}

func SignalVerifiedSubject(domain string) string {
	return fmt.Sprintf("signals.verified.%s", domain)
}

func AgentTrustUpdatedSubject(agentID string) string {
	return fmt.Sprintf("agents.trust.updated.%s", agentID)
}

func DomainEventSubject(domain string, event string) string {
	return fmt.Sprintf("domain.%s.%s", domain, event)
}
