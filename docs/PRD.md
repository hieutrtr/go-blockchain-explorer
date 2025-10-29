# Blockchain Explorer Product Requirements Document (PRD)

**Author:** Hieu
**Date:** 2025-10-29
**Project Level:** Level 2 (Small complete system)
**Project Type:** Backend service/API with minimal frontend
**Target Scale:** 12-15 stories across 1-2 epics, 7-day implementation sprint

---

## Description, Context and Goals

### Project Description

The **Blockchain Explorer** is a production-grade Ethereum blockchain indexer and query platform built in Go, designed to demonstrate advanced backend and data engineering competencies within a focused 1-week development sprint.

The system implements a complete data pipeline from blockchain nodes to end-users, featuring:

- **Intelligent Data Ingestion**: Parallel backfill of historical blocks (5,000 by default), real-time live-tail of new blocks and transactions, and automatic reorg detection and recovery (up to 6 blocks deep)

- **Optimized Data Storage**: PostgreSQL-backed storage with composite indexes designed for blockchain query patterns, normalized schema for blocks, transactions, and event logs

- **Comprehensive API Layer**: RESTful endpoints for historical queries, WebSocket streaming for real-time updates, structured around common blockchain use cases (block lookup, transaction search, address history)

- **Operational Excellence**: Prometheus metrics for observability, structured JSON logging, Docker Compose deployment, database migrations, and health checks

- **Minimal Frontend**: Single-page application with live blocks ticker, recent transactions table, and basic search functionality (transaction hash, block number, address)

The solution targets Ethereum Sepolia testnet and emphasizes production-ready patterns including clean architecture, performance optimization (5,000 blocks in <5 minutes, p95 API latency <150ms), comprehensive error handling, and reproducible deployment.

### Deployment Intent

**Demo/Portfolio Project** - Runs locally via Docker Compose for demonstration and evaluation purposes. Primary deployment target is development machine with the goal of showcasing technical competency to hiring managers, senior engineers, and technical evaluators. The system must be reproducible across different environments and demonstrate production-ready patterns despite being scoped for portfolio use.

### Context

Traditional portfolio projects often fail to demonstrate advanced technical competency because they're either too simple (basic CRUD apps) or too generic (todo lists, blog platforms). This Blockchain Explorer addresses that gap by implementing a data-intensive system that naturally requires sophisticated patterns: high-throughput parallel processing, real-time streaming, complex state management (reorg handling), database optimization, and operational maturity. The 1-week timeline constraint forces ruthless prioritization of the 20% of features that demonstrate 80% of competencies, making it both achievable and impressive. Success in this project enables targeting senior backend/data engineering roles ($120K-200K+ range) with concrete evidence of production-ready skills.

### Goals

1. **Demonstrate Technical Competency** - Complete a portfolio project that clearly showcases advanced Golang and data engineering skills sufficient for senior backend/data engineering roles, with mastery demonstrated across 8+ key technical areas (concurrency, database design, API design, real-time systems, observability, testing, containerization, documentation)

2. **Accelerate Job Search Timeline** - Create interview-ready portfolio material within 7 days that can be discussed confidently in technical interviews, with clear documentation enabling evaluators to assess competency within 15 minutes

3. **Achieve Production-Grade Quality** - Build a system with observability (Prometheus metrics), reliability (error handling, reorg recovery), performance (5,000 blocks in <5 minutes, p95 API latency <150ms), and operational maturity (Docker Compose, health checks, structured logging) that differentiates from typical portfolio projects

## Requirements

### Functional Requirements

**FR001: Historical Block Indexing** - System must be able to backfill and index historical blockchain blocks from Ethereum Sepolia testnet, configurable to fetch last N blocks (default: 5,000), processing them in parallel with bulk insert optimization for performance

**FR002: Real-Time Block Monitoring** - System must continuously monitor the blockchain for new blocks via WebSocket/RPC connection and index them in real-time, maintaining a lag of less than 2 seconds behind the network head under normal conditions

**FR003: Chain Reorganization Handling** - System must automatically detect chain reorganizations (reorgs) up to 6 blocks deep by comparing parent hashes, mark orphaned blocks appropriately (set orphaned flag), and re-process the canonical chain from the fork point forward

**FR004: Block Query API** - System must provide REST API endpoints to query blocks by height or hash, returning block details including height, hash, parent hash, miner address, gas used, transaction count, and timestamp

**FR005: Transaction Query API** - System must provide REST API endpoints to query transactions by hash, returning transaction details including block height, from/to addresses, value, fee, success status, and gas information

**FR006: Address Transaction History** - System must provide REST API endpoint to retrieve transaction history for a given Ethereum address, with pagination support for addresses with many transactions

**FR007: Event Log Filtering** - System must provide REST API endpoint to query event logs with filtering by contract address and event topics, supporting blockchain event analysis use cases

