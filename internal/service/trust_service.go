package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/openpup/agora/internal/core"
	"github.com/openpup/agora/internal/repository"
)

type TrustService struct {
	agents      repository.AgentRepository
	signals     repository.SignalRepository
	resolutions repository.ResolutionRepository
}

type roleStats struct {
	total   int
	correct int
}

type agentDomainTrustStats struct {
	avgConfidence     float64
	claims            roleStats
	counters          roleStats
	resolutions       roleStats
	challenges        roleStats
	confidenceTotal   float64
	confidenceSamples int
}

func NewTrustService(agents repository.AgentRepository, signals repository.SignalRepository, resolutions repository.ResolutionRepository) *TrustService {
	return &TrustService{agents: agents, signals: signals, resolutions: resolutions}
}

func (s *TrustService) Recalculate(ctx context.Context) error {
	signals, err := s.signals.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("trust_service.Recalculate signals: %w", err)
	}
	claimResolutions, err := s.resolutions.ListAllClaimResolutions(ctx)
	if err != nil {
		return fmt.Errorf("trust_service.Recalculate claim resolutions: %w", err)
	}
	attestations, err := s.resolutions.ListAllAttestations(ctx)
	if err != nil {
		return fmt.Errorf("trust_service.Recalculate attestations: %w", err)
	}

	signalByID := map[string]core.Signal{}
	rootClaimBySignalID := map[string]*core.Signal{}
	for _, signal := range signals {
		signalByID[signal.ID] = signal
	}
	for _, signal := range signals {
		rootClaimBySignalID[signal.ID] = findRootClaimSignal(signal.ID, signalByID)
	}

	resolutionByClaimID := map[string]core.ClaimResolution{}
	for _, resolution := range claimResolutions {
		resolutionByClaimID[resolution.ClaimID] = resolution
	}

	stats := map[string]map[string]*agentDomainTrustStats{}
	ensureStats := func(agentID, domain string) *agentDomainTrustStats {
		if _, ok := stats[agentID]; !ok {
			stats[agentID] = map[string]*agentDomainTrustStats{}
		}
		if _, ok := stats[agentID][domain]; !ok {
			stats[agentID][domain] = &agentDomainTrustStats{}
		}
		return stats[agentID][domain]
	}

	for _, signal := range signals {
		record := ensureStats(signal.AgentID, signal.Domain)
		if signal.Claim.Confidence > 0 {
			record.confidenceTotal += signal.Claim.Confidence
			record.confidenceSamples++
		}
		switch signal.Kind {
		case core.SignalKindClaim:
			resolution, ok := resolutionByClaimID[signal.ID]
			if !ok || resolution.State != core.ResolutionStateResolved || resolution.Outcome == nil {
				continue
			}
			record.claims.total++
			if *resolution.Outcome {
				record.claims.correct++
			}
		case core.SignalKindCounter:
			root := rootClaimBySignalID[signal.ID]
			if root == nil {
				continue
			}
			resolution, ok := resolutionByClaimID[root.ID]
			if !ok || resolution.State != core.ResolutionStateResolved || resolution.Outcome == nil {
				continue
			}
			record.counters.total++
			if isSignalDirectionVindicated(signal, *resolution.Outcome, root) {
				record.counters.correct++
			}
		}
	}

	for _, row := range attestations {
		claim := signalByID[row.Attestation.ClaimID]
		record := ensureStats(row.Attestation.AgentID, claim.Domain)
		resolution, ok := resolutionByClaimID[row.Attestation.ClaimID]
		if !ok {
			continue
		}
		switch row.Attestation.Kind {
		case core.ResolutionAttestationResolve:
			if resolution.State != core.ResolutionStateResolved || resolution.Outcome == nil || row.Attestation.Verdict == nil {
				continue
			}
			record.resolutions.total++
			if *row.Attestation.Verdict == *resolution.Outcome {
				record.resolutions.correct++
			}
		case core.ResolutionAttestationChallenge:
			record.challenges.total++
			if resolution.State == core.ResolutionStateChallenged {
				record.challenges.correct++
			}
		}
	}

	maxima := maxRoleTotals(stats)
	overallStats := map[string]struct {
		weighted float64
		weights  float64
	}{}
	now := time.Now().UTC()
	for agentID, byDomain := range stats {
		for domain, stat := range byDomain {
			stat.avgConfidence = average(stat.confidenceTotal, stat.confidenceSamples)
			claimAccuracy := accuracy(stat.claims)
			counterAccuracy := accuracy(stat.counters)
			resolutionAccuracy := accuracy(stat.resolutions)
			challengeAccuracy := accuracy(stat.challenges)
			profile := core.AgentTrustProfile{
				ClaimTrust:     dimensionTrust(stat.claims, maxima.claims),
				CounterTrust:   dimensionTrust(stat.counters, maxima.counters),
				ResolverTrust:  dimensionTrust(stat.resolutions, maxima.resolutions),
				ChallengeTrust: dimensionTrust(stat.challenges, maxima.challenges),
			}
			rec := core.AgentTrackRecord{
				AgentID:              agentID,
				Domain:               domain,
				TotalPredictions:     stat.claims.total,
				CorrectPredictions:   stat.claims.correct,
				Accuracy:             claimAccuracy,
				TotalClaims:          stat.claims.total,
				CorrectClaims:        stat.claims.correct,
				ClaimAccuracy:        claimAccuracy,
				TotalCounters:        stat.counters.total,
				CorrectCounters:      stat.counters.correct,
				CounterAccuracy:      counterAccuracy,
				TotalResolutions:     stat.resolutions.total,
				AlignedResolutions:   stat.resolutions.correct,
				ResolutionAccuracy:   resolutionAccuracy,
				TotalChallenges:      stat.challenges.total,
				SuccessfulChallenges: stat.challenges.correct,
				ChallengeAccuracy:    challengeAccuracy,
				ClaimTrust:           profile.ClaimTrust,
				CounterTrust:         profile.CounterTrust,
				ResolverTrust:        profile.ResolverTrust,
				ChallengeTrust:       profile.ChallengeTrust,
				AvgConfidence:        stat.avgConfidence,
				LastCalculatedAt:     now,
			}
			if err := s.agents.UpsertTrackRecord(ctx, rec); err != nil {
				return fmt.Errorf("trust_service.Recalculate upsert track record: %w", err)
			}
			weightClaims := roleWeight(stat.claims.total, 1.4)
			weightCounters := roleWeight(stat.counters.total, 1.1)
			weightResolvers := roleWeight(stat.resolutions.total, 1.0)
			weightChallenges := roleWeight(stat.challenges.total, 0.9)
			totalWeight := weightClaims + weightCounters + weightResolvers + weightChallenges
			if totalWeight == 0 {
				continue
			}
			overall := overallStats[agentID]
			overall.weighted += profile.ClaimTrust*weightClaims + profile.CounterTrust*weightCounters + profile.ResolverTrust*weightResolvers + profile.ChallengeTrust*weightChallenges
			overall.weights += totalWeight
			overallStats[agentID] = overall
		}
	}

	for agentID, overall := range overallStats {
		profile, err := aggregateAgentProfile(byDomainProfiles(stats[agentID], maxima))
		if err != nil {
			return fmt.Errorf("trust_service.Recalculate aggregate profile: %w", err)
		}
		trust := 0.5
		if overall.weights > 0 {
			trust = overall.weighted / overall.weights
		}
		if err := s.agents.UpdateTrustProfile(ctx, agentID, trust, profile); err != nil {
			return fmt.Errorf("trust_service.Recalculate update trust profile: %w", err)
		}
	}

	return nil
}

