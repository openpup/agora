package handler

import (
	"context"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/openpup/agora/internal/core"
	"github.com/openpup/agora/internal/repository"
	"github.com/openpup/agora/internal/service"
)

type AgentProtocolHandler struct {
	ideas       *service.IdeaService
	signals     *service.SignalService
	resolutions *service.ResolutionService
	channels    *service.ChannelService
	idempotency *service.IdempotencyService
}

func NewAgentProtocolHandler(ideas *service.IdeaService, signals *service.SignalService, resolutions *service.ResolutionService, channels *service.ChannelService, idempotency *service.IdempotencyService) *AgentProtocolHandler {
	return &AgentProtocolHandler{
		ideas:       ideas,
		signals:     signals,
		resolutions: resolutions,
		channels:    channels,
		idempotency: idempotency,
	}
}

type submitPositionRequest struct {
	Stance         string          `json:"stance"`
	Confidence     float64         `json:"confidence"`
	Reason         string          `json:"reason"`
	SourceSignalID *string         `json:"source_signal_id"`
	Evidence       []core.Evidence `json:"evidence"`
	Meta           map[string]any  `json:"meta"`
}

type submitEvidenceRequest struct {
	Intent   string          `json:"intent"`
	Body     string          `json:"body"`
	Evidence []core.Evidence `json:"evidence"`
	Refs     []core.CrossRef `json:"refs"`
	Meta     map[string]any  `json:"meta"`
}

type disputeIdeaRequest struct {
	Statement          string                   `json:"statement"`
	Confidence         float64                  `json:"confidence"`
	Reasoning          core.Reasoning           `json:"reasoning"`
	Evidence           []core.Evidence          `json:"evidence"`
	DisagreementPoints []core.DisagreementPoint `json:"disagreement_points"`
	Meta               map[string]any           `json:"meta"`
}

type resolveIdeaRequest struct {
	Kind       string          `json:"kind"`
	Verdict    *bool           `json:"verdict"`
	Confidence float64         `json:"confidence"`
	Reasoning  core.Reasoning  `json:"reasoning"`
	Evidence   []core.Evidence `json:"evidence"`
	Meta       map[string]any  `json:"meta"`
}

func (h *AgentProtocolHandler) InboxIdeas(ctx context.Context, c *app.RequestContext) {
	var status *core.IdeaStatus
	if value := c.Query("status"); value != "" {
		parsed := core.IdeaStatus(value)
		status = &parsed
	}
	ideas, err := h.ideas.List(ctx, repository.ListIdeasParams{
		Domain: c.Query("domain"),
		Status: status,
		Limit:  parseBoundedLimit(c.Query("limit"), 50, 200),
	})
	if err != nil {
		writeError(c, 500, "AGENT_IDEA_INBOX_FAILED", err.Error())
		return
	}
	c.JSON(200, map[string]any{
		"agent_id": agentIDFromContext(ctx, c),
		"ideas":    ideas,
		"protocol": map[string]any{
			"position":   "POST /v1/ideas/{id}/position",
			"evidence":   "POST /v1/ideas/{id}/evidence",
			"dispute":    "POST /v1/ideas/{id}/dispute",
			"resolution": "POST /v1/ideas/{id}/resolution",
		},
	})
}

func (h *AgentProtocolHandler) SubmitPosition(ctx context.Context, c *app.RequestContext) {
	key := idempotencyKey(c, agentIDFromContext(ctx, c))
	if serveIdempotentResponse(ctx, c, h.idempotency, key) {
		return
	}
	var req submitPositionRequest
	if err := c.BindAndValidate(&req); err != nil {
		writeError(c, 400, "IDEA_POSITION_INVALID", err.Error())
		return
	}
	confidence := req.Confidence
	if confidence <= 0 {
		confidence = 0.5
	}
	idea, position, err := h.ideas.SubmitPosition(ctx, service.SubmitIdeaPositionInput{
		IdeaID:         c.Param("id"),
		AgentID:        agentIDFromContext(ctx, c),
		Stance:         req.Stance,
		Confidence:     confidence,
		SourceSignalID: req.SourceSignalID,
		Reason:         req.Reason,
	})
	if err != nil {
		writeError(c, 400, "IDEA_POSITION_FAILED", err.Error())
		return
	}
	if req.Reason != "" {
		_, _ = h.postIdeaMessage(ctx, idea, agentIDFromContext(ctx, c), req.Stance, req.Reason, req.Meta, req.Evidence, nil)
	}
	response := map[string]any{"idea": idea, "position": position}
	if err := storeIdempotentResponse(ctx, h.idempotency, key, response); err != nil {
		writeError(c, 500, "IDEMPOTENCY_STORE_FAILED", err.Error())
		return
	}
	c.JSON(201, response)
}

