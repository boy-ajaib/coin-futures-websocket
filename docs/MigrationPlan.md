Migration Plan: Gorilla WebSocket → Centrifuge-Go

Overview

┌───────────────────┬───────────────────────────────┬─────────────────────────────┐
│      Aspect       │       Current (Gorilla)       │     Target (Centrifuge)     │
├───────────────────┼───────────────────────────────┼─────────────────────────────┤
│ Protocol          │ Custom JSON                   │ Centrifuge Protocol         │
├───────────────────┼───────────────────────────────┼─────────────────────────────┤
│ Transport         │ Raw WebSocket                 │ WebSocket (via Centrifuge)  │
├───────────────────┼───────────────────────────────┼─────────────────────────────┤
│ Server            │ http.Server + custom upgrade  │ centrifuge.Server           │
├───────────────────┼───────────────────────────────┼─────────────────────────────┤
│ Client Management │ Custom Hub + Client           │ Built-in connection handler │
├───────────────────┼───────────────────────────────┼─────────────────────────────┤
│ Channels          │ Custom channel map            │ Built-in channels engine    │
├───────────────────┼───────────────────────────────┼─────────────────────────────┤
│ Auth              │ JWT in X-Socket-Authorization │ Centrifuge ConnectHandler   │
└───────────────────┴───────────────────────────────┴─────────────────────────────┘

---

Milestone 1: Project Setup & Dependencies ✅ COMPLETED

Goal: Prepare the project for Centrifuge integration.

Tasks

┌───────┬──────────────────────────────┬──────────────────┬───────────────────────────────────────────────────────┐
│  ID   │             Task             │      Files       │                        Details                        │
├───────┼──────────────────────────────┼──────────────────┼───────────────────────────────────────────────────────┤
│ M1-T1 │ Add centrifuge-go dependency │ go.mod           │ Add github.com/centrifugal/centrifuge-go              │
├───────┼──────────────────────────────┼──────────────────┼───────────────────────────────────────────────────────┤
│ M1-T2 │ Remove gorilla/websocket     │ go.mod, go.sum   │ Remove unused dependency                              │
├───────┼──────────────────────────────┼──────────────────┼───────────────────────────────────────────────────────┤
│ M1-T3 │ Update configuration         │ config/config.go │ Add Centrifuge-specific config (node name, namespace) │
├───────┼──────────────────────────────┼──────────────────┼───────────────────────────────────────────────────────┤
│ M1-T4 │ Create feature branch        │ git              │ Create branch feature/migrate-to-centrifuge           │
└───────┴──────────────────────────────┴──────────────────┴───────────────────────────────────────────────────────┘

Acceptance Criteria:
- go mod tidy completes without errors
- New config options are defined
- Feature branch created and pushed

Estimated Effort: 0.5 day

---

Milestone 2: Centrifuge Server Foundation ✅ COMPLETED

Goal: Replace the WebSocket server with Centrifuge server.

Tasks

┌───────┬──────────────────────────────────┬────────────────────────────────────────────────┬────────────────────────────────────┐
│  ID   │               Task               │                     Files                      │              Details               │
├───────┼──────────────────────────────────┼────────────────────────────────────────────────┼────────────────────────────────────┤
│ M2-T1 │ Create Centrifuge server wrapper │ internal/websocket/server/centrifuge_server.go │ New file wrapping centrifuge.New() │
├───────┼──────────────────────────────────┼────────────────────────────────────────────────┼────────────────────────────────────┤
│ M2-T2 │ Implement Connect handler        │ internal/websocket/server/handlers.go          │ Handle JWT auth, user resolution   │
├───────┼──────────────────────────────────┼────────────────────────────────────────────────┼────────────────────────────────────┤
│ M2-T3 │ Implement Refresh handler        │ internal/websocket/server/handlers.go          │ Token refresh support              │
├───────┼──────────────────────────────────┼────────────────────────────────────────────────┼────────────────────────────────────┤
│ M2-T4 │ Implement Subscribe handler      │ internal/websocket/server/handlers.go          │ Channel subscription validation    │
├───────┼──────────────────────────────────┼────────────────────────────────────────────────┼────────────────────────────────────┤
│ M2-T5 │ Implement Publish handler        │ internal/websocket/server/handlers.go          │ Client publish validation          │
├───────┼──────────────────────────────────┼────────────────────────────────────────────────┼────────────────────────────────────┤
│ M2-T6 │ Implement RPC handler            │ internal/websocket/server/handlers.go          │ For future extensibility           │
├───────┼──────────────────────────────────┼────────────────────────────────────────────────┼────────────────────────────────────┤
│ M2-T7 │ Update main.go                   │ cmd/app/main.go                                │ Wire up Centrifuge server          │
└───────┴──────────────────────────────────┴────────────────────────────────────────────────┴────────────────────────────────────┘

