# Product Brief: Blockchain Explorer

**Date:** 2025-10-29
**Author:** Hieu
**Status:** Draft for PM Review

---

## Executive Summary

The **Blockchain Explorer** is a production-grade Ethereum blockchain indexer and query platform built in Go, designed as a high-impact portfolio project to demonstrate advanced backend and data engineering competencies within a focused 1-week development sprint. This project addresses the challenge of effectively showcasing technical skills for senior-level roles: traditional portfolio projects often fail to impress because they lack real-world complexity, are too generic, or don't demonstrate depth in critical areas like distributed systems, concurrent data processing, and operational maturity.

The solution is a **lean but comprehensive blockchain explorer** targeting Ethereum Sepolia testnet, implementing a complete data pipeline from blockchain nodes to end-users. Core capabilities include parallel backfill of historical blocks (5,000 by default), real-time live-tail of new blocks and transactions, intelligent reorg handling, PostgreSQL-backed storage with optimized indexing, RESTful and WebSocket APIs, and a minimal single-page frontend. The system is designed for both technical correctness and observability, featuring Prometheus metrics, structured logging, Docker Compose deployment, and comprehensive documentation—demonstrating production-ready thinking from day one.

The primary users are **technical hiring managers and senior engineers** evaluating candidates for backend/data engineering positions, who need to quickly assess (within 15 minutes) whether a candidate possesses practical skills in Go concurrency, database optimization, API design, and real-time systems. Secondary users include **fellow developers** seeking to learn blockchain indexing patterns or reference implementations for similar projects. Success metrics include technical performance (5,000 blocks indexed in <5 minutes, p95 API latency <150ms), code quality (70%+ test coverage, clear modular architecture), and portfolio impact (90%+ evaluator setup success rate, positive interview conversion).

This project delivers exceptional ROI: a 1-week investment creating a differentiated portfolio asset that enables targeting senior backend roles ($120K-200K+ salary range), potentially reducing job search time by 30-50%. The MVP deliberately focuses on demonstrating the most valuable 20% of features—data ingestion, storage optimization, API design, real-time streaming—that showcase 80% of required competencies. Post-MVP opportunities include expanding into a production-grade blockchain data infrastructure platform, consulting asset, or educational resource, with potential applications in DeFi analytics, NFT infrastructure, and compliance tooling.

---

## Problem Statement

Demonstrating advanced technical competency in Golang and data engineering requires more than theory—it demands real-world projects that showcase production-grade skills in distributed systems, high-throughput data processing, and real-time data pipelines. However, traditional portfolio projects often fail to impress because they either:

1. **Lack Real-World Complexity**: Simple CRUD applications don't demonstrate ability to handle challenging architectural decisions around concurrency, performance optimization, and data consistency
2. **Are Too Generic**: Common projects (todo apps, blog platforms) make it difficult for candidates to stand out among peers
3. **Don't Show Data Engineering Depth**: Many projects miss opportunities to showcase skills in indexing strategies, query optimization, batch vs. streaming processing, and handling data at scale

**Current Pain Points:**
- **For Job Seekers**: Need a portfolio project that demonstrates expertise in Go concurrency patterns, database optimization, API design, and real-time systems within a reasonable timeframe (1 week)
- **For Technical Evaluators**: Difficulty quickly assessing a candidate's practical skills in systems design, data pipeline architecture, and production-ready code quality
- **Time Investment**: Building a sufficiently complex project from scratch typically requires weeks or months, delaying job search momentum

**Why Blockchain Explorer?**
Blockchain indexing is an ideal demonstration vehicle because it naturally requires:
- High-throughput data ingestion (parallel workers, bulk inserts)
- Complex state management (reorg handling, consistency guarantees)
- Real-time streaming (WebSocket live-tail)
- Query optimization (composite indexes, efficient lookups)
- Operational maturity (metrics, structured logging, containerization)

**Urgency:** [NEEDS CONFIRMATION: Is there a specific job opportunity timeline or skill demonstration deadline driving the 1-week constraint?]

---

## Proposed Solution

Build a **production-grade Ethereum blockchain explorer** in Go that demonstrates mastery of data engineering principles and systems design within a focused 1-week development sprint. The solution is deliberately scoped to showcase the most valuable technical skills without unnecessary scope creep.

**Core Approach:**

The explorer will implement a complete data pipeline from blockchain nodes to end-users:

1. **Intelligent Data Ingestion**
   - Parallel backfill pipeline to efficiently index historical blocks (last 5,000 by default)
   - Live-tail mechanism using WebSocket/RPC for real-time block tracking
   - Automatic reorg detection and recovery (up to 6 blocks deep)

2. **Optimized Data Storage**
   - PostgreSQL schema designed for blockchain data access patterns
   - Composite indexes for common query patterns (address lookups, block/tx searches)
   - Normalized structure supporting blocks, transactions, and event logs

3. **Comprehensive API Layer**
   - RESTful endpoints for historical queries
   - WebSocket streaming for real-time updates
   - Structured around common use cases (block lookup, transaction search, address history)

4. **Operational Excellence**
   - Prometheus metrics for observability
   - Structured logging for debugging
   - Docker Compose for reproducible deployment
   - Database migrations for schema versioning

**Key Differentiators:**

- **Focused Scope**: Unlike full-featured explorers (Etherscan, Blockscout), this targets the essential 20% of features that demonstrate 80% of technical competency
- **Clean Architecture**: Showcases Go best practices with clear separation of concerns (RPC client, ingestion, indexing, API, storage)
- **Performance-Oriented**: Designed for high throughput (backfill 5,000 blocks in minutes) and low latency (p95 < 150ms API responses)
- **Production-Ready**: Includes observability, error handling, and operational tooling from day one
- **Quick Setup**: Complete Docker Compose environment allows evaluators to see it running in minutes