type roleMaxima struct {
	claims      int
	counters    int
	resolutions int
	challenges  int
}

func maxRoleTotals(stats map[string]map[string]*agentDomainTrustStats) roleMaxima {
	maxima := roleMaxima{claims: 1, counters: 1, resolutions: 1, challenges: 1}
	for _, byDomain := range stats {
		for _, stat := range byDomain {
			if stat.claims.total > maxima.claims {
				maxima.claims = stat.claims.total
			}
			if stat.counters.total > maxima.counters {
				maxima.counters = stat.counters.total
			}
			if stat.resolutions.total > maxima.resolutions {
				maxima.resolutions = stat.resolutions.total
			}
			if stat.challenges.total > maxima.challenges {
				maxima.challenges = stat.challenges.total
			}
		}
	}
	return maxima
}

func findRootClaimSignal(signalID string, index map[string]core.Signal) *core.Signal {
	current, ok := index[signalID]
	if !ok {
		return nil
	}
	for {
		if current.Kind == core.SignalKindClaim {
			signal := current
			return &signal
		}
		if current.ParentID == nil {
			return nil
		}
		parent, ok := index[*current.ParentID]
		if !ok {
			return nil
		}
		current = parent
	}
}

func isSignalDirectionVindicated(signal core.Signal, outcome bool, root *core.Signal) bool {
	winningDirection := expectedDirection(root, outcome)
	return signalDirection(signal) == winningDirection
}

