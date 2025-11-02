# Documentation Index

## 📚 Choose Your Learning Path

This project has comprehensive documentation. Choose the guide that matches your needs:

---

## 🚀 I Want to Start NOW (5 minutes)

**Start here:** [GETTING_STARTED.md](GETTING_STARTED.md)

Quick setup guide with minimal steps:
1. Get RPC endpoint (free, Alchemy recommended)
2. Setup database with one command
3. Run the application
4. Open browser and see live data

**Best for:** Users who want to see it working immediately

---

## 📖 I Want to Understand Everything (30 minutes)

**Read this:** [COMPLETE_WORKFLOW_GUIDE.md](COMPLETE_WORKFLOW_GUIDE.md)

Comprehensive guide covering:
- Prerequisites & environment setup
- RPC endpoint configuration (why, how, comparisons)
- Database setup with schema explanation
- Worker configuration & lifecycle
- API server startup
- 5 different ways to access data (web UI, REST API, WebSocket, database, metrics)
- Complete workflow examples (4 real-world scenarios)
- Troubleshooting guide with solutions
- Performance optimization tips
- Production deployment

**Sections:**
- Prerequisites & Setup
- Step 1: Configure RPC Endpoint
- Step 2: Setup Database
- Step 3: Run Worker to Fetch & Extract Data
- Step 4: Start API Server
- Step 5: Access & Verify Data
- Monitoring & Troubleshooting
- Complete Workflow Examples
- Quick Reference

**Best for:** Developers who need to understand every detail

---

## 🏗️ I Want to Understand Architecture (20 minutes)

**Read this:** [DATA_FLOW_GUIDE.md](DATA_FLOW_GUIDE.md)

Technical deep dive into system architecture:
- High-level data flow overview
- Detailed phase-by-phase breakdown:
  - Phase 1: Worker fetches blockchain data
  - Phase 2: Data storage in PostgreSQL
  - Phase 3: Query data via API
  - Phase 4: Real-time WebSocket streaming
- Data transformation examples (block → database)
- Performance metrics & benchmarks
- Database schema explained with indexes
- Common queries and patterns
- Error handling strategies
- Monitoring points for observability
- Scaling considerations

**Best for:** Developers who need to understand internal workings

---

## 🌐 I Need API Documentation

**See:** [API.md](API.md)

Complete REST API reference:
- All endpoints documented
- Request/response formats
- Pagination details
- Error codes
- WebSocket protocol
- Rate limiting info

**Quick links:**
- Blocks: `GET /v1/blocks`
- Transactions: `GET /v1/txs`
- Addresses: `GET /v1/address/{addr}/txs`
- Event Logs: `GET /v1/logs`
- Chain Stats: `GET /v1/stats/chain`
- Health: `GET /health`

---

## 🐳 I Need Docker Information

**See:** [DOCKER.md](DOCKER.md)

Docker-specific setup and deployment:
- Running with Docker Compose
- Database management
- Development environment setup
- Production deployment

---

## 📊 I Want Interactive API Testing

**See:** [SWAGGER.md](SWAGGER.md)

Swagger/OpenAPI documentation:
- Try endpoints directly in browser
- See request/response examples
- Complete API schema

**Access at:** http://localhost:8081 (after running `make swagger-up`)

---

## 📋 Quick Navigation

### Setup & Getting Started
- **Fast track:** [GETTING_STARTED.md](GETTING_STARTED.md) (5 min)
- **Complete guide:** [COMPLETE_WORKFLOW_GUIDE.md](COMPLETE_WORKFLOW_GUIDE.md) (30 min)
- **Existing:** [QUICKSTART.md](QUICKSTART.md)

### Understanding the System
- **Data flow:** [DATA_FLOW_GUIDE.md](DATA_FLOW_GUIDE.md) (20 min)
- **API reference:** [API.md](API.md)
- **Project overview:** [README.md](README.md)

### Deployment & Operations
- **Docker deployment:** [DOCKER.md](DOCKER.md)
- **Swagger testing:** [SWAGGER.md](SWAGGER.md)

---

## 🎯 Path by Role

### Role: Beginner / New User
```
GETTING_STARTED.md (5 min)
        ↓
Start using web UI
        ↓
COMPLETE_WORKFLOW_GUIDE.md - Section 5 (10 min)
        ↓
Experiment with API
        ↓
COMPLETE_WORKFLOW_GUIDE.md - Full read (20 min)
```