Acceptance Criteria:
- Centrifuge server starts without errors
- HTTP endpoint at /connection (or Centrifuge default)
- Connect handler validates JWT and resolves users
- Health check endpoint returns connection count

Estimated Effort: 2-3 days

---

Milestone 3: Authentication & User Resolution ✅ COMPLETED

Goal: Port authentication and user resolution logic to Centrifuge handlers.

Tasks

┌───────┬──────────────────────────────────┬───────────────────────────────────────┬───────────────────────────────────────────────┐
│  ID   │               Task               │                 Files                 │                    Details                    │
├───────┼──────────────────────────────────┼───────────────────────────────────────┼───────────────────────────────────────────────┤
│ M3-T1 │ Extract JWT parser               │ internal/auth/jwt.go                  │ New file for JWT parsing logic                │
├───────┼──────────────────────────────────┼───────────────────────────────────────┼───────────────────────────────────────────────┤
│ M3-T2 │ Implement Connect credentials    │ internal/websocket/server/handlers.go │ Map JWT to Centrifuge credentials             │
├───────┼──────────────────────────────────┼───────────────────────────────────────┼───────────────────────────────────────────────┤
│ M3-T3 │ Port user resolver integration   │ internal/websocket/server/handlers.go │ Call CFX user mapper at connect               │
├───────┼──────────────────────────────────┼───────────────────────────────────────┼───────────────────────────────────────────────┤
│ M3-T4 │ Port user preference integration │ internal/websocket/server/handlers.go │ Call quote preference provider                │
├───────┼──────────────────────────────────┼───────────────────────────────────────┼───────────────────────────────────────────────┤
│ M3-T5 │ Add connection metadata          │ internal/websocket/server/handlers.go │ Store ajaib_id, cfx_user_id, quote_preference │
├───────┼──────────────────────────────────┼───────────────────────────────────────┼───────────────────────────────────────────────┤
│ M3-T6 │ Update error codes               │ internal/websocket/protocol/errors.go │ Map to Centrifuge close codes                 │
└───────┴──────────────────────────────────┴───────────────────────────────────────┴───────────────────────────────────────────────┘

Acceptance Criteria:
- JWT in Authorization header is parsed correctly
- Invalid JWT returns 4000+ close code
- CFX user resolution failure returns 5031
- User preference stored in connection info
- User connection limit enforced

Estimated Effort: 2 days

---

Milestone 4: Channel Management & Validation ✅ COMPLETED

Goal: Port channel naming and validation to Centrifuge.

Tasks

┌──────────┬─────────────────────────────────┬─────────────────────────────────────────┬────────────────────────────────────────┐
│    ID    │              Task               │                  Files                  │                Details                 │
├──────────┼─────────────────────────────────┼─────────────────────────────────────────┼────────────────────────────────────────┤
│ M4-T1    │ Port channel validator          │ internal/websocket/channel/validator.go │ Adapt for Centrifuge Subscribe event   │
├──────────┼─────────────────────────────────┼─────────────────────────────────────────┼────────────────────────────────────────┤
│ M4-T2    │ Implement channel authorization │ internal/websocket/server/handlers.go   │ Validate user can subscribe to channel │
├──────────┼─────────────────────────────────┼─────────────────────────────────────────┼────────────────────────────────────────┤
│ M4-T3    │ Update channel naming           │ internal/websocket/channel/types.go     │ Ensure user:{ajaib_id}:{type} format   │
├──────────┼─────────────────────────────────┼─────────────────────────────────────────┼────────────────────────────────────────┤
│ M4-T4    │ Remove old Hub                  │ internal/websocket/server/hub.go        │ Delete unused Hub implementation       │
├──────────┼─────────────────────────────────┼─────────────────────────────────────────┼────────────────────────────────────────┤
│ M4-T5    │ Remove old Client               │ internal/websocket/server/client.go     │ Delete unused Client implementation    │
└──────────┴─────────────────────────────────┴─────────────────────────────────────────┴────────────────────────────────────────┘

