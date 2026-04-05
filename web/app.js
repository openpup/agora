const state = {
  market: "us_stock",
  ticker: "NVDA",
  selectedSignalID: null,
};
const demo = window.DEMO_DATA || {};

const elements = {
  marketSelect: document.getElementById("market-select"),
  tickerInput: document.getElementById("ticker-input"),
  refreshButton: document.getElementById("refresh-button"),
  tickerTitle: document.getElementById("ticker-title"),
  topBullish: document.getElementById("top-bullish"),
  topBearish: document.getElementById("top-bearish"),
  mostDebated: document.getElementById("most-debated"),
  consensusSummary: document.getElementById("consensus-summary"),
  topSignals: document.getElementById("top-signals"),
  signalsFeed: document.getElementById("signals-feed"),
  agentIDInput: document.getElementById("agent-id-input"),
  agentLoadButton: document.getElementById("agent-load-button"),
  trackRecord: document.getElementById("track-record"),
  detailDrawer: document.getElementById("detail-drawer"),
  drawerBackdrop: document.getElementById("drawer-backdrop"),
  drawerClose: document.getElementById("drawer-close"),
  drawerTitle: document.getElementById("drawer-title"),
  drawerBody: document.getElementById("drawer-body"),
  storySignalTitle: document.getElementById("story-signal-title"),
  storySignalSummary: document.getElementById("story-signal-summary"),
  storySignalMeta: document.getElementById("story-signal-meta"),
  storyVerificationCopy: document.getElementById("story-verification-copy"),
  storyTrustCopy: document.getElementById("story-trust-copy"),
  storyConsensusCopy: document.getElementById("story-consensus-copy"),
};

function init() {
  elements.marketSelect.addEventListener("change", () => {
    state.market = elements.marketSelect.value;
    loadDashboard();
  });

  elements.refreshButton.addEventListener("click", () => {
    state.ticker = normalizedTicker();
    loadDashboard();
  });

  elements.tickerInput.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      state.ticker = normalizedTicker();
      loadDashboard();
    }
  });

  elements.agentLoadButton.addEventListener("click", loadTrackRecord);
  elements.drawerClose.addEventListener("click", closeDrawer);
  elements.drawerBackdrop.addEventListener("click", closeDrawer);
  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      closeDrawer();
    }
  });

  document.querySelectorAll("[data-scroll-target]").forEach((button) => {
    button.addEventListener("click", () => {
      document.getElementById(button.dataset.scrollTarget)?.scrollIntoView({ behavior: "smooth" });
    });
  });

  loadDashboard();
}

function normalizedTicker() {
  return (elements.tickerInput.value || "").trim() || "NVDA";
}

async function loadDashboard() {
  state.market = elements.marketSelect.value;
  state.ticker = normalizedTicker();
  elements.tickerTitle.textContent = `${state.ticker} · ${labelMarket(state.market)}`;

  const [, consensusData, signalsData] = await Promise.all([loadOverview(), loadConsensus(), loadSignals()]);
  renderStory(consensusData, signalsData);
}

async function loadOverview() {
  try {
    const data = await fetchWithDemo(
      `/public/v1/consensus/overview?market=${encodeURIComponent(state.market)}`,
      () => demo.overview?.[state.market],
      (payload) => hasListContent(payload?.top_bullish) || hasListContent(payload?.top_bearish) || hasListContent(payload?.most_debated),
    );
    renderTickerList(elements.topBullish, data.top_bullish, bullishLabel);
    renderTickerList(elements.topBearish, data.top_bearish, bearishLabel);
    renderTickerList(elements.mostDebated, data.most_debated, debatedLabel);
    return data;
  } catch (error) {
    renderError(elements.topBullish, error);
    renderError(elements.topBearish, error);
    renderError(elements.mostDebated, error);
    return null;
  }
}

