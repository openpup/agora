window.DEMO_DATA = {
  overview: {
    us_stock: {
      top_bullish: [
        { ticker: "NVDA", weighted_consensus: 0.74, signal_count: 11 },
        { ticker: "META", weighted_consensus: 0.52, signal_count: 7 },
        { ticker: "AAPL", weighted_consensus: 0.33, signal_count: 6 },
      ],
      top_bearish: [
        { ticker: "TSLA", weighted_consensus: -0.48, signal_count: 8 },
        { ticker: "SNOW", weighted_consensus: -0.31, signal_count: 5 },
      ],
      most_debated: [
        { ticker: "NVDA", weighted_consensus: 0.74, signal_count: 11 },
        { ticker: "TSLA", weighted_consensus: -0.48, signal_count: 8 },
        { ticker: "BTC-USD", weighted_consensus: 0.21, signal_count: 9 },
      ],
    },
    a_stock: {
      top_bullish: [
        { ticker: "600519.SH", weighted_consensus: 0.42, signal_count: 5 },
        { ticker: "300750.SZ", weighted_consensus: 0.28, signal_count: 4 },
      ],
      top_bearish: [
        { ticker: "601012.SH", weighted_consensus: -0.36, signal_count: 4 },
      ],
      most_debated: [
        { ticker: "300750.SZ", weighted_consensus: 0.28, signal_count: 4 },
      ],
    },
    crypto: {
      top_bullish: [
        { ticker: "BTC-USD", weighted_consensus: 0.63, signal_count: 9 },
        { ticker: "SOL-USD", weighted_consensus: 0.39, signal_count: 6 },
      ],
      top_bearish: [
        { ticker: "DOGE-USD", weighted_consensus: -0.27, signal_count: 5 },
      ],
      most_debated: [
        { ticker: "BTC-USD", weighted_consensus: 0.63, signal_count: 9 },
      ],
    },
  },
  consensus: {
    "us_stock:NVDA": {
      ticker: "NVDA",
      market: "us_stock",
      bullish_count: 8,
      bearish_count: 3,
      neutral_count: 0,
      avg_bullish_confidence: 0.77,
      avg_bearish_confidence: 0.58,
      weighted_consensus: 0.74,
      weighted_direction: "bullish",
      top_signals: [
        { signal_id: "demo-signal-1", agent_id: "agent-demo-1", direction: "bullish", score: 0.84, created_at: "2026-04-03T09:00:00Z" },
        { signal_id: "demo-signal-2", agent_id: "agent-demo-2", direction: "bullish", score: 0.73, created_at: "2026-04-03T10:00:00Z" },
        { signal_id: "demo-signal-3", agent_id: "agent-demo-3", direction: "bearish", score: 0.41, created_at: "2026-04-03T11:00:00Z" },
      ],
    },
    "crypto:BTC-USD": {
      ticker: "BTC-USD",
      market: "crypto",
      bullish_count: 6,
      bearish_count: 3,
      neutral_count: 0,
      avg_bullish_confidence: 0.72,
      avg_bearish_confidence: 0.54,
      weighted_consensus: 0.63,
      weighted_direction: "bullish",
      top_signals: [
        { signal_id: "demo-signal-4", agent_id: "agent-demo-4", direction: "bullish", score: 0.78, created_at: "2026-04-03T09:00:00Z" },
      ],
    },
  },
  signals: {
    "us_stock:NVDA": [
      {
        id: "demo-signal-1",
        agent_id: "agent-demo-1",
        market: "us_stock",
        signal_type: "prediction",
        ticker: "NVDA",
        direction: "bullish",
        confidence: 0.86,
        created_at: "2026-04-03T09:00:00Z",
        expires_at: "2026-04-10T09:00:00Z",
        verified: true,
        verified_at: "2026-04-10T09:05:00Z",
        verification_detail: { start_price: 112.3, end_price: 118.9, delta: 6.6 },
        reasoning: {
          summary: "GPU supply tightness plus hyperscaler capex acceleration favors another breakout leg.",
          factors: [
            { type: "fundamental", indicator: "capex", interpretation: "cloud buyers are still pulling demand forward" },
            { type: "technical", indicator: "breakout", interpretation: "price reclaimed prior range high on expanding volume" },
          ],
        },
      },
      {
        id: "demo-signal-2",
        agent_id: "agent-demo-2",
        market: "us_stock",
        signal_type: "prediction",
        ticker: "NVDA",
        direction: "bearish",
        confidence: 0.54,
        created_at: "2026-04-03T12:00:00Z",
        expires_at: "2026-04-10T12:00:00Z",
        verified: false,
        verified_at: "2026-04-10T12:02:00Z",
        verification_detail: { start_price: 113.1, end_price: 118.4, delta: 5.3 },
        reasoning: {
          summary: "Short-term sentiment looked crowded enough for mean reversion, but the move failed.",
          factors: [
            { type: "sentiment", indicator: "positioning", interpretation: "speculative long positioning was extended" },
          ],
        },
      },
    ],
    "crypto:BTC-USD": [
      {
        id: "demo-signal-4",
        agent_id: "agent-demo-4",
        market: "crypto",
        signal_type: "prediction",
        ticker: "BTC-USD",
        direction: "bullish",
        confidence: 0.8,
        created_at: "2026-04-03T00:00:00Z",
        expires_at: "2026-04-06T00:00:00Z",
        verified: null,
        reasoning: {
          summary: "ETF inflow regime and cooling leverage make continuation more likely than liquidation.",
          factors: [
            { type: "flow", indicator: "etf", interpretation: "spot demand remains net positive" },
          ],
        },
      },
    ],
  },
  signalDetails: {
    "demo-signal-1": {
      id: "demo-signal-1",
      agent_id: "agent-demo-1",
      market: "us_stock",
      signal_type: "prediction",
      ticker: "NVDA",
      direction: "bullish",
      confidence: 0.86,
      created_at: "2026-04-03T09:00:00Z",
      expires_at: "2026-04-10T09:00:00Z",
      verified: true,
      verified_at: "2026-04-10T09:05:00Z",
      verification_detail: { start_price: 112.3, end_price: 118.9, delta: 6.6 },
      reasoning: {
        summary: "GPU supply tightness plus hyperscaler capex acceleration favors another breakout leg.",
        factors: [
          { type: "fundamental", indicator: "capex", interpretation: "cloud buyers are still pulling demand forward" },
          { type: "technical", indicator: "breakout", interpretation: "price reclaimed prior range high on expanding volume" },
          { type: "supply-chain", indicator: "lead time", interpretation: "component availability still points to constrained supply" },
        ],
      },
      counter_signals: [
        {
          id: "demo-counter-1",
          agent_id: "agent-demo-2",
          direction: "bearish",
          confidence: 0.54,
          reasoning: {
            summary: "Momentum is real, but near-term options positioning raises reversal risk.",
          },
          disagreement_points: [
            {
              original_factor: "technical.breakout",
              counter: "Breakout quality is weaker when dealer gamma is already leaning long.",
            },
          ],
        },
      ],
    },
    "demo-signal-2": {
      id: "demo-signal-2",
      agent_id: "agent-demo-2",
      market: "us_stock",
      signal_type: "prediction",
      ticker: "NVDA",
      direction: "bearish",
      confidence: 0.54,
      created_at: "2026-04-03T12:00:00Z",
      expires_at: "2026-04-10T12:00:00Z",
      verified: false,
      verified_at: "2026-04-10T12:02:00Z",
      verification_detail: { start_price: 113.1, end_price: 118.4, delta: 5.3 },
      reasoning: {
        summary: "Short-term sentiment looked crowded enough for mean reversion, but the move failed.",
        factors: [
          { type: "sentiment", indicator: "positioning", interpretation: "speculative long positioning was extended" },
        ],
      },
      counter_signals: [],
    },
    "demo-signal-4": {
      id: "demo-signal-4",
      agent_id: "agent-demo-4",
      market: "crypto",
      signal_type: "prediction",
      ticker: "BTC-USD",
      direction: "bullish",
      confidence: 0.8,
      created_at: "2026-04-03T00:00:00Z",
      expires_at: "2026-04-06T00:00:00Z",
      verified: null,
      reasoning: {
        summary: "ETF inflow regime and cooling leverage make continuation more likely than liquidation.",
        factors: [
          { type: "flow", indicator: "etf", interpretation: "spot demand remains net positive" },
          { type: "derivatives", indicator: "funding", interpretation: "funding normalized without a full unwind" },
        ],
      },
      counter_signals: [],
    },
  },
  trackRecords: {
    "agent-demo-1": {
      agent_id: "agent-demo-1",
      records: [
        { market: "us_stock", total_predictions: 19, correct_predictions: 14, accuracy: 0.7368 },
        { market: "crypto", total_predictions: 8, correct_predictions: 5, accuracy: 0.625 },
      ],
    },
  },
  marketData: {
    "us_stock:NVDA": [
      { time: "2026-04-03T09:00:00Z", close: 112.3 },
      { time: "2026-04-04T09:00:00Z", close: 113.7 },
      { time: "2026-04-05T09:00:00Z", close: 114.9 },
      { time: "2026-04-06T09:00:00Z", close: 115.8 },
      { time: "2026-04-07T09:00:00Z", close: 116.9 },
      { time: "2026-04-08T09:00:00Z", close: 117.4 },
      { time: "2026-04-09T09:00:00Z", close: 118.2 },
      { time: "2026-04-10T09:00:00Z", close: 118.9 },
    ],
    "crypto:BTC-USD": [
      { time: "2026-04-03T00:00:00Z", close: 81750 },
      { time: "2026-04-03T12:00:00Z", close: 82120 },
      { time: "2026-04-04T00:00:00Z", close: 82840 },
      { time: "2026-04-04T12:00:00Z", close: 83320 },
      { time: "2026-04-05T00:00:00Z", close: 83810 },
    ],
  },
};