Acceptance Criteria:
- Subscribe events validated against channel format
- User can only subscribe to their own channels
- Invalid channel format returns error
- Old Hub and Client files removed

Estimated Effort: 1-2 days

---

Milestone 5: Kafka Integration & Broadcasting ✅ COMPLETED

Goal: Integrate Kafka broadcaster with Centrifuge publication API.

Tasks

┌───────┬────────────────────────────────┬────────────────────────────────────────┬────────────────────────────────────────┐
│  ID   │              Task              │                 Files                  │                Details                 │
├───────┼────────────────────────────────┼────────────────────────────────────────┼────────────────────────────────────────┤
│ M5-T1 │ Update broadcaster interface   │ internal/kafka/broadcaster.go          │ Accept centrifuge.Node instead of *Hub │
├───────┼────────────────────────────────┼────────────────────────────────────────┼────────────────────────────────────────┤
│ M5-T2 │ Implement Centrifuge publisher │ internal/kafka/centrifuge_publisher.go │ Wrapper for Centrifuge Publish()       │
├───────┼────────────────────────────────┼────────────────────────────────────────┼────────────────────────────────────────┤
│ M5-T3 │ Update transformer integration │ internal/kafka/broadcaster.go          │ Pass data to Centrifuge publisher      │
├───────┼────────────────────────────────┼────────────────────────────────────────┼────────────────────────────────────────┤
│ M5-T4 │ Remove message handler         │ internal/websocket/handler/handler.go  │ No longer needed with Centrifuge       │
├───────┼────────────────────────────────┼────────────────────────────────────────┼────────────────────────────────────────┤
│ M5-T5 │ Update main.go wiring          │ cmd/app/main.go                        │ Wire broadcaster to Centrifuge node    │
└───────┴────────────────────────────────┴────────────────────────────────────────┴────────────────────────────────────────┘

Acceptance Criteria:
- Kafka messages published to Centrifuge channels
- Transformer still applies to data before publication
- Channel names match format: user:{ajaib_id}:{margin|position}
- Old message handler removed

Estimated Effort: 2 days

---

Milestone 6: Protocol Migration ✅ COMPLETED

Goal: Remove custom protocol, use Centrifuge protocol.

Tasks

┌───────┬─────────────────────────┬──────────────────────────────┬────────────────────────────────────────────┐
│  ID   │          Task           │            Files             │                  Details                   │
├───────┼─────────────────────────┼──────────────────────────────┼────────────────────────────────────────────┤
│ M6-T1 │ Delete protocol package │ internal/websocket/protocol/ │ Remove custom messages.go                  │
├───────┼─────────────────────────┼──────────────────────────────┼────────────────────────────────────────────┤
│ M6-T2 │ Update documentation    │ README.md                    │ Document Centrifuge protocol usage         │
├───────┼─────────────────────────┼──────────────────────────────┼────────────────────────────────────────────┤
│ M6-T3 │ Create client example   │ examples/client/             │ Example client using Centrifuge SDK        │
├───────┼─────────────────────────┼──────────────────────────────┼────────────────────────────────────────────┤
│ M6-T4 │ Update API docs         │ docs/api.md                  │ Document Centrifuge endpoints and protocol │
└───────┴─────────────────────────┴──────────────────────────────┴────────────────────────────────────────────┘

Acceptance Criteria:
- Custom protocol package removed
- Documentation updated
- Working client example provided
- All message types handled by Centrifuge protocol

Estimated Effort: 1 day

---

Milestone 7: Configuration & Logging ✅ COMPLETED

Goal: Finalize configuration and logging for Centrifuge.

Tasks