**Target Chain:** Ethereum Sepolia testnet (free access, representative of mainnet behavior)

---

## Target Users

### Primary User Segment

**Technical Hiring Managers & Senior Engineers (Evaluators)**

**Profile:**
- Engineering managers and tech leads at companies hiring for backend/data engineering roles
- Senior engineers conducting technical interviews and code reviews
- CTOs and technical co-founders at startups evaluating potential team members
- Typically have 5-15+ years of experience and limited time for candidate evaluation

**Current Workflow:**
- Review dozens of candidate portfolios and GitHub profiles weekly
- Spend 5-15 minutes per candidate on initial screening
- Look for signals of production-readiness, not just ability to complete tutorials
- Value projects that demonstrate problem-solving in realistic scenarios

**Pain Points:**
- Difficulty distinguishing genuinely skilled candidates from those who completed bootcamp projects
- Need to quickly assess depth of knowledge in specific areas (concurrency, database optimization, API design)
- Generic projects (todo apps, blog platforms) provide weak signals
- Uncertainty about whether code is production-ready or just "works on my machine"

**Goals:**
- Identify candidates who can immediately contribute to production systems
- Assess architectural thinking and decision-making ability
- Evaluate code quality, testing practices, and operational awareness
- Make confident hiring decisions with limited evaluation time

**Success Criteria:**
- Can assess technical competency within 15 minutes (quick demo + code review)
- Clear evidence of skills in specific technical areas (Go, data pipelines, APIs)
- Confidence that candidate understands production systems beyond theory

### Secondary User Segment

**Fellow Developers & Technical Community (Learning & Reference)**

**Profile:**
- Backend developers learning Go or blockchain technology
- Data engineers exploring blockchain data pipeline architectures
- Students and bootcamp graduates looking for portfolio project inspiration
- Open-source contributors interested in blockchain infrastructure

**Current Behavior:**
- Search GitHub for example projects demonstrating specific patterns
- Study well-architected codebases to learn best practices
- Fork/adapt projects as starting points for their own work
- Share interesting technical implementations in developer communities

**Pain Points:**
- Many blockchain projects are either too simple (basic tutorials) or too complex (production systems with enterprise features)
- Difficulty finding clean, well-documented examples of specific patterns (reorg handling, parallel ingestion)
- Need reference implementations that balance completeness with comprehensibility

**Goals:**
- Learn how to structure a data-intensive Go application
- Understand blockchain indexing architecture and challenges
- See practical examples of Go concurrency patterns, error handling, and testing
- Build similar projects for their own portfolios

[NEEDS CONFIRMATION: Are there other user segments, such as potential clients for freelance/consulting work, or internal teams who might use this as a foundation?]

---

## Goals & Success Metrics

### Business Objectives

1. **Demonstrate Technical Competency** - Complete a portfolio project that clearly showcases advanced Golang and data engineering skills sufficient for senior backend/data engineering roles
   - Target: Project demonstrates mastery of 8+ key technical areas (concurrency, database design, API design, real-time systems, observability, testing, containerization, documentation)

2. **Accelerate Job Search Timeline** - Create interview-ready portfolio material within 1 week that can be discussed confidently in technical interviews
   - Target: Project completion and documentation within 7 days from start

3. **Differentiate from Competition** - Build a project complex enough to stand out among typical portfolio projects, but focused enough to complete quickly
   - Target: Positive feedback from 80%+ of technical evaluators who review the project

4. **Establish Technical Credibility** - Produce work that generates GitHub stars, social proof, and potential leads for opportunities
   - Target: [NEEDS CONFIRMATION: Specific goals for GitHub stars, job interviews secured, or other metrics?]

### User Success Metrics

**For Technical Evaluators:**

1. **Quick Assessment** - Evaluators can understand project scope, architecture, and code quality within 15 minutes
   - Measured by: Clear README with demo instructions, architecture diagrams, and code organization

2. **Confidence in Skills** - After reviewing the project, evaluators feel confident about candidate's ability to contribute to production systems
   - Measured by: Interview conversion rate, technical discussion depth, feedback quality

3. **Easy Verification** - Project can be run locally via Docker Compose with < 5 minutes of setup
   - Measured by: `docker compose up` success rate, time to first successful query

**For Learning Developers:**

1. **Educational Value** - Code and documentation provide clear learning value for understanding blockchain indexing or Go patterns
   - Measured by: GitHub stars, forks, positive issues/discussions, blog post shares

2. **Adaptability** - Project structure is clean enough to be forked and adapted for similar use cases
   - Measured by: Number of forks, derivative projects, questions in issues

### Key Performance Indicators (KPIs)

**Technical Performance KPIs:**
- **Backfill Speed**: Index 5,000 blocks in < 5 minutes on standard hardware
- **Live Tail Latency**: Stay within 2 seconds of network head during live indexing
- **API Response Time**: p95 latency < 150ms for standard queries
- **Reorg Recovery**: Automatically detect and heal from reorgs ≤ 6 blocks within 10 seconds
- **Uptime**: Indexer runs continuously for 24+ hours without errors or memory leaks

**Code Quality KPIs:**
- **Test Coverage**: Unit and integration test coverage > 70% of critical paths
- **Code Organization**: Clear modular structure with < 500 lines per file average
- **Documentation**: Comprehensive README, API docs, and architecture documentation
- **Error Handling**: All RPC errors, database errors, and edge cases handled gracefully