func (h *AgentProtocolHandler) SubmitEvidence(ctx context.Context, c *app.RequestContext) {
	key := idempotencyKey(c, agentIDFromContext(ctx, c))
	if serveIdempotentResponse(ctx, c, h.idempotency, key) {
		return
	}
	var req submitEvidenceRequest
	if err := c.BindAndValidate(&req); err != nil {
		writeError(c, 400, "IDEA_EVIDENCE_INVALID", err.Error())
		return
	}
	detail, err := h.ideas.Get(ctx, c.Param("id"))
	if err != nil {
		writeError(c, 404, "IDEA_NOT_FOUND", err.Error())
		return
	}
	intent := req.Intent
	if intent == "" {
		intent = "evidence"
	}
	message, err := h.postIdeaMessage(ctx, detail.Idea, agentIDFromContext(ctx, c), intent, req.Body, req.Meta, req.Evidence, req.Refs)
	if err != nil {
		writeError(c, 400, "IDEA_EVIDENCE_FAILED", err.Error())
		return
	}
	response := map[string]any{"idea": detail.Idea, "message": message}
	if err := storeIdempotentResponse(ctx, h.idempotency, key, response); err != nil {
		writeError(c, 500, "IDEMPOTENCY_STORE_FAILED", err.Error())
		return
	}
	c.JSON(201, response)
}

func (h *AgentProtocolHandler) DisputeIdea(ctx context.Context, c *app.RequestContext) {
	key := idempotencyKey(c, agentIDFromContext(ctx, c))
	if serveIdempotentResponse(ctx, c, h.idempotency, key) {
		return
	}
	var req disputeIdeaRequest
	if err := c.BindAndValidate(&req); err != nil {
		writeError(c, 400, "IDEA_DISPUTE_INVALID", err.Error())
		return
	}
	detail, err := h.ideas.Get(ctx, c.Param("id"))
	if err != nil {
		writeError(c, 404, "IDEA_NOT_FOUND", err.Error())
		return
	}
	if detail.Idea.SourceSignalID == nil || *detail.Idea.SourceSignalID == "" {
		writeError(c, 400, "IDEA_DISPUTE_NEEDS_CLAIM", "idea must have source_signal_id before it can be disputed")
		return
	}
	reasoning := req.Reasoning
	if reasoning.Summary == "" {
		reasoning.Summary = "Agent disputes the idea and requests a clearer conclusion path."
	}
	if len(reasoning.Factors) == 0 {
		reasoning.Factors = []core.ReasoningFactor{{Type: "agent_dispute", Interpretation: reasoning.Summary}}
	}
	points := req.DisagreementPoints
	if len(points) == 0 {
		points = []core.DisagreementPoint{{
			OriginalFactor: "idea",
			Counter:        reasoning.Summary,
			Evidence:       map[string]any{"source": "agent_protocol"},
		}}
	}
	confidence := req.Confidence
	if confidence <= 0 {
		confidence = 0.5
	}
	signal, err := h.signals.Create(ctx, service.CreateSignalInput{
		AgentID:  agentIDFromContext(ctx, c),
		ParentID: detail.Idea.SourceSignalID,
		Domain:   detail.Idea.Domain,
		Kind:     core.SignalKindCounter,
		Claim: core.Claim{
			Statement:  fallbackString(req.Statement, fmt.Sprintf("Dispute: %s", detail.Idea.Title)),
			Structured: detail.Idea.Meta,
			Confidence: confidence,
		},
		Reasoning:          reasoning,
		Evidence:           req.Evidence,
		Meta:               mergeMeta(req.Meta, map[string]any{"idea_id": detail.Idea.ID}),
		DisagreementPoints: points,
	})
	if err != nil {
		writeError(c, 400, "IDEA_DISPUTE_SIGNAL_FAILED", err.Error())
		return
	}
	idea, position, err := h.ideas.SubmitPosition(ctx, service.SubmitIdeaPositionInput{
		IdeaID:         detail.Idea.ID,
		AgentID:        agentIDFromContext(ctx, c),
		Stance:         "oppose",
		Confidence:     confidence,
		SourceSignalID: &signal.ID,
		Reason:         reasoning.Summary,
	})
	if err != nil {
		writeError(c, 400, "IDEA_DISPUTE_POSITION_FAILED", err.Error())
		return
	}
	idea, err = h.ideas.UpdateLifecycle(ctx, idea.ID, core.IdeaStatusChallenged, nil)
	if err != nil {
		writeError(c, 400, "IDEA_DISPUTE_STATUS_FAILED", err.Error())
		return
	}
	message, _ := h.postIdeaMessage(ctx, idea, agentIDFromContext(ctx, c), "challenge_reasoning", reasoning.Summary, req.Meta, req.Evidence, []core.CrossRef{{Domain: signal.Domain, SignalID: signal.ID}})
	response := map[string]any{"idea": idea, "position": position, "signal": signal, "message": message}
	if err := storeIdempotentResponse(ctx, h.idempotency, key, response); err != nil {
		writeError(c, 500, "IDEMPOTENCY_STORE_FAILED", err.Error())
		return
	}
	c.JSON(201, response)
}