**FR008: Chain Statistics API** - System must provide REST API endpoint that returns current chain statistics including latest indexed block height, total blocks indexed, current indexer lag, and system health indicators

**FR009: Real-Time Event Streaming** - System must provide WebSocket API endpoint that allows clients to subscribe to real-time updates for new blocks and transactions as they are indexed

**FR010: Block Search Interface** - Frontend must provide search functionality allowing users to look up blocks by height or hash, transactions by hash, and addresses for transaction history

**FR011: Live Block Display** - Frontend must display a live-updating ticker of recently indexed blocks with basic information (height, hash, timestamp, transaction count)

**FR012: Recent Transactions View** - Frontend must display a table of recent transactions with pagination, showing key transaction details (hash, from/to addresses, value, block height)

**FR013: System Metrics Exposure** - System must expose Prometheus metrics at `/metrics` endpoint including blocks indexed, indexer lag, RPC errors, and API response times for operational monitoring

**FR014: Structured Logging** - System must emit structured JSON logs for all significant events (block processing, API requests, errors, reorg detection) to enable debugging and operational visibility

### Non-Functional Requirements

**NFR001: Performance - Backfill Speed** - The indexer must be capable of backfilling 5,000 blocks in under 5 minutes on standard hardware (4 CPU cores, 8GB RAM), demonstrating efficient parallel processing and bulk database operations

**NFR002: Performance - API Latency** - REST API endpoints must maintain p95 response latency under 150ms for standard queries on indexed data, demonstrating proper database indexing and query optimization

**NFR003: Reliability - Continuous Operation** - The indexer must run continuously for 24+ hours without crashes, memory leaks, or degradation, with automatic retry logic for transient RPC failures using exponential backoff

**NFR004: Reproducibility - Easy Setup** - The complete system must be runnable via `docker compose up` with database migrations executing automatically, enabling evaluators to run a working demo in under 5 minutes

**NFR005: Code Quality - Testability** - Critical code paths must have unit and integration test coverage exceeding 70%, with clear modular architecture enabling easy testing of individual components

## User Journeys

### Journey 1: Technical Evaluator Reviews Portfolio Project

**Actor:** Senior Engineer evaluating candidate for backend role

**Scenario:** Evaluator has 15 minutes to assess technical competency

**Steps:**
1. Evaluator clones repository and reads README
2. Runs `docker compose up` to start the system
3. Observes indexer logs showing parallel block processing and performance metrics
4. Opens frontend in browser to see live blocks appearing in real-time
5. Tests API endpoints using provided examples (query block, transaction, address history)
6. Views `/metrics` endpoint to see Prometheus instrumentation
7. Reviews code structure noting modular architecture (RPC, ingestion, indexing, storage, API layers)
8. Examines reorg handling logic and error recovery patterns
9. Checks test coverage and documentation quality
10. Forms assessment of candidate's practical skills in Go, databases, APIs, and production-ready thinking

**Success Criteria:**
- System starts successfully within 2 minutes
- Live functionality is immediately visible (blocks indexing, frontend updating)
- Code review reveals clean architecture and production patterns
- Evaluator gains confidence in candidate's ability to build production systems

## UX Design Principles

While this is primarily a backend-focused project with minimal UI, the following UX principles guide the design:

1. **Immediate Feedback** - Live-updating interface shows system activity in real-time, making the demo more engaging and demonstrating WebSocket functionality

2. **Technical Transparency** - Metrics endpoint and structured logs provide visibility into system internals, appealing to technical audience

3. **Zero-Friction Setup** - Docker Compose eliminates environment configuration barriers, ensuring evaluators can see the project running quickly

4. **Focus on Core Value** - Frontend deliberately kept minimal to avoid distraction from backend capabilities being demonstrated

5. **Self-Documenting** - API responses and frontend display key information without requiring deep documentation diving

## Epics

### Epic 1: Core Indexing & Data Pipeline (7-9 stories)

Build the foundational blockchain data pipeline including parallel backfill, real-time indexing, reorg handling, and PostgreSQL storage layer.

**Stories:**
- Implement RPC client with retry logic and error handling
- Create PostgreSQL schema for blocks, transactions, and logs
- Build parallel backfill worker pool for historical blocks
- Implement live-tail mechanism for new blocks
- Add reorg detection and recovery logic
- Create database migration system
- Add Prometheus metrics for indexer performance
- Implement structured logging for debugging
- Add integration tests for indexer pipeline

### Epic 2: API Layer & User Interface (5-6 stories)

Implement REST and WebSocket APIs for data access, along with minimal frontend for demonstration.

**Stories:**
- Build REST API endpoints for blocks, transactions, address history
- Add WebSocket streaming for real-time updates
- Implement pagination for large result sets
- Create minimal SPA frontend with live blocks ticker
- Add transaction search and display interface
- Add health check and metrics exposure endpoints

**Note:** Detailed story breakdown with acceptance criteria will be generated in `epic-stories.md`

## Out of Scope

