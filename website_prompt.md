# Website Creation Prompt for Ratelord Project

Create a comprehensive website for the Ratelord project that serves as the public-facing documentation and landing page. The website should be modern, clean, and informative, targeting developers, system administrators, and organizations interested in constraint management for autonomous systems.

## Project Overview (Include in Prompt)

Ratelord is a local-first constraint control plane for agentic and human-driven software systems. It provides a "sensory organ" and "prefrontal cortex" for resource availability and budget planning, enabling systems to negotiate, forecast, govern, and adapt under constraints like API rate limits, token budgets, and monetary spend.

Key features:
- Local-first, zero-ops daemon architecture
- Event-sourced and replayable system
- Predictive modeling with time-to-exhaustion forecasts
- Intent negotiation protocol for agents
- Hierarchical constraint graph modeling
- Provider-agnostic design (starts with GitHub API)

## Website Structure Requirements

### 1. Homepage
- Hero section with project tagline: "Budget-Literate Autonomy for Agentic Systems"
- Brief overview (3-4 paragraphs) explaining the problem and solution
- Key benefits/features callouts
- Call-to-action buttons: "Get Started", "View Documentation", "GitHub Repo"
- Architecture diagram or visual representation
- Current status badge (e.g., "Active Development - Phase 5")

### 2. Documentation Section
- **Getting Started**: Installation instructions, basic setup, first identity registration
- **Architecture**: Detailed breakdown of components (daemon, storage, clients, providers)
- **Concepts**: Constraint graphs, identities, scopes, pools, forecasts, policies
- **API Reference**: HTTP endpoints, intent negotiation protocol
- **Configuration**: Policy files, provider setup
- **Troubleshooting**: Common issues, logs, debugging

### 3. About Section
- **Vision**: The problem of "blind" agents, solution of budget-literacy
- **Principles**: Local-first, daemon authority, event sourcing, prediction, intent negotiation
- **Constitution**: Immutable laws governing the system
- **Roadmap**: Current phase, upcoming features

### 4. Community/Contributing
- How to contribute (code, docs, testing)
- Development setup
- Issue tracking on GitHub
- Contact information

## Technical Requirements

### Design
- Responsive design (mobile-friendly)
- Dark/light mode toggle
- Clean typography (use a monospace font for code)
- Consistent color scheme (consider blues/greens for tech/trust)
- Fast loading (optimize images, use CDN if needed)

### Content Integration
- Embed or link to all markdown documentation files
- Syntax highlighting for code examples
- Interactive elements where appropriate (e.g., expandable code blocks)
- Search functionality across docs

### Implementation
- Use modern web technologies (React/Next.js recommended for dynamic content)
- Static generation where possible for performance
- SEO optimized with proper meta tags
- Accessible (WCAG 2.1 AA compliance)

## Key Content to Include

### From PROJECT_CONTEXT.md
- Core problem statement
- Foundational principles
- System architecture overview
- Data philosophy
- Agent interaction contract

### From VISION.md
- The "Blind" Agent problem
- Budget-Literate Autonomy solution
- Strategic goals

### From ARCHITECTURE.md
- System invariants
- Component responsibilities
- Core abstractions (constraint graph, identity layer, etc.)
- Dataflow explanation
- Operational modes

### From CONSTITUTION.md
- Authority of the daemon
- Event sourcing requirements
- Negotiation mandate
- Scoping rigor

### Current Status
- Phase 5: Remediation (implementation mostly complete)
- Go-based daemon with SQLite storage
- TUI and Web UI clients
- Mock provider for testing
- GitHub API integration planned

## Call-to-Actions
- "Try the Demo" (if available)
- "Read the Docs"
- "Join the Discussion" (GitHub issues/discussions)
- "Contribute Code"

## Additional Elements
- Footer with links to GitHub, license (assume MIT or similar)
- Last updated timestamps on docs
- Version information
- Social sharing buttons

Ensure the website positions Ratelord as a serious, production-ready tool for managing constraints in autonomous systems, while being approachable for developers to try and contribute to.