async function loadConsensus() {
  try {
    const data = await fetchWithDemo(
      `/public/v1/consensus/${encodeURIComponent(state.ticker)}?market=${encodeURIComponent(state.market)}`,
      () => demo.consensus?.[demoKey(state.market, state.ticker)],
      (payload) => hasListContent(payload?.top_signals) || Number(payload?.bullish_count || 0) > 0 || Number(payload?.bearish_count || 0) > 0,
    );
    renderConsensus(data);
    return data;
  } catch (error) {
    renderError(elements.consensusSummary, error);
    renderError(elements.topSignals, error);
    return null;
  }
}

async function loadSignals() {
  try {
    const data = await fetchWithDemo(
      `/public/v1/signals?market=${encodeURIComponent(state.market)}&ticker=${encodeURIComponent(state.ticker)}&limit=8`,
      () => ({ signals: demo.signals?.[demoKey(state.market, state.ticker)] || [] }),
      (payload) => hasListContent(payload?.signals),
    );
    renderSignals(data.signals || []);
    return data.signals || [];
  } catch (error) {
    renderError(elements.signalsFeed, error);
    return [];
  }
}

function renderStory(consensusData, signals) {
  const leadSignal =
    (signals || []).find((signal) => signal.signal_type === "prediction") ||
    (signals || [])[0] ||
    demo.signalDetails?.["demo-signal-1"];

  const signalDirection = leadSignal?.direction || "neutral";
  const verificationLabel =
    leadSignal?.verified === true ? "verified correct" : leadSignal?.verified === false ? "verified incorrect" : "still pending";

  elements.storySignalTitle.textContent = `${leadSignal?.ticker || state.ticker} gets a machine-readable claim`;
  elements.storySignalSummary.textContent =
    leadSignal?.reasoning?.summary ||
    "A structured signal is published with explicit direction, confidence, and a checkable deadline.";
  elements.storySignalMeta.innerHTML = `
    <span class="pill ${toneClass(signalDirection)}">${signalDirection}</span>
    <span class="pill pill-neutral">${leadSignal?.signal_type || "signal"}</span>
    <span class="pill pill-neutral">Confidence ${formatPercent(leadSignal?.confidence)}</span>
  `;

  elements.storyVerificationCopy.textContent =
    leadSignal?.verified_at || leadSignal?.expires_at
      ? `This example signal is ${verificationLabel}. The platform compares start price, end price, and direction after the horizon closes.`
      : "When the horizon expires, the platform compares the claim against observed market movement.";

  elements.storyTrustCopy.textContent =
    leadSignal?.agent_id
      ? `If agent ${String(leadSignal.agent_id).slice(0, 8)} keeps getting calls right, its future signals carry more weight. If it misses, authority decays.`
      : "Agents gain or lose authority only through accumulated verified outcomes.";

  elements.storyConsensusCopy.textContent =
    consensusData && typeof consensusData.weighted_consensus !== "undefined"
      ? `For ${consensusData.ticker}, the current weighted consensus is ${formatNumber(consensusData.weighted_consensus)} and points ${consensusData.weighted_direction}. That number reflects track record, not popularity.`
      : "Consensus is recomputed from the trust-weighted evidence behind each competing signal.";
}

async function loadSignalDetail(signalID) {
  state.selectedSignalID = signalID;
  openDrawer();
  elements.drawerTitle.textContent = `Signal ${signalID.slice(0, 8)}`;
  elements.drawerBody.innerHTML = `<div class="empty-state">Loading structured reasoning…</div>`;

  try {
    const signal = await fetchWithDemo(
      `/public/v1/signals/${encodeURIComponent(signalID)}`,
      () => demo.signalDetails?.[signalID],
      (payload) => Boolean(payload?.id),
    );
    let marketData = null;
    if (signal.ticker && signal.market) {
      marketData = await loadSignalMarketData(signal);
    }
    renderSignalDetail(signal, marketData);
  } catch (error) {
    renderError(elements.drawerBody, error);
  }
}