**Portfolio Impact KPIs:**
- **Setup Success Rate**: 90%+ of evaluators can run the demo successfully on first try
- **Interview Conversion**: Project discussion leads to positive technical assessment
- **Community Engagement**: [NEEDS CONFIRMATION: Target GitHub stars, forks, or community feedback?]

---

## Strategic Alignment & Financial Impact

### Financial Impact

**Development Investment:**
- **Time**: 1 week (7 days) of focused development
- **Infrastructure Costs**: Minimal
  - Ethereum Sepolia testnet (free)
  - Local PostgreSQL via Docker (no hosting costs during development)
  - Optional: $0-20 for RPC service if local node isn't used (Alchemy/Infura free tiers)
- **Total Direct Cost**: ~$0-20

**Expected ROI:**
- **Career Advancement**: Portfolio project enables targeting senior backend/data engineering roles (typically $120K-200K+ salary range)
- **Time-to-Hire Reduction**: Strong portfolio projects can reduce job search time by 30-50% (4-6 weeks faster job placement)
- **Interview Conversion**: Technical projects increase interview-to-offer conversion rates by demonstrating practical skills upfront
- **Consulting Opportunities**: [NEEDS CONFIRMATION: Is this being positioned for potential freelance/consulting work in blockchain data infrastructure?]

**Value Proposition:**
- **High Leverage**: 1 week investment → differentiated portfolio asset usable for years
- **Reusable Asset**: Can be referenced in multiple job applications, interviews, blog posts, and technical discussions
- **Learning ROI**: Skills developed (Go, PostgreSQL optimization, real-time systems) are broadly applicable beyond blockchain

**Opportunity Cost Analysis:**
- **Alternative: Generic Projects** - Lower differentiation, similar time investment
- **Alternative: Extended Project (1+ month)** - Higher quality but delays job search, risks scope creep
- **Alternative: No Portfolio Project** - Rely solely on resume/experience, miss opportunity to demonstrate current skills

### Company Objectives Alignment

**Note:** This is a personal portfolio project, so traditional company objectives are reframed as personal career objectives:

**Personal Career Objectives:**

1. **Technical Skill Validation** - Demonstrate up-to-date proficiency in modern backend technologies
   - Alignment: Project directly showcases Go, PostgreSQL, API design, containerization, and observability

2. **Market Positioning** - Position for senior backend or data engineering roles in blockchain/crypto, fintech, or data infrastructure companies
   - Alignment: Blockchain explorers are relevant to crypto companies, while the underlying patterns (data pipelines, indexing, real-time systems) apply broadly

3. **Professional Network Growth** - Build visibility in technical communities and among potential employers
   - Alignment: Open-source project creates opportunities for community engagement, blog posts, conference talks

4. **Continuous Learning** - Maintain technical edge through hands-on building
   - Alignment: Project provides deep experience with production-grade system design patterns

