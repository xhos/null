# CLAUDE.md

This is a **personal finance tracker**, NOT enterprise forex software.

Expected scale: 1-10 users, <50K transactions.

## Core Principle: SIMPLICITY OVER COMPLEXITY

Prefer simple solutions that work. This is not a high-scale system.

## DO NOT Suggest

- Materialized views, partitioning, complex triggers
- Microservices, event sourcing, message queues
- Heavy frameworks or over-engineering

## DO Suggest

- Simple normalized tables with standard PostgreSQL types
- Business logic in Go, not SQL procedures
- Minimal dependencies and standard patterns
- chi/stdlib for HTTP, sqlc for DB, HTMX/simple React for frontend

## Decision Rule

Before suggesting anything, ask: "Is this actually needed RIGHT NOW for <50K transactions?"

If no → don't suggest it.

**Choose boring technology.**

## Overview

null is a high-performance financial operations backend built with Go, using Connect-RPC (gRPC-compatible) for API communication. It's part of the null ecosystem for personal finance management, handling transactions, categorization, receipts, and financial dashboards.

## Scripts

see flake.nix for development scripts.

useful ones include:

regen: regenerate all code (sqlc, proto)

## Documentation

- **[BALANCE_CALCULATIONS.md](./BALANCE_CALCULATIONS.md)** - Comprehensive guide to the anchor-based balance calculation system
  - How `balance_after_cents` is calculated for transactions
  - Forward and backward calculation logic
  - When balances are recalculated (SyncAccountBalances)
  - Edge cases and common pitfalls

## Architecture

### Layered Architecture

The codebase follows a clean layered architecture:

1. **API Layer** (`internal/api/`) - Connect-RPC handlers
   - Receives proto requests, extracts user context
   - Validates permissions via middleware
   - Calls service layer
   - Maps between proto types and internal types

2. **Service Layer** (`internal/service/`) - Business logic
   - All services instantiated in `service.go:New()`
   - Each service is an interface with a private implementation (e.g., `TransactionService` → `txnSvc`)
   - Services orchestrate database queries and apply business rules
   - **Critical**: Services may call other services (e.g., transactions call rules, categories)

3. **Database Layer** (`internal/db/`) - Data persistence
   - Uses sqlc for type-safe SQL
   - Queries in `internal/db/queries/*.sql`
   - Generated code in `internal/db/sqlc/`
   - Migrations in `internal/db/migrations/` (goose format)

4. **Types Layer** (`internal/types/`) - Domain models
   - Custom types like `Money` for JSONB money fields
   - Implements `sql.Scanner` and `driver.Valuer` for database compatibility

### Key Design Patterns

#### SQL Query Updates with sqlc

When updating database records, use `coalesce(sqlc.narg('field'), field)` pattern to enable partial updates:

```sql
-- This allows updating only specific fields without touching others
UPDATE table SET
  field1 = coalesce(sqlc.narg('field1'), field1),
  field2 = coalesce(sqlc.narg('field2'), field2)
WHERE id = sqlc.arg(id);
```

This prevents NULL constraint violations when only updating a subset of fields.

#### Money Handling

All monetary amounts are stored as JSONB in PostgreSQL with structure:
```json
{"currency_code": "CAD", "units": 100, "nanos": 500000000}
```

The `internal/types/Money` wrapper handles conversion between:
- Database JSONB (`[]byte`)
- Proto `google.type.Money`
- Go structs

#### Transaction Categorization Flow

1. **Creation**: Transaction created → Balance calculated → Rules applied automatically
2. **Rules Engine** (`internal/rules/`):
   - JSON-based condition matching (AND/OR logic)
   - Field types: merchant, tx_desc, amount, account_type, etc.
   - Operators: equals, contains, starts_with, between, regex, etc.
   - See `internal/rules/schema.go` for complete schema
3. **AI Fallback**: If no rules match, AI providers (OpenAI, Anthropic, Ollama, Gemini) can categorize
4. **Manual Override**: User-set categories/merchants skip automatic processing (tracked via `_manually_set` flags)

#### Balance Calculations

Account balances use an anchor-based system:
- Each account has `anchor_date` and `anchor_balance`
- Transactions store `balance_after_cents` calculated from anchor using window functions
- `SyncAccountBalances` query recalculates all balances atomically (both forward and backward in time)
- Critical: Call `SyncAccountBalances` after any transaction amount/date/direction/account change

**See [BALANCE_CALCULATIONS.md](./BALANCE_CALCULATIONS.md) for detailed documentation on how balances are calculated.**

### Authentication & Authorization

- Uses BetterAuth for authentication (external service)
- JWT tokens validated via middleware
- User context extracted to `context.Context` via `UserContext()` middleware
- All database queries verify ownership through `owner_id` or `account_users` join table
- Multi-user account access via `account_users` table

### Service Dependencies

Service initialization order matters (from `service.go:New()`):
```
Categories → Rules → Transactions
            ↓
         Accounts → Dashboard
            ↓
         Users, Receipts, Backup
```

Key dependency: `TransactionService` requires both `CategoryService` and `RuleService` for auto-categorization.

### Logging

- Uses charmbracelet/log throughout
- Configurable via `LOG_LEVEL` (debug, info, warn, error) and `LOG_FORMAT` (json, text)
- Each service gets prefixed logger: `lg.WithPrefix("txn")`
- Development uses `LOG_FORMAT=text` (set in `.air.toml`)
- Production should use `LOG_FORMAT=json` for structured logging

## Database Schema Notes

### Critical Constraints

- `tx_date` on transactions is NOT NULL - always required
- Category and merchant can be NULL (uncategorized)
- `category_manually_set` and `merchant_manually_set` prevent auto-updates
- Account balances recalculated on transaction changes

### Enum Mappings

sqlc maps PostgreSQL smallint columns to proto enums directly:
- `account_type` → `pb.AccountType`
- `tx_direction` → `pb.TransactionDirection` (1=INCOMING, 2=OUTGOING)
- `parse_status`, `link_status` → Receipt enums

### Extensions

- `pg_trgm` - Fuzzy text search (similarity function)
- `pgcrypto` - UUID generation

## Environment Configuration

Required environment variables (see `.env.example`):
- `DATABASE_URL` - PostgreSQL connection string
- `API_KEY` - Internal API authentication
- `BETTER_AUTH_URL` - Auth service endpoint
- `NULL_RECEIPTS_URL` - Receipt parser service endpoint
- `EXCHANGE_API_URL` - Exchange rate API

Optional AI providers:
- `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `OLLAMA_API_KEY`, `GOOGLE_API_KEY`

## Proto/Connect-RPC

- Proto definitions in git submodule `proto/` (from null-protos repo)
- Generate with `buf generate` in proto directory
- Server uses `connect.WithInterceptors()` for logging and user extraction
- All services implement generated `*ServiceHandler` interfaces

## Common Patterns

### Adding a New Database Query

1. Write SQL in `internal/db/queries/[service].sql` with sqlc comments
2. Run `sqlc generate` to create Go code
3. Use generated method in service layer

### Adding a New Service Method

1. Add method to service interface in `internal/service/[service].go`
2. Implement in private struct (e.g., `txnSvc`)
3. Add proto definition (in proto submodule)
4. Run `buf generate` in proto directory
5. Implement handler in `internal/api/[service].go`
6. Map between proto and internal types in `internal/api/mappers.go`

### Transaction Balance Updates

Always call `s.queries.SyncAccountBalances(ctx, accountID)` after:
- Creating transactions
- Updating transaction amount/date/direction
- Deleting transactions
- Bulk operations

Use `sync` not individual recalculation for correctness.