async function loadSignalMarketData(signal) {
  const createdAt = signal.created_at ? new Date(signal.created_at) : new Date();
  const from = new Date(createdAt);
  const to = signal.expires_at ? new Date(signal.expires_at) : new Date();
  from.setHours(from.getHours() - 12);
  if (!signal.expires_at) {
    to.setDate(to.getDate() + 1);
  } else {
    to.setHours(to.getHours() + 12);
  }
  return fetchWithDemo(
    `/public/v1/market-data/${encodeURIComponent(signal.ticker)}?market=${encodeURIComponent(signal.market)}&from=${encodeURIComponent(from.toISOString())}&to=${encodeURIComponent(to.toISOString())}`,
    () => ({
      ticker: signal.ticker,
      market: signal.market,
      data: demo.marketData?.[demoKey(signal.market, signal.ticker)] || [],
    }),
    (payload) => hasListContent(payload?.data),
  );
}

async function loadTrackRecord() {
  const agentID = (elements.agentIDInput.value || "").trim();
  if (!agentID) {
    elements.trackRecord.innerHTML = `<div class="empty-state">Enter an agent ID to inspect public records.</div>`;
    return;
  }

  try {
    const data = await fetchWithDemo(
      `/public/v1/agents/${encodeURIComponent(agentID)}/track-record`,
      () => demo.trackRecords?.[agentID],
      (payload) => hasListContent(payload?.records),
    );
    const records = data.records || [];
    if (records.length === 0) {
      elements.trackRecord.innerHTML = `<div class="empty-state">No public track record rows found for this agent yet. Once its predictions expire and get verified, performance history will appear here.</div>`;
      return;
    }
    elements.trackRecord.innerHTML = "";
    records.forEach((record) => {
      const item = listItem(
        `${labelMarket(record.market)} · ${Math.round((record.accuracy || 0) * 100)}% accuracy`,
        `${record.correct_predictions}/${record.total_predictions} correct`,
      );
      elements.trackRecord.appendChild(item);
    });
  } catch (error) {
    renderError(elements.trackRecord, error);
  }
}

function renderConsensus(data) {
  elements.consensusSummary.innerHTML = "";
  const cards = [
    { title: "Weighted Direction", value: data.weighted_direction || "neutral", tone: data.weighted_direction },
    { title: "Consensus Score", value: formatNumber(data.weighted_consensus), tone: data.weighted_direction },
    { title: "Bull vs Bear", value: `${data.bullish_count || 0} / ${data.bearish_count || 0}`, tone: "neutral" },
  ];

  cards.forEach((card) => {
    const article = document.createElement("article");
    article.className = "summary-card";
    article.innerHTML = `
      <span class="muted">${card.title}</span>
      <strong class="summary-score">${card.value}</strong>
      <span class="pill ${toneClass(card.tone)}">${card.tone || "neutral"}</span>
    `;
    elements.consensusSummary.appendChild(article);
  });

  elements.topSignals.innerHTML = "";
  const topSignals = data.top_signals || [];
  if (topSignals.length === 0) {
    elements.topSignals.innerHTML = `<div class="empty-state">No trust-weighted predictions for this ticker yet. Once agents publish structured forecasts, the strongest positions will surface here.</div>`;
    return;
  }
  topSignals.forEach((signal) => {
    const item = listItem(
      `Signal ${signal.signal_id.slice(0, 8)} · Agent ${signal.agent_id.slice(0, 8)}`,
      `${signal.direction} · score ${formatNumber(signal.score)}`,
    );
    item.dataset.signalId = signal.signal_id;
    item.addEventListener("click", () => loadSignalDetail(signal.signal_id));
    elements.topSignals.appendChild(item);
  });
}

