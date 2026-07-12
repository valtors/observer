# Observer - GitHub Go-To-Market Strategy

## Product

Observer is a transparent MCP proxy for agent observability. It sits between any MCP client (Claude Desktop, Cline, Goose, Codex) and MCP servers, logging every tool call to SQLite and exposing trace history through 4 injected MCP tools (trace.history, trace.stats, trace.search, trace.replay).

## Problem (validated)

1665 discussions analyzed across 8 major AI agent communities (MCP, Codex, Cline, Goose, Anthropic SDK, OpenAI Python, shadcn, agent-orchestrator). The #1 common problem is agent observability - nobody can see what their agent is doing. 194 discussions, 17+ high-engagement threads. The #2 problem is tool bloat/token overhead (190 mentions).

Observer solves both.

## Target Audience

- AI agent developers using MCP servers with Claude Desktop, Cline, Goose, Codex, or any MCP client
- Teams debugging agent behavior in production
- Developers concerned about token cost from tool bloat
- Open source contributors looking for a small, well-structured Go project

## Phase 1: Foundation (Week 1 - DONE)

**Goal:** Repo is ready for visitors. Code builds. CI passes. Issues are open.

**Done:**
- [x] Repo created with description, topics, license
- [x] README with clear value prop, architecture diagram, quick start
- [x] CONTRIBUTING.md with AI agent contribution guide
- [x] Issue templates (bug, feature, config)
- [x] PR template with checklist
- [x] CI workflow (build + test on push/PR)
- [x] 5 issues open (#1 store tests, #2 tool filtering, #3 SSE transport, #10 security hardening, #11 regression tests)
- [x] Discussions enabled
- [x] Go source pushed and builds locally
- [x] Proxy tested end-to-end (initialize, tools/list with trace tools injected, echo call logged, trace.history/stats/search all return data)
- [x] Tested with Relay as real MCP server target (40 tools proxied, 4 trace tools injected, all calls logged)
- [x] Topics set: mcp, model-context-protocol, observability, agent-observability, ai-agents, proxy, tracing, developer-tools, go, sqlite
- [x] Security hardening: metadata-only trace by default, 0600 DB perms, session-scoped queries, OBSERVER_RAW_PAYLOAD opt-in, OBSERVER_REDACT_PATTERNS
- [x] Independently verified by external tester (caioribeiroclw-pixel) - all security boundaries hold

## Phase 2: Community Outreach (Week 1-2 - IN PROGRESS)

**Goal:** Drive initial stars, forks, and contributors from the communities that asked for this.

### 2a. Discussion thread replies (3/12 done)

Go back to the 17+ high-engagement discussion threads where people complained about agent observability. Reply with genuine, helpful responses that mention Observer as a solution we built in response to the problem.

**Completed:**
1. MCP #269 (17 comments) - "Adding OpenTelemetry Trace Support to MCP" - replied, got external tester who verified security fix
2. MCP #2036 (7 comments) - "handling tool bloat" - replied with tool filtering approach
3. Codex #28114 - "Where can I see subagent logs?" - replied with transparent proxy approach

**Remaining:**
1. MCP #269 (17 comments) - "Adding OpenTelemetry Trace Support to MCP"
2. MCP #845 (26 comments) - "Semantic tool auto-filtering at MCP Client/Proxy"
3. MCP #1567 (38 comments) - "Primitive Groups for Tools, Resources, Prompts"
4. MCP #2812 (7 comments) - "tool schema token overhead"
5. MCP #2999 - "agentsense - transparent MCP proxy for protocol-level tracing"
6. Codex #28114 - "Where can I see subagent logs?"
7. Codex #22749 - "Orchestration for parallel project-level coordination"
8. Goose #1685 - "What Just Happened / diff visibility"
9. Goose #6418 - "Diagnostic bundle with tool definitions"
10. Cline #786 - "Logging input/output from agent to LLM"
11. Cline #489 - "Multi-Agent Framework with Automated Provider Selection"
12. Anthropic #1419 - "Multi-Agent Memory Architecture"

**Rules for replies:**
- Lead with the problem, not the product. Acknowledge their pain.
- Mention Observer naturally: "we ran into the same problem and built a transparent MCP proxy that logs every tool call"
- Link to the repo, not to a landing page
- Do not post in more than 3 threads per day to avoid looking spammy
- Be genuinely helpful - add technical value to the discussion beyond just the link

### 2b. awesome-mcp-servers PR

Submit a PR to awesome-mcp-servers listing Observer under a new "Observability" or "Proxy" category. This is a high-traffic list that MCP users browse for new servers.

### 2c. X/LinkedIn post

Post about Observer from Tamish's account. Focus on the research angle: "we analyzed 1665 discussions across 8 AI agent communities and found the #1 problem is agent observability. So we built Observer."

This is a compelling narrative because it shows data-driven product development, not just another tool.

### 2d. Hacker News (if Tamish creates account)

Title: "Show HN: Observer - transparent MCP proxy for agent observability"
Body: The research story + how it works + link to repo.

HN loves data-driven product stories. The "we analyzed 1665 discussions" angle is strong.

## Phase 3: Contributor Funnel (Week 2-3)

**Goal:** Convert visitors into contributors.

### 3a. Good first issues

We have 2 good first issues (#1 store tests, #2 tool filtering). These are genuinely simple and well-scoped. When someone comments on them, respond fast and offer help.

### 3b. Discussion seeding

Create 3-4 discussions in the Observer repo to seed conversation:
- "What trace tools would be most useful?" (get feature requests)
- "Which MCP servers are you using Observer with?" (get usage data)
- "Token overhead: how many tools does your agent expose?" (data collection)

### 3c. Contributor recognition

When someone opens a PR, review it within 24 hours. Merge fast. Thank them publicly. This creates a positive contributor experience that leads to word of mouth.

## Phase 4: Ecosystem Expansion (Week 3-4)

**Goal:** Observer becomes a known name in the MCP ecosystem.

### 4a. Add SSE transport

Issue #3 is open for this. Either wait for a contributor or do it ourselves. SSE support makes Observer useful for remote/cloud agents, not just local.

### 4b. OpenTelemetry export

Add a tool that exports trace data to OpenTelemetry-compatible backends (Jaeger, Grafana, Honeycomb). This connects Observer to the existing observability ecosystem and makes it relevant to DevOps/SRE teams, not just agent developers.

MCP #269 specifically asks for OpenTelemetry support. If we ship it, that's a direct response to a 17-comment thread.

### 4c. Tool recommendation engine

Instead of just filtering tools, recommend which tools the agent should use based on the current context. This addresses MCP #845 (26 comments, "Semantic tool auto-filtering").

### 4d. npm wrapper

Create a simple npm package (`observer-mcp`) that wraps the Go binary and makes it installable via `npx observer-mcp`. This removes the Go dependency for end users and makes Observer accessible to the JS/TS ecosystem.

## Phase 5: Authority Building (Ongoing)

**Goal:** Tamish is recognized as a thought leader in the MCP/agent observability space.

### 5a. Blog posts (GitHub Discussions or external)

- "What 1665 AI agent discussions taught us about observability"
- "How to debug MCP tool calls with Observer"
- "Token overhead in MCP: why tool filtering matters"

### 5b. Technical commentary

Continue contributing to discussions in MCP, Codex, Cline, Goose repos with technical insights. Not promoting Observer every time - just being a knowledgeable voice in the community.

### 5c. Conference/meetup talks

If Tamish is interested, propose talks at Go meetups or AI dev meetups in Bangalore on "Building MCP infrastructure with Go" or "Agent observability patterns".

## Metrics to Track

- Stars (target: 50 in week 1, 200 in month 1, 1000 in 3 months)
- Forks (target: 5 in week 1, 20 in month 1)
- Contributors (target: 3 in week 2, 10 in month 1)
- Discussion thread replies mentioning Observer (organic mentions from others)
- npm downloads (after npm wrapper ships)
- PRs from external contributors

## Anti-patterns to avoid

- Do not post Observer link in more than 3 discussions per day
- Do not post in discussions that are not about observability or tool bloat
- Do not use marketing language. Talk like a developer, not a marketer.
- Do not compare to specific competitors by name unless asked
- Do not create fake stars or fake engagement. Organic only.
- Do not rush Phase 2 before Phase 1 is solid (CI green, code builds, issues open)