┌───────┬───────────────────────────────┬───────────────────────────────────────────┬─────────────────────────────────────────┐
│  ID   │             Task              │                   Files                   │                 Details                 │
├───────┼───────────────────────────────┼───────────────────────────────────────────┼─────────────────────────────────────────┤
│ M7-T1 │ Add Centrifuge config options │ config/config.go                          │ Add node name, namespace, log level     │
├───────┼───────────────────────────────┼───────────────────────────────────────────┼─────────────────────────────────────────┤
│ M7-T2 │ Update config YAML            │ config/config.yml, config/development.yml │ Add Centrifuge section                  │
├───────┼───────────────────────────────┼───────────────────────────────────────────┼─────────────────────────────────────────┤
│ M7-T3 │ Integrate structured logging  │ internal/websocket/server/                │ Use slog with Centrifuge logger adapter │
├───────┼───────────────────────────────┼───────────────────────────────────────────┼─────────────────────────────────────────┤
│ M7-T4 │ Add metrics endpoint          │ internal/websocket/server/metrics.go      │ Optional: expose Prometheus metrics     │
├───────┼───────────────────────────────┼───────────────────────────────────────────┼─────────────────────────────────────────┤
│ M7-T5 │ Update shutdown handling      │ cmd/app/main.go                           │ Graceful shutdown for Centrifuge        │
└───────┴───────────────────────────────┴───────────────────────────────────────────┴─────────────────────────────────────────┘

Acceptance Criteria:
- All Centrifuge options configurable
- Logs use structured format (slog)
- Graceful shutdown works correctly
- Optional metrics endpoint available

Estimated Effort: 1 day

---

Milestone 8: Testing & Validation ✅ COMPLETED

Goal: Comprehensive testing of the migrated system.

Tasks