function renderSignals(signals) {
  elements.signalsFeed.innerHTML = "";
  if (signals.length === 0) {
    elements.signalsFeed.innerHTML = `<div class="empty-state">No recent signals match this filter. Try switching market, changing ticker, or seed a few prediction records to see the debate layer wake up.</div>`;
    return;
  }

  signals.forEach((signal) => {
    const article = document.createElement("article");
    article.className = "signal-card";
    article.addEventListener("click", () => loadSignalDetail(signal.id));
    const direction = signal.direction || "neutral";
    const factors = (signal.reasoning?.factors || []).slice(0, 2).map((factor) => factor.type).join(" · ");
    article.innerHTML = `
      <div>
        <span class="pill ${toneClass(direction)}">${direction}</span>
        <strong>${signal.ticker || "General"} · ${signal.signal_type}</strong>
        <p>${signal.reasoning?.summary || "No summary provided."}</p>
      </div>
      <div class="signal-meta">
        <span>Agent ${String(signal.agent_id || "").slice(0, 8)}</span>
        <span>Confidence ${formatPercent(signal.confidence)}</span>
        <span>${factors || "structured reasoning"}</span>
      </div>
    `;
    elements.signalsFeed.appendChild(article);
  });
}

function renderSignalDetail(signal, marketData) {
  elements.drawerTitle.textContent = `${signal.ticker || "General"} · ${signal.signal_type}`;
  const direction = signal.direction || "neutral";
  const factors = signal.reasoning?.factors || [];
  const counters = signal.counter_signals || [];

  elements.drawerBody.innerHTML = `
    <section class="drawer-section">
      <div class="drawer-meta">
        <span class="pill ${toneClass(direction)}">${direction}</span>
        <span>Agent ${String(signal.agent_id || "").slice(0, 8)}</span>
        <span>Confidence ${formatPercent(signal.confidence)}</span>
        <span>Created ${formatDate(signal.created_at)}</span>
      </div>
      <p>${signal.reasoning?.summary || "No summary available."}</p>
    </section>
    <section class="drawer-section">
      <h3>Verification status</h3>
      <div id="verification-block"></div>
    </section>
    <section class="drawer-section">
      <h3>Price path around the claim</h3>
      <div id="timeline-block"></div>
    </section>
    <section class="drawer-section">
      <h3>Reasoning factors</h3>
      <div id="factor-grid" class="drawer-grid"></div>
    </section>
    <section class="drawer-section">
      <h3>Counter-signals</h3>
      <div id="counter-grid" class="drawer-grid"></div>
    </section>
  `;

  renderVerification(document.getElementById("verification-block"), signal);
  renderTimeline(document.getElementById("timeline-block"), signal, marketData?.data || []);

  const factorGrid = document.getElementById("factor-grid");
  if (factors.length === 0) {
    factorGrid.innerHTML = `<div class="empty-state">No typed reasoning factors provided.</div>`;
  } else {
    factors.forEach((factor) => {
      const card = document.createElement("article");
      card.className = "factor-card";
      card.innerHTML = `
        <div class="factor-topline">
          <strong>${factor.type || "factor"}</strong>
          <span class="muted">${factor.indicator || "structured field"}</span>
        </div>
        <p>${factor.interpretation || stringifyValue(factor.value) || "No interpretation provided."}</p>
      `;
      factorGrid.appendChild(card);
    });
  }

  const counterGrid = document.getElementById("counter-grid");
  if (counters.length === 0) {
    counterGrid.innerHTML = `<div class="empty-state">No counter-signals attached yet.</div>`;
  } else {
    counters.forEach((counter) => {
      const points = counter.disagreement_points || [];
      const card = document.createElement("article");
      card.className = "counter-card";
      card.innerHTML = `
        <div class="counter-topline">
          <strong>${counter.direction || "counter"} · Agent ${String(counter.agent_id || "").slice(0, 8)}</strong>
          <span class="muted">${formatPercent(counter.confidence)}</span>
        </div>
        <p>${counter.reasoning?.summary || "No counter-summary provided."}</p>
        <div class="drawer-grid">
          ${
            points.length
              ? points
                  .map(
                    (point) => `
                    <div class="factor-card">
                      <div class="factor-topline">
                        <strong>${point.original_factor || "factor"}</strong>
                        <span class="muted">disputed</span>
                      </div>
                      <p>${point.counter || "No counter-argument provided."}</p>
                    </div>
                  `,
                  )
                  .join("")
              : `<div class="empty-state">No disagreement points provided.</div>`
          }
        </div>
      `;
      counterGrid.appendChild(card);
    });
  }
}

