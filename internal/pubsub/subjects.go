package pubsub

import (
	"fmt"

	"github.com/openpup/agora/internal/domain"
)

func SignalPublishedSubject(market domain.Market, ticker string) string {
	return fmt.Sprintf("signals.published.%s.%s", market, ticker)
}

func SignalCounteredSubject(market domain.Market, ticker string) string {
	return fmt.Sprintf("signals.countered.%s.%s", market, ticker)
}

func SignalVerifiedSubject(market domain.Market, ticker string) string {
	return fmt.Sprintf("signals.verified.%s.%s", market, ticker)
}

func AgentTrustUpdatedSubject(agentID string) string {
	return fmt.Sprintf("agents.trust.updated.%s", agentID)
}

func MarketDataSubject(market domain.Market, ticker string) string {
	return fmt.Sprintf("market.data.%s.%s", market, ticker)
}