### Role: Developer
```
GETTING_STARTED.md (5 min)
        ↓
Get system running
        ↓
COMPLETE_WORKFLOW_GUIDE.md (30 min)
        ↓
DATA_FLOW_GUIDE.md (20 min)
        ↓
Review code in internal/
```

### Role: DevOps / System Admin
```
COMPLETE_WORKFLOW_GUIDE.md - Prerequisites & Database (15 min)
        ↓
DOCKER.md (10 min)
        ↓
COMPLETE_WORKFLOW_GUIDE.md - Production Deployment (10 min)
        ↓
Setup monitoring
```

### Role: API Consumer
```
COMPLETE_WORKFLOW_GUIDE.md - Step 5 (10 min)
        ↓
API.md (20 min)
        ↓
SWAGGER.md - Interactive testing (10 min)
```

### Role: Architect / Technical Lead
```
README.md - Overview (15 min)
        ↓
DATA_FLOW_GUIDE.md (20 min)
        ↓
COMPLETE_WORKFLOW_GUIDE.md (30 min)
        ↓
Review codebase
```

---

## 📖 Documentation Cheat Sheet

### "I want to..."

| Goal | Document | Section |
|------|----------|---------|
| Start immediately | GETTING_STARTED.md | Entire doc (5 min) |
| Configure RPC | COMPLETE_WORKFLOW_GUIDE.md | Step 1 |
| Setup database | COMPLETE_WORKFLOW_GUIDE.md | Step 2 |
| Index blockchain data | COMPLETE_WORKFLOW_GUIDE.md | Step 3 |
| Start API server | COMPLETE_WORKFLOW_GUIDE.md | Step 4 |
| Access data in browser | COMPLETE_WORKFLOW_GUIDE.md | Step 5.1 |
| Query with REST API | COMPLETE_WORKFLOW_GUIDE.md | Step 5.2 or API.md |
| Get real-time updates | COMPLETE_WORKFLOW_GUIDE.md | Step 5.3 or DATA_FLOW_GUIDE.md |
| Query database directly | COMPLETE_WORKFLOW_GUIDE.md | Step 5.4 |
| Monitor indexing | COMPLETE_WORKFLOW_GUIDE.md | Monitoring section |
| Fix an error | COMPLETE_WORKFLOW_GUIDE.md | Troubleshooting section |
| Understand architecture | DATA_FLOW_GUIDE.md | Entire doc |
| Deploy to production | COMPLETE_WORKFLOW_GUIDE.md | Complete Workflow Examples (Example 4) |
| Use Docker | DOCKER.md | Entire doc |
| Test API interactively | SWAGGER.md | Entire doc |

---

## 🔍 Search for Topics

### Configuration
- RPC endpoint: COMPLETE_WORKFLOW_GUIDE.md - Step 1
- Database: COMPLETE_WORKFLOW_GUIDE.md - Step 2
- Environment variables: COMPLETE_WORKFLOW_GUIDE.md - Step 3
- API port: COMPLETE_WORKFLOW_GUIDE.md - Quick Reference

### Running Services
- Start everything: GETTING_STARTED.md or COMPLETE_WORKFLOW_GUIDE.md - Step 3 & 4
- Run worker only: COMPLETE_WORKFLOW_GUIDE.md - Step 3
- Run API only: COMPLETE_WORKFLOW_GUIDE.md - Step 4
- Background execution: COMPLETE_WORKFLOW_GUIDE.md - Step 3/4

### Accessing Data
- Web UI: COMPLETE_WORKFLOW_GUIDE.md - Step 5.1
- REST API: COMPLETE_WORKFLOW_GUIDE.md - Step 5.2
- WebSocket: COMPLETE_WORKFLOW_GUIDE.md - Step 5.3
- Database: COMPLETE_WORKFLOW_GUIDE.md - Step 5.4
- Metrics: COMPLETE_WORKFLOW_GUIDE.md - Step 5.5
- pgAdmin GUI: COMPLETE_WORKFLOW_GUIDE.md - Step 5.6

### Troubleshooting
- Timeout errors: COMPLETE_WORKFLOW_GUIDE.md - Monitoring & Troubleshooting
- Database issues: COMPLETE_WORKFLOW_GUIDE.md - Monitoring & Troubleshooting
- Port conflicts: COMPLETE_WORKFLOW_GUIDE.md - Monitoring & Troubleshooting
- RPC problems: COMPLETE_WORKFLOW_GUIDE.md - Step 1