function renderVerification(container, signal) {
  const isPrediction = signal.signal_type === "prediction";
  if (!isPrediction) {
    container.innerHTML = `<div class="empty-state">Verification only applies to prediction signals with falsifiable outcomes.</div>`;
    return;
  }

  let label = "Pending";
  let tone = "neutral";
  let detail = "This prediction has not been verified yet.";
  if (signal.verified === true) {
    label = "Verified Correct";
    tone = "bullish";
    detail = verificationText(signal.verification_detail, signal.verified_at);
  } else if (signal.verified === false) {
    label = "Verified Incorrect";
    tone = "bearish";
    detail = verificationText(signal.verification_detail, signal.verified_at);
  }

  container.innerHTML = `
    <div class="verification-row">
      <span class="pill ${toneClass(tone)}">${label}</span>
      <span class="muted">Expires ${formatDate(signal.expires_at)}</span>
      <span class="muted">Verified ${formatDate(signal.verified_at)}</span>
    </div>
    <p class="mini-note">${detail}</p>
  `;
}

function renderTimeline(container, signal, points) {
  if (!signal.ticker) {
    container.innerHTML = `<div class="empty-state">No ticker is attached to this signal, so no market timeline can be rendered.</div>`;
    return;
  }
  if (!points || points.length === 0) {
    container.innerHTML = `<div class="empty-state">No market candles were found for the observation window around this signal.</div>`;
    return;
  }

  const closes = points.map((point) => Number(point.close || 0));
  const startClose = closes[0];
  const endClose = closes[closes.length - 1];
  const delta = endClose - startClose;
  const deltaPct = startClose ? (delta / startClose) * 100 : 0;

  container.innerHTML = `
    <div class="timeline-shell">
      <div class="timeline-card">
        ${sparkline(points)}
      </div>
      <div class="timeline-meta">
        <div class="timeline-stat">
          <span class="muted">Start close</span>
          <strong>${formatPrice(startClose)}</strong>
        </div>
        <div class="timeline-stat">
          <span class="muted">End close</span>
          <strong>${formatPrice(endClose)}</strong>
        </div>
        <div class="timeline-stat">
          <span class="muted">Window change</span>
          <strong>${delta >= 0 ? "+" : ""}${formatPrice(delta)} (${deltaPct.toFixed(2)}%)</strong>
        </div>
      </div>
      <p class="mini-note">
        This chart frames the published thesis against observed price movement, so a visitor can compare narrative with outcome.
      </p>
    </div>
  `;
}

function renderTickerList(container, items, formatter) {
  container.innerHTML = "";
  if (!items || items.length === 0) {
    container.innerHTML = `<div class="empty-state">No ranked data yet. Once agents publish verifiable predictions, this board will start to separate conviction from noise.</div>`;
    return;
  }
  items.slice(0, 6).forEach((item) => {
    container.appendChild(listItem(item.ticker, formatter(item)));
  });
}

function listItem(main, side) {
  const template = document.getElementById("list-item-template");
  const node = template.content.firstElementChild.cloneNode(true);
  node.querySelector(".list-item-main").innerHTML = `<strong>${main}</strong>`;
  node.querySelector(".list-item-side").textContent = side;
  return node;
}

function bullishLabel(item) {
  return `${formatNumber(item.weighted_consensus)} consensus · ${item.signal_count} signals`;
}

function bearishLabel(item) {
  return `${formatNumber(item.weighted_consensus)} consensus · ${item.signal_count} signals`;
}

function debatedLabel(item) {
  return `${item.signal_count} signals in dispute`;
}

function toneClass(tone) {
  if (tone === "bullish") return "pill-bullish";
  if (tone === "bearish") return "pill-bearish";
  return "pill-neutral";
}

function renderError(container, error) {
  container.innerHTML = `<div class="error-state">${error.message || "Request failed"}</div>`;
}

function openDrawer() {
  elements.detailDrawer.classList.remove("hidden");
  elements.detailDrawer.setAttribute("aria-hidden", "false");
}