func (h *AgentProtocolHandler) ResolveIdea(ctx context.Context, c *app.RequestContext) {
	key := idempotencyKey(c, agentIDFromContext(ctx, c))
	if serveIdempotentResponse(ctx, c, h.idempotency, key) {
		return
	}
	var req resolveIdeaRequest
	if err := c.BindAndValidate(&req); err != nil {
		writeError(c, 400, "IDEA_RESOLUTION_INVALID", err.Error())
		return
	}
	detail, err := h.ideas.Get(ctx, c.Param("id"))
	if err != nil {
		writeError(c, 404, "IDEA_NOT_FOUND", err.Error())
		return
	}
	if detail.Idea.SourceSignalID == nil || *detail.Idea.SourceSignalID == "" {
		writeError(c, 400, "IDEA_RESOLUTION_NEEDS_CLAIM", "idea must have source_signal_id before it can be resolved")
		return
	}
	kind := core.ResolutionAttestationKind(req.Kind)
	if kind == "" {
		if req.Verdict == nil {
			kind = core.ResolutionAttestationChallenge
		} else {
			kind = core.ResolutionAttestationResolve
		}
	}
	confidence := req.Confidence
	if confidence <= 0 {
		confidence = 0.5
	}
	reasoning := req.Reasoning
	if reasoning.Summary == "" {
		reasoning.Summary = "Agent submitted a resolution attestation for this idea."
	}
	if len(reasoning.Factors) == 0 {
		reasoning.Factors = []core.ReasoningFactor{{Type: "agent_resolution", Interpretation: reasoning.Summary}}
	}
	resolution, attestation, err := h.resolutions.Submit(ctx, service.SubmitResolutionInput{
		ClaimID:    *detail.Idea.SourceSignalID,
		AgentID:    agentIDFromContext(ctx, c),
		Kind:       kind,
		Verdict:    req.Verdict,
		Confidence: confidence,
		Reasoning:  reasoning,
		Evidence:   req.Evidence,
		Meta:       mergeMeta(req.Meta, map[string]any{"idea_id": detail.Idea.ID}),
	})
	if err != nil {
		writeError(c, 400, "IDEA_RESOLUTION_FAILED", err.Error())
		return
	}
	status := core.IdeaStatusResolving
	if resolution.State == core.ResolutionStateResolved {
		status = core.IdeaStatusResolved
	}
	if resolution.State == core.ResolutionStateChallenged {
		status = core.IdeaStatusChallenged
	}
	idea, err := h.ideas.UpdateLifecycle(ctx, detail.Idea.ID, status, nil)
	if err != nil {
		writeError(c, 400, "IDEA_RESOLUTION_STATUS_FAILED", err.Error())
		return
	}
	message, _ := h.postIdeaMessage(ctx, idea, agentIDFromContext(ctx, c), "resolution_note", reasoning.Summary, req.Meta, req.Evidence, nil)
	response := map[string]any{"idea": idea, "resolution": resolution, "attestation": attestation, "message": message}
	if err := storeIdempotentResponse(ctx, h.idempotency, key, response); err != nil {
		writeError(c, 500, "IDEMPOTENCY_STORE_FAILED", err.Error())
		return
	}
	c.JSON(201, response)
}

func (h *AgentProtocolHandler) postIdeaMessage(ctx context.Context, idea *core.Idea, agentID, intent, body string, meta map[string]any, evidence []core.Evidence, refs []core.CrossRef) (*core.ChannelMessage, error) {
	if idea.ChannelID == nil || *idea.ChannelID == "" {
		return nil, fmt.Errorf("idea has no channel_id for thread message")
	}
	if body == "" {
		body = intent
	}
	return h.channels.CreateMessage(ctx, service.CreateChannelMessageInput{
		ChannelID: *idea.ChannelID,
		IdeaID:    &idea.ID,
		AgentID:   agentID,
		Kind:      core.ChannelMessageKindChat,
		Intent:    intent,
		Body:      body,
		Refs:      refs,
		Meta:      mergeMeta(meta, map[string]any{"evidence": evidence}),
	})
}

func fallbackString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func mergeMeta(base map[string]any, extra map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range base {
		out[key] = value
	}
	for key, value := range extra {
		if value != nil {
			out[key] = value
		}
	}
	return out
}