### Examples
- Fresh start: COMPLETE_WORKFLOW_GUIDE.md - Example 1
- Recent blocks: COMPLETE_WORKFLOW_GUIDE.md - Example 2
- Development: COMPLETE_WORKFLOW_GUIDE.md - Example 3
- Production: COMPLETE_WORKFLOW_GUIDE.md - Example 4
- Data flow: DATA_FLOW_GUIDE.md - Examples 1-3

---

## 📊 Documentation Stats

| Document | Size | Time | Best For |
|----------|------|------|----------|
| GETTING_STARTED.md | 7.5 KB | 5 min | Quick setup |
| COMPLETE_WORKFLOW_GUIDE.md | 22 KB | 30 min | Full understanding |
| DATA_FLOW_GUIDE.md | 19 KB | 20 min | Architecture |
| API.md | - | 20 min | API reference |
| README.md | - | 15 min | Project overview |
| DOCKER.md | - | 10 min | Docker details |
| SWAGGER.md | - | 10 min | Interactive testing |

**Total learning time: ~45 minutes for complete understanding**

---

## 🚀 One-Minute Summary

**What this project does:**
- Indexes Ethereum blockchain in real-time
- Extracts all transactions and contracts
- Stores in PostgreSQL
- Serves via REST API + WebSocket + Web UI

**To start:**
1. Get free RPC key (Alchemy.com)
2. Run `make db-setup && make run`
3. Open http://localhost:8080/

**To understand:**
- Read COMPLETE_WORKFLOW_GUIDE.md (30 min)

**To know architecture:**
- Read DATA_FLOW_GUIDE.md (20 min)

---

## 📞 Need Help?

1. **Documentation:** Check the relevant guide above
2. **Troubleshooting:** See COMPLETE_WORKFLOW_GUIDE.md - Troubleshooting section
3. **Issues:** https://github.com/hieutt50/go-blockchain-explorer/issues
4. **API Help:** See API.md or try SWAGGER.md

---

## 📝 Document Overview

```
DOCUMENTATION_INDEX.md (this file)
├── How to choose which guide to read
├── Navigation by goal
├── Navigation by role
├── Search by topic
└── Quick links

GETTING_STARTED.md
├── 5-minute setup guide
├── Step 1: Get RPC endpoint
├── Step 2: Setup project
├── Step 3: Run services
├── Step 4: View data
├── Troubleshooting
└── Performance tips

COMPLETE_WORKFLOW_GUIDE.md
├── Prerequisites & environment
├── Step 1: Configure RPC (detailed)
├── Step 2: Setup database (detailed)
├── Step 3: Run worker (detailed)
├── Step 4: Start API (detailed)
├── Step 5: Access data (5 ways)
├── Monitoring & troubleshooting
├── 4 complete workflow examples
├── Quick reference
└── All commands

DATA_FLOW_GUIDE.md
├── High-level architecture
├── Phase 1: Fetch data
├── Phase 2: Store data
├── Phase 3: Query data
├── Phase 4: WebSocket streaming
├── Data transformation examples
├── Performance metrics
├── Database schema
├── Error handling
├── Monitoring points
└── Scaling info

API.md
├── All endpoints
├── Request/response formats
├── Pagination
├── Error codes
└── WebSocket protocol

README.md
├── Project overview
├── Features
├── Setup options
├── Architecture
└── Development status

DOCKER.md
├── Docker setup
├── Docker Compose
├── Database management
└── Production deployment

SWAGGER.md
├── Interactive API testing
├── Try endpoints in browser
└── Live examples
```

---

## ✅ Your Next Step

Choose one:

- **Want to start NOW?** → Read [GETTING_STARTED.md](GETTING_STARTED.md)
- **Want full details?** → Read [COMPLETE_WORKFLOW_GUIDE.md](COMPLETE_WORKFLOW_GUIDE.md)
- **Want to understand architecture?** → Read [DATA_FLOW_GUIDE.md](DATA_FLOW_GUIDE.md)

---

**Last updated:** 2025-11-02
**Project:** Blockchain Explorer - Production-Grade Ethereum Indexer
**Maintainer:** Hieu (https://github.com/hieutt50)