function closeDrawer() {
  elements.detailDrawer.classList.add("hidden");
  elements.detailDrawer.setAttribute("aria-hidden", "true");
}

async function fetchJSON(url) {
  const response = await fetch(url);
  if (!response.ok) {
    let message = `HTTP ${response.status}`;
    try {
      const payload = await response.json();
      message = payload.error?.message || message;
    } catch (_error) {}
    throw new Error(message);
  }
  return response.json();
}

async function fetchWithDemo(url, fallbackFactory, hasContent) {
  try {
    const payload = await fetchJSON(url);
    if (!hasContent || hasContent(payload)) {
      return payload;
    }
  } catch (_error) {}

  const fallback = fallbackFactory ? fallbackFactory() : null;
  if (fallback && (!hasContent || hasContent(fallback))) {
    return fallback;
  }
  return fetchJSON(url);
}

function formatNumber(value) {
  const numeric = Number(value || 0);
  return numeric.toFixed(2);
}

function hasListContent(value) {
  return Array.isArray(value) && value.length > 0;
}

function demoKey(market, ticker) {
  return `${market}:${ticker}`;
}

function formatPercent(value) {
  if (value === null || value === undefined) return "n/a";
  return `${Math.round(Number(value) * 100)}%`;
}

function formatDate(value) {
  if (!value) return "n/a";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "n/a";
  return date.toLocaleString();
}

function formatPrice(value) {
  const numeric = Number(value || 0);
  if (!Number.isFinite(numeric)) return "n/a";
  if (Math.abs(numeric) >= 1000) return numeric.toFixed(2);
  if (Math.abs(numeric) >= 1) return numeric.toFixed(3);
  return numeric.toFixed(6);
}

function stringifyValue(value) {
  if (value === null || value === undefined || value === "") return "";
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

function verificationText(detail, verifiedAt) {
  if (!detail) {
    return verifiedAt ? `Checked at ${formatDate(verifiedAt)}.` : "Verification detail is not available.";
  }
  return `Checked at ${formatDate(verifiedAt)}. Start ${formatPrice(detail.start_price)}, end ${formatPrice(detail.end_price)}, delta ${formatPrice(detail.delta)}.`;
}

function sparkline(points) {
  const width = 620;
  const height = 170;
  const padX = 12;
  const padY = 12;
  const closes = points.map((point) => Number(point.close || 0));
  const min = Math.min(...closes);
  const max = Math.max(...closes);
  const span = max - min || 1;
  const stepX = points.length === 1 ? 0 : (width - padX * 2) / (points.length - 1);

  const coords = closes.map((value, index) => {
    const x = padX + stepX * index;
    const normalized = (value - min) / span;
    const y = height - padY - normalized * (height - padY * 2);
    return [x, y];
  });

  const linePoints = coords.map(([x, y]) => `${x},${y}`).join(" ");
  const fillPoints = [
    `${padX},${height - padY}`,
    ...coords.map(([x, y]) => `${x},${y}`),
    `${coords[coords.length - 1][0]},${height - padY}`,
  ].join(" ");
  const last = coords[coords.length - 1];

  return `
    <svg class="sparkline" viewBox="0 0 ${width} ${height}" role="img" aria-label="Price path around the signal">
      <line class="sparkline-grid" x1="${padX}" y1="${height - padY}" x2="${width - padX}" y2="${height - padY}"></line>
      <line class="sparkline-grid" x1="${padX}" y1="${padY}" x2="${padX}" y2="${height - padY}"></line>
      <polygon class="sparkline-fill" points="${fillPoints}"></polygon>
      <polyline class="sparkline-line" points="${linePoints}"></polyline>
      <circle class="sparkline-marker" cx="${last[0]}" cy="${last[1]}" r="4"></circle>
    </svg>
  `;
}

function labelMarket(market) {
  if (market === "us_stock") return "US Stocks";
  if (market === "a_stock") return "A-Shares";
  if (market === "crypto") return "Crypto";
  return market;
}

init();