┌───────┬────────────────────────────┬─────────────────────────────────────┬───────────────────────────────────┐
│  ID   │            Task            │                Files                │              Details              │
├───────┼────────────────────────────┼─────────────────────────────────────┼───────────────────────────────────┤
│ M8-T1 │ Unit tests for handlers    │ internal/websocket/server/*_test.go │ Test all Centrifuge handlers      │
├───────┼────────────────────────────┼─────────────────────────────────────┼───────────────────────────────────┤
│ M8-T2 │ Unit tests for broadcaster │ internal/kafka/*_test.go            │ Test Centrifuge integration       │
├───────┼────────────────────────────┼─────────────────────────────────────┼───────────────────────────────────┤
│ M8-T3 │ Integration tests          │ tests/integration/                  │ End-to-end tests with real client │
├───────┼────────────────────────────┼─────────────────────────────────────┼───────────────────────────────────┤
│ M8-T4 │ Load tests                 │ tests/load/                         │ Performance comparison            │
├───────┼────────────────────────────┼─────────────────────────────────────┼───────────────────────────────────┤
│ M8-T5 │ Manual testing             │ -                                   │ Verify all scenarios              │
├───────┼────────────────────────────┼─────────────────────────────────────┼───────────────────────────────────┤
│ M8-T6 │ Update README              │ README.md                           │ Setup and usage instructions      │
└───────┴────────────────────────────┴─────────────────────────────────────┴───────────────────────────────────┘

Acceptance Criteria:
- 80%+ code coverage
- Integration tests pass
- Load tests meet performance requirements
- README complete and accurate

Estimated Effort: 3 days

---

Milestone 9: Code Cleanup & Documentation ✅ COMPLETED

Goal: Final cleanup and documentation.

Tasks

┌───────┬────────────────────────┬───────────────────┬────────────────────────────────┐
│  ID   │          Task          │       Files       │            Details             │
├───────┼────────────────────────┼───────────────────┼────────────────────────────────┤
│ M9-T1 │ Remove unused imports  │ All .go files     │ Clean up go.mod and imports    │
├───────┼────────────────────────┼───────────────────┼────────────────────────────────┤
│ M9-T2 │ Run go mod tidy        │ go.mod, go.sum    │ Final dependency cleanup       │
├───────┼────────────────────────┼───────────────────┼────────────────────────────────┤
│ M9-T3 │ Format code            │ All .go files     │ go fmt and gofumpt             │
├───────┼────────────────────────┼───────────────────┼────────────────────────────────┤
│ M9-T4 │ Lint code              │ All .go files     │ golangci-lint                  │
├───────┼────────────────────────┼───────────────────┼────────────────────────────────┤
│ M9-T5 │ Update CLAUDE.md       │ CLAUDE.md         │ Document migration decisions   │
├───────┼────────────────────────┼───────────────────┼────────────────────────────────┤
│ M9-T6 │ Create migration guide │ docs/MigrationGuide.md │ Document changes for reference │
└───────┴────────────────────────┴───────────────────┴────────────────────────────────┘

Acceptance Criteria:
- No unused imports or code
- Linter passes without errors
- Documentation complete
- Code formatted consistently

Estimated Effort: 1 day

---

Milestone 10: Merge & Deployment Prep

Goal: Merge to main and prepare for deployment.

Tasks

┌────────┬──────────────────────┬───────────┬───────────────────────┐
│   ID   │         Task         │   Files   │        Details        │
├────────┼──────────────────────┼───────────┼───────────────────────┤
│ M10-T1 │ Final review         │ All files │ Code review session   │
├────────┼──────────────────────┼───────────┼───────────────────────┤
│ M10-T2 │ Update version tags  │ -         │ Tag release version   │
├────────┼──────────────────────┼───────────┼───────────────────────┤
│ M10-T3 │ Merge to main        │ git       │ PR and merge          │
├────────┼──────────────────────┼───────────┼───────────────────────┤
│ M10-T4 │ Deploy to staging    │ -         │ First deployment      │
├────────┼──────────────────────┼───────────┼───────────────────────┤
│ M10-T5 │ Smoke test staging   │ -         │ Verify staging works  │
├────────┼──────────────────────┼───────────┼───────────────────────┤
│ M10-T6 │ Deploy to production │ -         │ Production deployment │
└────────┴──────────────────────┴───────────┴───────────────────────┘

Acceptance Criteria:
- Code review approved
- Tests pass in CI/CD
- Staging deployment successful
- Production deployment successful

Estimated Effort: 1 day

---

Summary

┌───────────┬──────────────────────────────────┬──────────────────┐
│ Milestone │              Focus               │ Estimated Effort │
├───────────┼──────────────────────────────────┼──────────────────┤
│ M1        │ Project Setup                    │ 0.5 day          │
├───────────┼──────────────────────────────────┼──────────────────┤
│ M2        │ Centrifuge Server Foundation     │ 2-3 days         │
├───────────┼──────────────────────────────────┼──────────────────┤
│ M3        │ Authentication & User Resolution │ 2 days           │
├───────────┼──────────────────────────────────┼──────────────────┤
│ M4        │ Channel Management               │ 1-2 days         │
├───────────┼──────────────────────────────────┼──────────────────┤
│ M5        │ Kafka Integration                │ 2 days           │
├───────────┼──────────────────────────────────┼──────────────────┤
│ M6        │ Protocol Migration               │ 1 day            │
├───────────┼──────────────────────────────────┼──────────────────┤
│ M7        │ Configuration & Logging          │ 1 day            │
├───────────┼──────────────────────────────────┼──────────────────┤
│ M8        │ Testing & Validation             │ 3 days           │
├───────────┼──────────────────────────────────┼──────────────────┤
│ M9        │ Code Cleanup & Docs              │ 1 day            │
├───────────┼──────────────────────────────────┼──────────────────┤
│ M10       │ Merge & Deployment               │ 1 day            │
├───────────┼──────────────────────────────────┼──────────────────┤
│ Total     │                                  │ ~15-17 days      │
└───────────┴──────────────────────────────────┴──────────────────┘

---

Key Architectural Changes

Before (Gorilla)                          After (Centrifuge)
┌─────────────────┐                       ┌─────────────────┐
│  HTTP Server    │                       │ Centrifuge      │
│  + Upgrader     │                       │ Server          │
└────────┬────────┘                       └────────┬────────┘
         │                                         │
         ▼                                         ▼
┌─────────────────┐                       ┌─────────────────┐
│ Custom Hub      │◄─────────────────────►│ Built-in Engine │
│ - Clients map   │                       │ - Channels      │
│ - Channels map  │                       │ - Presences     │
│ - Broadcast     │                       │ - History       │
└────────┬────────┘                       └────────┬────────┘
         │                                         │
         ▼                                         ▼
┌─────────────────┐                       ┌─────────────────┐
│ Custom Client   │                       │ Centrifuge      │
│ - ReadPump      │                       │ Connection      │
│ - WritePump     │                       │ - Built-in I/O  │
│ - Subscriptions │                       │ - Protocol      │
└─────────────────┘                       └─────────────────┘