[NEEDS CONFIRMATION: Are there specific companies or roles being targeted that would influence the project's positioning?]

### Strategic Initiatives

**Short-term (1-3 months):**
1. **Complete MVP** - Finish all must-have features within 1-week timeline
2. **Documentation & Polish** - Create comprehensive README, API docs, architecture overview
3. **Public Launch** - Open-source on GitHub, share in relevant communities (r/golang, blockchain dev forums)
4. **Job Application Integration** - Feature project prominently in resume, LinkedIn, portfolio site

**Medium-term (3-6 months):**
1. **Technical Content Creation** - Write blog posts about architectural decisions, performance optimization, reorg handling
2. **Community Engagement** - Respond to issues, accept PRs, build reputation as maintainer
3. **Interview Leverage** - Use project as technical discussion anchor in interviews
4. **Potential Extensions** - Based on feedback, consider stretch goals (ERC-20 decoding, ClickHouse analytics)

**Long-term (6-12 months):**
1. **Foundation for Advanced Projects** - Potentially extend into full blockchain data platform if pursuing blockchain space
2. **Consulting Asset** - Use as reference implementation for potential consulting engagements
3. **Teaching/Mentoring** - Use project as teaching tool for other developers learning these patterns
4. **Career Pivot Enabler** - Serves as credential for transitioning into blockchain/crypto industry if desired

---

## MVP Scope

### Core Features (Must Have)

**1. Blockchain Indexer**
- **Backfill Pipeline**: Parallel workers to index last N blocks (default: 5,000) with bulk inserts for performance
- **Live Tail**: Real-time block monitoring via WebSocket/RPC connection to stay within 2s of network head
- **Reorg Handling**: Automatic detection and recovery from chain reorganizations up to 6 blocks deep
- **Rationale**: Core data ingestion capability; demonstrates concurrency, error handling, and state management

**2. PostgreSQL Data Layer**
- **Schema Design**: Normalized tables for blocks, transactions, and logs with proper foreign keys
- **Indexing Strategy**: Composite indexes optimized for common query patterns (address lookups, block/tx searches, log filtering)
- **Migrations**: Schema versioning for reproducible database setup
- **Rationale**: Showcases database design skills and understanding of query optimization

**3. REST API Endpoints**
- `GET /v1/blocks` - Paginated block listing
- `GET /v1/blocks/{height}` - Block details by height
- `GET /v1/txs/{hash}` - Transaction details and status
- `GET /v1/address/{addr}/txs` - Transaction history for address
- `GET /v1/logs` - Event log filtering with address/topic parameters
- `GET /v1/stats/chain` - Chain statistics (head block, indexed blocks, indexer lag)
- **Rationale**: Demonstrates RESTful API design and common blockchain query patterns

**4. WebSocket Streaming**
- `WS /v1/stream` - Real-time updates for new blocks and transactions
- Channel subscription model (clients can subscribe to specific event types)
- **Rationale**: Shows understanding of real-time systems and WebSocket protocol

**5. Minimal Frontend (SPA)**
- Live blocks ticker showing recent blocks as they're indexed
- Latest transactions table with pagination
- Basic search functionality (transaction hash, block number/hash, address)
- **Rationale**: Provides visual demonstration for evaluators; keeps UI simple to focus on backend

**6. Operational Infrastructure**
- **Docker Compose**: Complete local environment (PostgreSQL + Go services) with `docker compose up`
- **Prometheus Metrics**: Key operational metrics exposed at `/metrics` endpoint
  - `explorer_blocks_indexed_total` - Total blocks processed
  - `explorer_index_lag_blocks` - How far behind network head
  - `explorer_rpc_errors_total` - RPC connection issues
  - `explorer_api_latency_ms` - API response times
- **Structured Logging**: JSON-formatted logs for debugging and monitoring
- **Makefile**: Common operations (build, test, migrate, run)
- **Rationale**: Demonstrates production-readiness and operational thinking

### Out of Scope for MVP

**Deferred Features (Potential Phase 2):**
- ERC-20 token transfer decoding and indexing
- Contract source code verification
- Advanced analytics (gas price charts, transaction volume trends)
- ClickHouse or TimescaleDB for time-series analytics
- Address labeling/tagging system
- API key management and rate limiting
- GraphQL API
- Multi-chain support (Polygon, Optimism, etc.)
- Historical state queries ("what was this address's balance at block X?")
- Transaction trace/debug data
- Uncle/ommer block handling (Ethereum-specific)

**Explicitly Not Included:**
- Smart contract interaction/write operations
- Wallet functionality
- User authentication/accounts (beyond potential API keys)
- Mobile apps
- Advanced UI features (charts, graphs, dashboards)

**Rationale for Exclusions:**
These features would significantly increase scope without proportionally increasing demonstration of core competencies. The MVP focuses on the data pipeline and querying capabilities that showcase backend/data engineering skills.

### MVP Success Criteria

**Functional Requirements:**
- ✅ Successfully backfill 5,000 blocks in under 5 minutes
- ✅ Live tail maintains <2 second lag from network head
- ✅ Reorgs up to 6 blocks deep are detected and healed automatically
- ✅ All API endpoints return correct data with <150ms p95 latency
- ✅ WebSocket stream delivers new blocks/txs in real-time
- ✅ Frontend displays live data and search works correctly
- ✅ System runs continuously for 24+ hours without errors

**Technical Requirements:**
- ✅ Clean separation of concerns (RPC client, ingestion, indexing, API, storage layers)
- ✅ Comprehensive error handling for RPC failures, database errors, edge cases
- ✅ Unit and integration test coverage >70% for critical paths
- ✅ Prometheus metrics accurately reflect system state
- ✅ Structured logs provide debugging visibility

**Documentation Requirements:**
- ✅ README with clear setup instructions, architecture overview, demo script
- ✅ API.md documenting all endpoints with request/response examples
- ✅ Design.md explaining architecture decisions and reorg handling strategy
- ✅ Code comments explaining complex logic (reorg detection, parallel workers, etc.)

**Deployment Requirements:**
- ✅ `docker compose up` successfully starts all services
- ✅ Database migrations run automatically
- ✅ Services are configurable via environment variables
- ✅ Health check endpoints for monitoring

**Demo Requirements:**
- ✅ Evaluator can run the system locally in <5 minutes
- ✅ Demo script successfully shows key features (indexing, API queries, live updates, metrics)
- ✅ Project can be discussed confidently in technical interviews

---

## Post-MVP Vision

### Phase 2 Features

If the MVP proves successful in achieving its portfolio/career objectives, potential Phase 2 enhancements include:

**Enhanced Token Support:**
- ERC-20 token transfer decoding and indexing
- Token balance tracking per address
- Token holder analytics

**Advanced Analytics:**
- ClickHouse integration for time-series analytics
- Gas price trends and network activity charts
- Address behavior analysis (whale tracking, active addresses)

**Developer Experience:**
- GraphQL API alongside REST for flexible querying
- Webhook subscriptions for address monitoring
- API key management with usage tiers

**Data Quality:**
- Address labeling system (known exchanges, contracts, etc.)
- Contract source code verification integration
- ENS name resolution

**Scale & Performance:**
- Multi-chain support (Polygon, Optimism, Base, Arbitrum)
- Horizontal scaling for indexer workers
- Read replicas for query performance
- Caching layer (Redis) for frequently accessed data

### Long-term Vision

**1-2 Year Vision:**

Transform the MVP from a portfolio demonstration into a **production-grade blockchain data infrastructure platform** that serves multiple use cases:

1. **Developer Tool**: Reference implementation and library for building custom blockchain indexers
   - Extract core components into reusable Go packages
   - Provide plugin architecture for custom data transformations
   - Support for custom event decoding and indexing rules

2. **Self-Hosted Alternative**: Lightweight alternative to managed services (Alchemy, Infura, TheGraph) for teams wanting control over their infrastructure
   - Complete observability and monitoring stack
   - High availability deployment patterns
   - Cost-effective for moderate query volumes

3. **Analytics Platform Foundation**: Base layer for blockchain analytics and research
   - Pre-computed aggregations and metrics
   - Support for complex analytical queries
   - Data export capabilities for external analytics tools

4. **Educational Resource**: Become a go-to learning resource for blockchain infrastructure
   - Comprehensive documentation of design decisions
   - Tutorial series on building data-intensive systems
   - Conference talks and technical blog posts

### Expansion Opportunities

**Technical Expansion:**
- **Multi-Chain Aggregation**: Single API for querying across multiple EVM chains
- **State Reconstruction**: Historical state queries ("what was balance at block X?")
- **Transaction Simulation**: Pre-flight transaction simulation and gas estimation
- **MEV Detection**: Identify and analyze MEV (Maximum Extractable Value) activities
- **L2 Specialization**: Deep integration with Layer 2 networks (optimistic rollups, zk-rollups)

**Product Expansion:**
- **Blockchain Data API SaaS**: Hosted version with API keys and usage-based pricing
- **Consulting Service**: Help companies build custom blockchain data infrastructure
- **Training/Education**: Course on building high-performance data pipelines using this as example
- **Enterprise Features**: Multi-tenancy, role-based access, audit logs, SLAs

**Community Expansion:**
- **Open-Source Ecosystem**: Foster community contributions and plugins
- **Integration Partners**: Partnerships with wallet providers, analytics tools, DeFi protocols
- **Research Collaboration**: Work with blockchain researchers on data availability and analysis tools

**Market Expansion:**
- **DeFi Analytics**: Specialized indexing for DeFi protocols (Uniswap, Aave, etc.)
- **NFT Marketplace Infrastructure**: Optimized indexing for NFT collections and marketplaces
- **Gaming/Metaverse**: Support for blockchain gaming asset tracking
- **Compliance/Forensics**: Tools for regulatory compliance and blockchain forensics

[NEEDS CONFIRMATION: Which expansion directions align with your career interests—staying in individual contributor roles, moving toward consulting, or building a product?]

---

## Technical Considerations

### Platform Requirements

**Target Platforms:**
- **Backend Services**: Linux (Docker containers) - primary deployment target
- **Development**: macOS/Linux/Windows via Docker Compose for cross-platform compatibility
- **Frontend**: Modern web browsers (Chrome, Firefox, Safari, Edge) - latest 2 versions
- **API Consumers**: Any HTTP/WebSocket client (curl, Postman, programming language HTTP libraries)

**Performance Requirements:**
- **Backfill Speed**: Process 5,000 blocks in <5 minutes on standard hardware (4 CPU cores, 8GB RAM)
- **Live Tail Latency**: Maintain <2 second lag behind network head under normal conditions
- **API Response Time**: p95 latency <150ms for standard queries on indexed data
- **Concurrent Users**: Support 10-20 concurrent API users without degradation (sufficient for demo/small production use)
- **Database Size**: Optimize for ~5,000-50,000 blocks initially (scales to millions with proper indexing)

**Reliability Requirements:**
- **Indexer Uptime**: Run continuously for 24+ hours without crashes or memory leaks
- **Reorg Recovery**: Automatically detect and heal from reorganizations without manual intervention
- **Error Recovery**: RPC connection failures should retry with exponential backoff, not crash
- **Data Consistency**: Ensure transactional consistency when writing blocks/transactions to database

**Observability Requirements:**
- Prometheus metrics endpoint for monitoring
- Structured JSON logging for aggregation/searching
- Health check endpoints for load balancer/orchestration integration

### Technology Preferences

**Stack Overview:** Go 1.22+, PostgreSQL 16, Docker Compose, vanilla HTML/JavaScript

**Backend (Go 1.22+):**
- **HTTP Router**: `chi` - lightweight, idiomatic, good middleware support
- **Database Driver**: `pgx` - high-performance PostgreSQL driver with native Go types
- **Ethereum RPC**: `go-ethereum` (geth) client library for JSON-RPC communication
- **Metrics**: `prometheus/client_golang` for instrumentation
- **Logging**: Standard library `log/slog` for structured logging
- **Testing**: Standard library `testing` + `testify` for assertions

**Database (PostgreSQL 16):**
- **Why PostgreSQL**: Excellent for structured relational data, strong indexing, ACID guarantees, wide deployment knowledge
- **Migration Tool**: `golang-migrate` or embedded migrations in Go
- **Connection Pooling**: pgx native pool with configurable limits

**Infrastructure:**
- **Containerization**: Docker with multi-stage builds for optimal image size
- **Orchestration**: Docker Compose for local development (Kubernetes-ready architecture if needed later)
- **Process Management**: Separate containers for API server and indexer worker

**Frontend (Minimal):**
- **Framework**: None - vanilla HTML/JavaScript to minimize complexity
- **Styling**: Minimal CSS or lightweight framework (Tailwind CDN)
- **Real-time Updates**: Native WebSocket API
- **Build**: No build step required; serve static files directly

**Development Tools:**
- **Build**: Makefile for common commands (build, test, migrate, docker)
- **Linting**: `golangci-lint` for code quality
- **Formatting**: `gofmt` standard formatter
- **Version Control**: Git with conventional commits

### Architecture Considerations

**System Architecture (High-Level):**

```
[Ethereum RPC Node] <--JSON-RPC--> [Indexer Worker] --> [PostgreSQL]
                                          |
                                          v
                                   [API Server] <--HTTP/WS--> [Frontend SPA]
                                          |
                                          v
                                   [Prometheus /metrics]
```

**Modular Design:**

1. **RPC Client Layer** (`internal/rpc/`)
   - JSON-RPC communication with retry logic and exponential backoff
   - Connection pooling and rate limiting
   - Error classification (transient vs permanent failures)

2. **Ingestion Layer** (`internal/ingest/`)
   - Fetch blocks from RPC node
   - Decode and normalize blockchain data
   - Transform into internal domain models

3. **Indexing Layer** (`internal/index/`)
   - Batch processing for backfill (parallel workers)
   - Sequential processing for live tail (maintain ordering)
   - Reorg detection algorithm (compare parent hashes, mark orphaned blocks)
   - Transaction boundaries for atomicity

4. **Storage Layer** (`internal/store/pg/`)
   - PostgreSQL abstraction with interface for testability
   - Bulk insert optimization for backfill
   - Query builders for API layer
   - Migration management

5. **API Layer** (`internal/api/`)
   - REST endpoint handlers with pagination
   - WebSocket subscription management
   - Request validation and error responses
   - Metrics middleware

**Key Architectural Decisions:**

- **Separation of Indexer and API**: Run as separate processes for independent scaling and fault isolation
- **Worker Pool Pattern**: Parallel backfill workers with configurable concurrency
- **Eventual Consistency**: Accept brief inconsistency during reorgs (mark blocks as orphaned rather than delete)
- **Stateless API**: API servers hold no state; all data in PostgreSQL
- **Idempotent Operations**: Indexer can re-run blocks safely (upsert operations)

**Reorg Handling Strategy:**
- Track parent_hash for each block
- When new block's parent doesn't match DB head, walk backward to find fork point
- Mark orphaned blocks (set `orphaned = true`) rather than delete
- Re-process canonical chain from fork point forward

**Scalability Considerations:**
- Horizontal API scaling: Run multiple API servers behind load balancer
- Indexer scaling: Single writer for ordering, but parallel workers for backfill
- Database scaling: Read replicas for queries, write to primary for indexing
- Caching: Add Redis layer if query patterns show hotspots (not in MVP)

**Trade-offs Made:**
- **PostgreSQL vs Time-Series DB**: PostgreSQL chosen for simplicity and broad applicability; ClickHouse/TimescaleDB could improve analytics later
- **Single Chain**: Focus on Ethereum Sepolia; multi-chain adds complexity without demonstrating new patterns
- **Limited Frontend**: Minimal UI keeps focus on backend; full-featured UI would be Phase 2
- **No Authentication**: Simplifies demo; API keys and auth are Phase 2

---

## Constraints & Assumptions

### Constraints

**Time Constraints:**
- **Hard Deadline**: 1 week (7 days) from project start to functional MVP
- **Daily Time Allocation**: [NEEDS CONFIRMATION: Full-time focus (8+ hours/day) or part-time alongside other commitments?]
- **Trade-off Implication**: Must ruthlessly prioritize core features over polish; comprehensive testing comes after basic functionality

**Resource Constraints:**
- **Team Size**: Solo developer (no code reviews, pair programming, or division of labor)
- **Budget**: Essentially $0 (free tier RPC services, local development only)
- **Hardware**: Standard development machine (assumption: 4+ CPU cores, 8+ GB RAM, SSD)
- **External Services**: Limited to free tiers (Alchemy/Infura RPC free tier ~300K requests/day)

**Technical Constraints:**
- **Blockchain Data**: Dependent on external RPC node reliability and rate limits
- **Testnet Stability**: Sepolia testnet may have instability periods (lower priority than mainnet for node operators)
- **Data Volume**: Starting with 5,000 blocks limits historical analysis depth (Sepolia has ~50M+ blocks total)
- **Go Ecosystem**: Limited to Go-compatible libraries (no polyglot architecture in MVP)

**Knowledge Constraints:**
- **Learning Curve**: Time required to learn Ethereum JSON-RPC specifics, reorg handling patterns
- **Documentation Time**: Balance between code completion and documentation quality
- **Unknown Unknowns**: First time building this specific architecture may reveal unexpected complexity

**Operational Constraints:**
- **No Production Hosting**: MVP runs locally only; no cloud deployment, CDN, or managed services
- **No 24/7 Monitoring**: Development machine-based, not production-grade infrastructure
- **Limited Scalability Testing**: Cannot test true production load scenarios

### Key Assumptions

**About Users:**
- Technical evaluators will spend 10-20 minutes reviewing the project (README, code structure, running demo)
- Evaluators have Docker installed or are willing to install it
- Target audience values clean code and architecture over feature completeness
- Hiring managers prioritize depth in core competencies over breadth of features

**About Technology:**
- Go 1.22+ is available or installable on evaluator machines
- PostgreSQL 16 via Docker is sufficient; no need for managed database services
- Ethereum Sepolia testnet will remain accessible and relatively stable during development
- Free tier RPC services (Alchemy/Infura) provide adequate throughput for demo purposes

**About the Market:**
- Backend/data engineering roles continue to value Go expertise
- Blockchain/crypto industry remains relevant for job opportunities
- Portfolio projects significantly influence hiring decisions for senior roles
- Open-source projects on GitHub provide credibility signals

**About Development:**
- Core blockchain indexing patterns can be implemented in 1 week with focused effort
- Reorg handling logic won't require extensive testing (testnet reorgs are infrequent)
- Docker Compose setup will work reliably across different developer environments
- Third-party libraries (go-ethereum, pgx, chi) are stable and well-documented

**About Scope:**
- 5,000 blocks is sufficient to demonstrate indexing capabilities
- Basic REST API + WebSocket is enough to show API design skills
- Minimal frontend is acceptable (evaluators focus on backend)
- Unit tests for critical paths are sufficient (don't need 90%+ coverage)

**About Success Metrics:**
- Project completion within 1 week is realistic and sufficient for goals
- Clean architecture and documentation will differentiate from typical portfolio projects
- Technical depth in specific areas (concurrency, database optimization) is more valuable than surface-level breadth
- GitHub repository with good README will be discovered by relevant audiences

**Validation Needed:**
- [NEEDS CONFIRMATION: Is the 1-week timeline based on full-time availability or part-time?]
- [NEEDS CONFIRMATION: Have you built similar data pipeline systems before, or is this exploring new territory?]
- [NEEDS CONFIRMATION: Is there existing Go/blockchain experience to build on, or learning from scratch?]

---

## Risks & Open Questions

### Key Risks

**Technical Risks:**

1. **Complexity Underestimation** (Likelihood: Medium, Impact: High)
   - Risk: Blockchain reorg handling or parallel worker coordination proves more complex than anticipated
   - Impact: Timeline slips, MVP scope must be reduced
   - Mitigation: Start with riskiest components (reorg logic, parallel workers) first; have fallback simpler approaches

2. **Third-Party Service Reliability** (Likelihood: Medium, Impact: Medium)
   - Risk: Free tier RPC services are rate-limited or unreliable during development
   - Impact: Delays in testing, difficulty demonstrating live functionality
   - Mitigation: Set up multiple RPC providers (Alchemy, Infura, public nodes); implement robust retry logic

3. **Performance Targets Missed** (Likelihood: Low, Impact: Medium)
   - Risk: Cannot achieve 5-minute backfill or <150ms API latency on standard hardware
   - Impact: Less impressive demo, need to adjust success criteria
   - Mitigation: Profile early, optimize database indexes, use bulk inserts, test on target hardware

4. **Testnet Instability** (Likelihood: Low, Impact: Low)
   - Risk: Sepolia testnet has downtime or anomalous behavior during development
   - Impact: Difficulty testing, potential demo issues
   - Mitigation: Have backup testnet option (Holesky), design system to be testnet-agnostic

**Schedule Risks:**

5. **Scope Creep** (Likelihood: High, Impact: High)
   - Risk: Adding "just one more feature" derails timeline
   - Impact: Project incomplete at 1 week, misses job search window
   - Mitigation: Strict adherence to must-have list, explicitly defer all stretch goals, daily scope check

6. **Learning Curve Delays** (Likelihood: Medium, Impact: Medium)
   - Risk: Unfamiliarity with Ethereum specifics (reorgs, gas, logs) causes delays
   - Impact: Less time for implementation, rushed code quality
   - Mitigation: Front-load research (day 1), use existing examples (go-ethereum docs), ask for help in communities

**Portfolio/Career Risks:**

7. **Insufficient Differentiation** (Likelihood: Low, Impact: High)
   - Risk: Project doesn't stand out enough from other portfolio projects
   - Impact: Doesn't achieve career advancement goals
   - Mitigation: Focus on production-ready elements (metrics, logging, Docker), emphasize architecture in README

8. **Demo Failures** (Likelihood: Medium, Impact: High)
   - Risk: Technical evaluators can't run the project successfully (environment issues, unclear docs)
   - Impact: Negative impression despite code quality
   - Mitigation: Test setup on multiple machines/OS, detailed README, troubleshooting section, demo video as backup

### Open Questions

**Technical Questions:**
- What's the optimal worker pool size for backfill? (Need to benchmark: too few = slow, too many = RPC rate limits)
- Should we store raw block JSON or only parsed fields? (Trade-off: storage size vs. flexibility for future queries)
- How to handle uncle/ommer blocks on Ethereum? (May be out of scope if infrequent on Sepolia)
- What pagination strategy for large result sets? (Offset/limit vs. cursor-based)

**Product Questions:**
- What documentation format resonates most with technical evaluators? (Long-form design docs vs. inline code comments vs. video walkthrough)
- Should we optimize for GitHub stars or private portfolio review? (Affects whether to market/promote publicly)
- Is a demo video more effective than live demo for asynchronous evaluation? (Many recruiters review async)
- What's the right balance of test coverage? (70% is target, but what's the minimum acceptable?)

**Career Strategy Questions:**
- Should this be positioned for blockchain-specific roles or general backend roles? (Affects README framing)
- Is open-sourcing immediately optimal, or better to use privately for job applications first? (IP/timing considerations)
- Should contributions be accepted from community, or keep as solo project? (Affects maintainer burden)
- Is a companion blog post series worth the time investment? (Extends timeline but increases visibility)

**Operational Questions:**
- How to demo live-tail functionality if Sepolia has slow block times (12s)? (May need demo mode with simulated blocks)
- What's the ideal block count for demo? (5,000 may take too long to index during live demo)
- Should we provide a pre-seeded database for instant demo? (Trade-off: convenience vs. demonstrating indexing speed)
- How to handle breaking changes in go-ethereum or pgx libraries during 1-week dev window?

### Areas Needing Further Research

**Before Implementation Starts:**
1. **Ethereum Reorg Patterns**: Research how frequently reorgs occur on Sepolia, typical depth, and best practices for detection
2. **Performance Benchmarks**: Find baseline metrics for similar Go+PostgreSQL data pipelines to validate 5-minute target
3. **RPC Provider Comparison**: Test Alchemy vs. Infura vs. public nodes for reliability and rate limits
4. **Go-Ethereum Library**: Review documentation and examples for block fetching, WebSocket subscriptions, and error handling

**During Implementation:**
1. **Database Index Optimization**: Research optimal composite index strategies for blockchain query patterns (may need iteration)
2. **Worker Pool Patterns**: Investigate Go worker pool best practices and bounded concurrency patterns
3. **WebSocket Scaling**: Understand goroutine per connection costs and potential optimization strategies
4. **Error Taxonomy**: Classify different RPC errors (transient network vs. permanent) for appropriate retry logic

**For Future Iterations:**
1. **Alternative Databases**: Research ClickHouse and TimescaleDB for blockchain analytics use cases
2. **Observability Best Practices**: Study production blockchain indexer monitoring and alerting strategies
3. **Multi-Chain Patterns**: How do production explorers (Blockscout, Subscan) handle multiple chains?
4. **State Sync Strategies**: How to efficiently catch up if indexer is offline for extended period?

**Competitive Research:**
1. **Reference Implementations**: Review open-source blockchain explorers for architectural patterns
   - Blockscout (Elixir)
   - Etherscan-like projects
   - TheGraph indexer architecture
2. **Portfolio Benchmarking**: Analyze successful technical portfolios to understand presentation and documentation best practices
3. **Job Posting Analysis**: Review backend/data engineering job postings to ensure project aligns with in-demand skills

---

## Appendices

### A. Research Summary

**Source Document:** `blockchain_explorer_mvp.md`

The primary input for this Product Brief was a comprehensive technical MVP specification document that outlined:

**Key Technical Specifications:**
- 7-day development timeline with daily breakdown
- Target blockchain: Ethereum Sepolia testnet
- Core objectives: Backfill 5,000 blocks, live-tail new blocks, index into PostgreSQL, provide REST + WebSocket APIs, minimal SPA frontend

**Architecture Insights:**
- Modular folder structure separating concerns (cmd/, internal/, migrations/, web/)
- Focus on production-ready elements: Docker Compose, Prometheus metrics, structured logging, Makefile
- Emphasis on performance: parallel workers for backfill, bulk inserts, <150ms API latency

**Data Model:**
- Well-defined PostgreSQL schema for blocks, transactions, and logs
- Consideration for reorg handling with `orphaned` flag on blocks
- Composite indexes for common query patterns

**Stretch Goals Identified:**
- ERC-20 transfer decoding
- ClickHouse analytics
- Address labeling
- API key management and rate limiting

**Acceptance Criteria:**
- Clear performance targets (5,000 blocks in minutes, <2s live-tail lag, p95 <150ms API latency)
- Emphasis on reproducibility and clear documentation

This technical specification provided the "what" and "how" of the project. This Product Brief adds the strategic "why," user perspectives, success metrics, and portfolio positioning that will inform both implementation priorities and documentation approach.

### B. Stakeholder Input

**Primary Stakeholder:** Hieu (Project Owner/Developer)

**Context Gathered:**
- Project name: "Blockchain Explorer"
- Output location preference: `{project-root}/docs`
- Collaboration approach: YOLO mode (draft then refine)
- Existing technical specification document as primary input

**Stakeholder Objectives** (inferred from project structure and goals):
- Demonstrate advanced Golang and data engineering competency
- Create portfolio material suitable for senior backend/data engineering positions
- Complete project within aggressive 1-week timeline
- Produce work that stands out from typical portfolio projects

**Key Requirements Emphasized:**
- Production-grade quality (observability, error handling, containerization)
- Clear documentation and reproducible setup
- Performance optimization opportunities
- Clean architectural patterns

**Areas Requiring Confirmation:**
Several aspects were flagged with `[NEEDS CONFIRMATION]` tags throughout the brief:
- Specific job timeline or urgency driving the 1-week constraint
- Full-time vs part-time availability during development week
- Target companies or specific roles influencing positioning
- Prior experience with similar systems (Go, blockchain, data pipelines)
- Career direction preferences (IC vs consulting vs product building)
- Community engagement goals (GitHub stars, blog posts, etc.)

### C. References

**Technical Documentation:**
1. **Ethereum JSON-RPC Specification** - https://ethereum.github.io/execution-apis/api-documentation/
   - Primary reference for RPC methods and data structures

2. **Go-Ethereum (Geth) Documentation** - https://geth.ethereum.org/docs
   - Go library for Ethereum interaction

3. **PostgreSQL 16 Documentation** - https://www.postgresql.org/docs/16/
   - Database features, indexing strategies, performance tuning

4. **PGX Documentation** - https://github.com/jackc/pgx
   - PostgreSQL driver and toolkit for Go

**Architecture References:**
5. **Blockscout** - https://github.com/blockscout/blockscout
   - Open-source blockchain explorer (Elixir) for architectural patterns

6. **TheGraph Protocol** - https://thegraph.com/docs
   - Decentralized blockchain indexing protocol for patterns and best practices

7. **Etherscan** - https://etherscan.io
   - Industry-standard blockchain explorer for feature reference

**Best Practices:**
8. **Go Concurrency Patterns** - https://go.dev/blog/pipelines
   - Official Go blog on concurrent pipeline patterns

9. **Prometheus Best Practices** - https://prometheus.io/docs/practices/
   - Instrumentation and metric naming conventions

10. **Docker Compose Documentation** - https://docs.docker.com/compose/
    - Multi-container application orchestration

**Portfolio & Career Resources:**
11. **Technical Portfolio Best Practices** - Various blog posts and guides on effective technical portfolio presentation
12. **Job Market Research** - Backend and data engineering role requirements from job boards (LinkedIn, Indeed, relevant company career pages)

**Project Source:**
13. **blockchain_explorer_mvp.md** - Local project specification document (primary input for this brief)

---

_This Product Brief serves as the foundational input for Product Requirements Document (PRD) creation._

_Next Steps: Handoff to Product Manager for PRD development using the `workflow prd` command._