func expectedDirection(root *core.Signal, outcome bool) string {
	direction := signalDirection(*root)
	if outcome {
		return direction
	}
	switch direction {
	case "bullish":
		return "bearish"
	case "bearish":
		return "bullish"
	default:
		return "neutral"
	}
}

func signalDirection(signal core.Signal) string {
	if direction, ok := signal.Claim.Structured["direction"].(string); ok && direction != "" {
		return direction
	}
	return "neutral"
}

func accuracy(stats roleStats) float64 {
	if stats.total == 0 {
		return 0
	}
	return float64(stats.correct) / float64(stats.total)
}

func dimensionTrust(stats roleStats, maxTotal int) float64 {
	if stats.total == 0 {
		return 0.5
	}
	acc := accuracy(stats)
	return acc * math.Log(float64(stats.total)+1) / math.Log(float64(maxTotal)+1)
}

func average(total float64, count int) float64 {
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func roleWeight(count int, multiplier float64) float64 {
	if count == 0 {
		return 0
	}
	return float64(count) * multiplier
}

func byDomainProfiles(byDomain map[string]*agentDomainTrustStats, maxima roleMaxima) []core.AgentTrustProfile {
	out := make([]core.AgentTrustProfile, 0, len(byDomain))
	for _, stat := range byDomain {
		out = append(out, core.AgentTrustProfile{
			ClaimTrust:     dimensionTrust(stat.claims, maxima.claims),
			CounterTrust:   dimensionTrust(stat.counters, maxima.counters),
			ResolverTrust:  dimensionTrust(stat.resolutions, maxima.resolutions),
			ChallengeTrust: dimensionTrust(stat.challenges, maxima.challenges),
		})
	}
	return out
}

func aggregateAgentProfile(profiles []core.AgentTrustProfile) (core.AgentTrustProfile, error) {
	if len(profiles) == 0 {
		return core.AgentTrustProfile{
			ClaimTrust:     0.5,
			CounterTrust:   0.5,
			ResolverTrust:  0.5,
			ChallengeTrust: 0.5,
		}, nil
	}
	var profile core.AgentTrustProfile
	for _, item := range profiles {
		profile.ClaimTrust += item.ClaimTrust
		profile.CounterTrust += item.CounterTrust
		profile.ResolverTrust += item.ResolverTrust
		profile.ChallengeTrust += item.ChallengeTrust
	}
	divisor := float64(len(profiles))
	profile.ClaimTrust /= divisor
	profile.CounterTrust /= divisor
	profile.ResolverTrust /= divisor
	profile.ChallengeTrust /= divisor
	return profile, nil
}