The following features are explicitly deferred to maintain focus on core demonstration objectives:

**Advanced Blockchain Features:**
- ERC-20 token transfer decoding and balance tracking
- Smart contract source code verification
- Uncle/ommer block handling (Ethereum-specific edge case)
- Historical state queries ("balance at block X")
- Transaction trace/debug data

**Platform Features:**
- Multi-chain support (Polygon, Optimism, Arbitrum, etc.)
- User authentication and API key management
- Rate limiting beyond basic protections
- GraphQL API (REST + WebSocket sufficient for demo)
- Advanced analytics (gas price charts, network statistics trends)

**Data Infrastructure:**
- ClickHouse or TimescaleDB for time-series analytics
- Read replicas for horizontal scaling
- Redis caching layer
- Advanced backup/disaster recovery

**UI/UX:**
- Full-featured block explorer interface
- Charts and visualizations
- Dark mode or theming
- Mobile-responsive design optimization
- Accessibility features beyond basic HTML semantics

**Rationale:** These features would significantly increase scope without proportionally increasing demonstration of core backend/data engineering competencies. The MVP focuses on indexing pipeline, database optimization, API design, and real-time systems—the areas most relevant to target roles.

---

## Assumptions and Dependencies

### Assumptions

1. **Technical Evaluator Behavior** - Evaluators will spend 10-20 minutes reviewing the project and value production-ready patterns over feature completeness

2. **Infrastructure Access** - Ethereum Sepolia testnet will remain accessible throughout development, and free-tier RPC services (Alchemy/Infura) will provide adequate throughput (~300K requests/day)

3. **Development Environment** - Standard development machine with 4+ CPU cores, 8+ GB RAM, and SSD storage is available for development and testing

4. **Timeline Feasibility** - Core blockchain indexing patterns can be implemented in 7 days with focused effort, leveraging well-documented libraries (go-ethereum, pgx, chi)

5. **Scope Sufficiency** - Indexing 5,000 blocks is sufficient to demonstrate capabilities; full historical indexing (50M+ blocks) is not required for portfolio demonstration

6. **Testing Strategy** - Unit tests for critical paths (70%+ coverage) are sufficient; comprehensive end-to-end testing across all edge cases is not required for MVP

7. **Documentation Time** - Adequate time can be allocated for README, API docs, and architecture documentation within the 7-day window

### Dependencies

**External Services:**
- Ethereum Sepolia testnet RPC endpoints (Alchemy, Infura, or public nodes)
- Docker and Docker Compose for local development and deployment

**Technology Stack:**
- Go 1.22+ compiler and toolchain
- PostgreSQL 16 (via Docker)
- Go libraries: go-ethereum, pgx, chi, prometheus/client_golang

**Development Tools:**
- Git for version control
- Text editor/IDE for Go development
- Browser for frontend testing
- API testing tool (curl, Postman, etc.)

**Knowledge Dependencies:**
- Understanding of Ethereum blockchain structure (blocks, transactions, logs)
- Go programming language proficiency
- PostgreSQL database design and optimization
- Docker containerization
- REST API and WebSocket protocols

## Next Steps

**This is a Level 2 project requiring solution architecture and technical specifications before implementation.**

### Immediate Next Actions:

1. **✅ PRD Complete** - Product requirements documented with 14 functional requirements, 5 non-functional requirements, and 2 epics (12-15 stories)

2. **⏭️ Run Solutioning Workflow** - Generate architecture and technical specifications
   - Start new session and run: `/bmad:bmm:workflows:3-solutioning`
   - Input documents: PRD.md, epic-stories.md, Product Brief
   - Expected outputs: solution-architecture.md, tech-spec files per epic

3. **⏭️ Generate Detailed Stories** - Convert epic outlines into full user stories with acceptance criteria
   - Use generated architecture to inform technical acceptance criteria
   - Define story point estimates and dependencies

4. **⏭️ Create Implementation Plan** - Map stories to 7-day sprint schedule
   - Day 1: Project setup + RPC client
   - Day 2: Backfill pipeline
   - Day 3: Live-tail + reorg handling
   - Day 4: REST + WebSocket APIs
   - Day 5: Frontend SPA
   - Day 6: Testing + performance validation
   - Day 7: Documentation + polish

### Handoff to Architecture Phase:

The following artifacts are ready for solution architecture:
- ✅ Product Brief (comprehensive strategic context)
- ✅ PRD with functional/non-functional requirements
- ✅ Epic structure with story titles
- ✅ Technical preferences captured in project-workflow-analysis.md

## Document Status

- [ ] Goals and context validated with stakeholders
- [ ] All functional requirements reviewed
- [ ] User journeys cover all major personas
- [ ] Epic structure approved for phased delivery
- [ ] Ready for architecture phase

_Note: See technical-decisions.md for captured technical context_

---

_This PRD adapts to project level Level 2 (Small complete system) - providing appropriate detail without overburden._
