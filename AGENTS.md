# AGENTS.md — SiTukang (Sistem Informasi Tukang)

Read this whole file before writing any code. Follow it literally. If something here conflicts with your instincts, this file wins. If something is not covered, choose the simplest option and note the assumption in your final message. Do not ask for confirmation on small things.

## 1. What this app is

SiTukang helps a construction supervisor record daily attendance of workers (tukang) and pay them weekly. **The core feature is the weekly pay recap (rekap gaji).** Everything else supports it.

- Only admins/supervisors have accounts. Workers have NO login; they are just records (name, role, daily fee).
- UI language: **Indonesian only**. No i18n library. Hardcode Indonesian strings. This applies to user-visible UI text and to the `{"error": "..."}` messages the API returns to the UI.
- Everything else is **English**: code identifiers, comments, log messages, CLI output and prompts, error wrapping (`fmt.Errorf`), test names and failure messages, migration comments, and commit messages. Never mix.
- **Mobile first.** Most use is on a phone at a work site. Design for 360px width first, then scale up.
- Single-user-ish load (a few admins, dozens of workers). Do not over-engineer.

## 2. Tech stack (do not substitute)

| Layer | Choice |
|---|---|
| Backend | Go (latest stable), `github.com/go-chi/chi/v5` |
| Database | SQLite via `modernc.org/sqlite` (pure Go, no CGO), plain `database/sql`. No ORM. |
| Migrations | Plain numbered `.sql` files in `migrations/` at the repo root, embedded with `//go:embed migrations/*.sql` in the root package and applied at startup by a tiny runner (`schema_migrations` table) |
| Excel export | `github.com/xuri/excelize/v2` |
| Passwords | `golang.org/x/crypto/bcrypt` |
| Frontend | React + TypeScript + Vite, **Bun** as package manager/runner |
| UI | Tailwind CSS + shadcn/ui |
| Frontend libs | `react-router`, `@tanstack/react-query`, `react-hook-form` + `zod`, `date-fns` |
| Serving | Go serves the built frontend (`web/dist`) via `go:embed`, with SPA fallback to `index.html` for non-`/api` routes |

Do not add new dependencies without a clear need. Never add an ORM, a state-management library (Redux/Zustand), or a CSS-in-JS library.

## 3. Repository layout

```
situkang/
├── AGENTS.md
├── Makefile
├── go.mod
├── cmd/situkang/main.go          # entrypoint: config, DB, migrate, router, serve
├── internal/
│   ├── config/                   # env config
│   ├── db/                       # open DB, migration runner
│   ├── auth/                     # sessions, middleware, bcrypt
│   ├── worker/                   # handler.go, repo.go, service.go per feature
│   ├── project/
│   ├── attendance/
│   ├── cashadvance/              # kas bon
│   ├── recap/                    # the core: calc, finalize, reopen, export
│   ├── httpx/                    # JSON helpers, error responses, validation
│   ├── money/                    # rupiah helpers
│   └── clock/                    # Jakarta-local "today"/"last Monday", injectable for tests
├── migrations/                   # 0001_init.sql, 0002_...sql
├── web/                          # React app (Bun)
│   ├── src/
│   │   ├── components/ui/        # shadcn components
│   │   ├── components/           # app components
│   │   ├── pages/
│   │   ├── lib/                  # api client, format.ts (rupiah, tanggal), utils
│   │   └── hooks/
│   └── dist/                     # build output (gitignored, embedded by Go)
└── embed.go                      # //go:embed all:web/dist
```

Each Go feature package follows the same pattern: `handler.go` (HTTP only), `service.go` (business rules), `repo.go` (SQL only). Handlers never contain SQL. Repos never contain business rules.

## 4. Commands (use exactly these)

```
make dev-api      # go run ./cmd/situkang  (listens :8080)
make dev-web      # cd web && bun run dev  (Vite on :5173, proxies /api to :8080)
make build        # cd web && bun install && bun run build, then go build -o bin/situkang ./cmd/situkang
make test         # go test ./...  and  cd web && bun run typecheck
make lint         # go vet ./...  and  cd web && bun run lint
make create-admin # go run ./cmd/situkang create-admin  (prompts for username/password)
```

Before declaring any task done, run `make test` and `make lint` and fix failures. If a command in this list does not exist yet, create it in the Makefile as part of the task.

`web/package.json` must expose a `typecheck` script (`tsc --noEmit`) and a `lint` script, because `make test` and `make lint` depend on them.

Config via env vars (with defaults): `PORT=8080`, `DB_PATH=./data/situkang.db`, `SESSION_SECRET` (required in production), `TZ=Asia/Jakarta`.

## 5. Domain rules (the most important section)

### 5.1 Money
- Money is **`int64` rupiah**. Never use floats for money, anywhere (Go, SQL, TypeScript math on money must use integers).
- In SQL: `INTEGER`. In JSON: number (integer).
- Display format: `Rp 1.250.000` (dot thousands separator, no decimals). Implement once in `web/src/lib/format.ts` and `internal/money`. Reuse it everywhere.

### 5.2 Attendance
- One attendance record per worker per date. Enforce with `UNIQUE(worker_id, date)`.
- Attendance amount is stored as **`quarters` INTEGER 1–4**: 1 = ¼ hari, 2 = ½ hari, 3 = ¾ hari, 4 = penuh. **Default is 4.** Absent = no record at all (never store 0).
- **Snapshot the fee**: each attendance row stores `daily_fee_snapshot` (copied from the worker's `daily_fee` at the time of recording). Changing a worker's fee later must NOT change existing attendance.
- Earned amount for a record = `daily_fee_snapshot * quarters / 4` using integer division (floor). Sum these per record; do not multiply the total.
- The fee is per worker per day, **regardless of how many projects** the worker was on. Projects are only tags on the attendance record (many-to-many via `attendance_projects`). Project tags never change the amount. Consequence: the project timeline (§9 step 8) is a record of what actually happened, not a plan — do not model planned assignments.
- Working days default to Monday–Saturday (configurable setting `working_days`, stored as a list of weekday numbers). This only controls which days are shown/highlighted by default. **Recording attendance on a non-working day (e.g. Sunday) must still be allowed.**

### 5.3 Dates and time
- Dates are date-only strings `YYYY-MM-DD` in the **Asia/Jakarta** timezone. Store as `TEXT` in SQLite. Never store attendance dates as timestamps.
- Compute "today" and "last Monday" in Asia/Jakarta on the server, never from the UTC clock. Week starts on Monday.
- Display format in UI: `Sen, 12 Jan 2026` (Indonesian day/month abbreviations). Use `date-fns/locale/id`.

### 5.4 Kas bon (cash advance)
- A kas bon is recorded for a worker on a date with an `amount` (positive integer) and optional note.
- It is subtracted from what the worker is owed in the next recap that includes it.

### 5.5 Settled vs unsettled (this replaces any "last recap date")
- Every attendance, kas bon, and extra fee has a nullable `recap_id`. **`recap_id IS NULL` means unsettled.**
- There is NO "paid through" date on workers. Never add one.
- A worker not paid in a recap simply keeps their items unsettled; those items show up in the next recap automatically.
- A worker with `active = 0` (soft-deleted) who still has unsettled items **must still appear in recaps** — their money must be payable. Never filter recap contents by `active`.

### 5.6 Recaps
A recap has `status`: `draft` or `final`.

**Creating a draft** takes a period mode. `all_unsettled` is the **default and recommended** mode in the UI, labeled *"Semua yang belum dibayar"*:
- `all_unsettled`: all unsettled items regardless of date (this is the meaning of "sejak rekap terakhir"). The day the recap is created never affects who is included.
- `since_monday`: unsettled items dated from the most recent Monday (on or before today) through today. Convenience for "this week" only when the recap runs inside the week it covers; **on a Monday this range is Monday alone**, so it silently excludes the previous week. Never use it to pay a worker who missed an earlier payout.
- `custom`: unsettled items between `start_date` and `end_date` inclusive.

For `custom` and `since_monday`, if a worker has unsettled items dated **before** the range start, return a warning count per worker so the UI can show "Ada X hari belum dibayar sebelum tanggal ini". Do not include those items.

**Only one draft recap may exist at a time.** Creating a second one returns an error (409) with an Indonesian message.

**Draft contents** (computed live, not stored per item): for each worker with unsettled items in range (a worker whose only item in range is an extra fee still appears):
- attendance days and `gross` (sum of earned amounts)
- `advances` (sum of kas bon)
- `extras` (sum of extra fees, which are signed: positive = bonus, **negative = deduction**)
- `net = gross + extras - advances`
- `paid` flag, default `true`, persisted per worker per draft in `recap_paid` (§6)

**Extra fees** (`extra_fees` table) are added during a draft, belong to a worker and the draft recap (`recap_id` set immediately), have a signed integer `amount` and a required `description`. Editable/deletable while the recap is draft.

**Finalize**: in ONE database transaction, for each worker with `paid = true`:
1. Set `recap_id` on all of that worker's in-range unsettled attendance and kas bon.
2. Write a `recap_lines` row snapshot (worker name, role, days, gross, advances, extras, net, and the item ids it covers are implied by `recap_id`).
3. If `net < 0`: create an automatic kas bon for that worker with `amount = -net`, note `Saldo sebelumnya (rekap #<id>)`, dated today, unsettled. This carries the negative balance forward. Store the line's `net` as the real negative number for the record.
For workers with `paid = false`: their extra fees for this draft are detached (`recap_id = NULL`) so they appear again next time, and nothing else changes. Then set recap `status = final`, `finalized_at = now`.

**Worker not on site (the normal `paid = false` case)**: this is how a worker who is absent on payday gets paid later, whenever they show up (Monday, the next Saturday, whenever). Nothing is settled for them and no carry-forward record is created (the automatic kas bon above applies only to `paid = true` workers with `net < 0`). Their items reappear automatically in whatever recap is created next, on any day of the week. There is no "pending payment" field anywhere, and the payout day never enters the data model.

**Locked after finalize**: attendance, kas bon, and extra fees with a non-NULL `recap_id` pointing to a final recap cannot be edited or deleted. The API must reject with 409 and message `Data sudah masuk rekap dan terkunci. Buka kembali rekap untuk mengubahnya.`

**Reopen (Buka kembali rekap)**:
- Allowed only for the **most recent final recap**, and only if no draft recap exists.
- In one transaction: set `recap_id = NULL` on all attendance and kas bon that have this recap's id. Extra fees keep their `recap_id` so they return to the reopened draft. Delete the auto-generated "Saldo sebelumnya" kas bon created by this recap, identified **only** by `cash_advances.source_recap_id = <this recap id>` (never by matching the note text). That kas bon is unsettled; if it had already been settled in a later recap, that later recap would be the most recent one, so reopen would not be allowed. Delete this recap's `recap_lines`, keep its `recap_paid` rows so the `paid` toggles survive the reopen, and set status back to `draft`.
- A reopened draft behaves like any draft (edit, re-finalize).
- Require a confirmation in the UI.

**Delete draft**: allowed. It also deletes the draft's extra fees (they only exist for that draft). Final recaps cannot be deleted, only reopened.

### 5.7 Exports (from a finalized recap, or a draft as preview)
- **Excel**: generated server-side with excelize at `GET /api/recaps/{id}/export.xlsx`. One summary sheet (all workers) plus optionally one detail sheet with per-day attendance. Rupiah cells use number format, not strings.
- **Periode label** (both exports and UI): derive it from the items actually included for that worker — earliest attendance/kas bon date to latest — never from the recap `mode`. For `all_unsettled` a worker catching up may span several weeks; the label must show that honestly (`06 Jan 2026 - 18 Jan 2026`). If a worker has no dated items (extra fees only), show the recap's `created_at` date.
- **WhatsApp**: server returns formatted plain text at `GET /api/recaps/{id}/whatsapp?worker_id=` (per worker, or all workers if omitted). The frontend offers a **Salin** (copy) button and a **Kirim WA** button that opens `https://wa.me/?text=<urlencoded>`. No WhatsApp API.

WhatsApp text format (per worker):

```
*REKAP GAJI*
Nama: Budi (Tukang Batu)
Periode: 06 Jan 2026 - 11 Jan 2026

Kehadiran: 5,5 hari
Gaji kotor: Rp 825.000
Kas bon: - Rp 100.000
Tambahan: + Rp 50.000 (Lembur)
*Total diterima: Rp 775.000*
```

Use Indonesian decimal comma for fractional days (`5,5`, `5,25`, `5,75`).

## 6. Data model (SQLite)

Create in `migrations/0001_init.sql`. Use `STRICT` tables where possible. Enable `PRAGMA foreign_keys = ON` and `journal_mode = WAL` at connection open.

- `users(id, username UNIQUE, password_hash, created_at)`
- `sessions(id TEXT PK, user_id, expires_at, created_at)` — `id` stores the **SHA-256 hash** of the cookie token, never the raw token
- `settings(key PK, value)` — e.g. `working_days` = `1,2,3,4,5,6`
- `workers(id, name, role, daily_fee INTEGER, active INTEGER DEFAULT 1, created_at)` — **soft delete only** (set `active = 0`); never hard-delete a worker with history
- `projects(id, name, start_date, end_date, status, notes, created_at)` — status: `aktif` | `selesai` | `ditunda`
- `attendances(id, worker_id, date, quarters CHECK 1..4, daily_fee_snapshot, recap_id NULL, created_at, UNIQUE(worker_id, date))`
- `attendance_projects(attendance_id, project_id, PRIMARY KEY(attendance_id, project_id))`
- `cash_advances(id, worker_id, date, amount CHECK > 0, note, recap_id NULL, source_recap_id NULL, created_at)` — `source_recap_id` is set only on the automatic "Saldo sebelumnya" kas bon and points at the recap that created it
- `recaps(id, mode, start_date NULL, end_date NULL, status, created_at, finalized_at NULL)` — `mode` is `since_monday` | `all_unsettled` | `custom`
- `recap_lines(id, recap_id, worker_id, worker_name, worker_role, days_quarters, gross, advances, extras, net)`
- `recap_paid(recap_id, worker_id, paid INTEGER DEFAULT 1, PRIMARY KEY(recap_id, worker_id))` — the draft's per-worker `paid` toggle; a missing row means `paid = true`. Rows are kept after finalize (they record which workers were skipped) and survive reopen. Cascade-delete with the recap.
- `extra_fees(id, worker_id, recap_id NULL, amount, description, created_at)` — `recap_id` is nullable because `paid = false` workers have their extras detached on finalize

Add indexes on `attendances(worker_id, date)`, `attendances(recap_id)`, `cash_advances(worker_id, recap_id)`, `cash_advances(source_recap_id)`, `extra_fees(recap_id)`, `recap_paid(recap_id)`.

## 7. API conventions

- All routes under `/api`. JSON in/out. Chi router, middleware: request logger, recoverer, session auth (everything except `POST /api/auth/login`).
- Auth: `POST /api/auth/login`, `POST /api/auth/logout`, `GET /api/auth/me`. Session id in an `HttpOnly`, `SameSite=Lax`, `Secure` (when not localhost) cookie, 30-day expiry. Password hashed with bcrypt. Mutating routes accept only `application/json`.
- Errors always: `{"error": "<pesan dalam Bahasa Indonesia>"}` with a correct status code (400 validation, 401, 404, 409 conflict/locked, 500).
- Validate every input on the server. Never trust the client. Minimum rules: worker `name` required (1–100 chars), `role` required, `daily_fee >= 0`; project `name` required, `status` in `aktif|selesai|ditunda`, `end_date >= start_date` when both are set; kas bon `amount > 0`; extra fee `description` required and `amount != 0`; `quarters` in 1–4; dates must be real calendar dates in `YYYY-MM-DD`.
- Use database transactions for anything that writes to more than one table.
- Routes (REST, plural nouns): `/api/workers`, `/api/projects`, `/api/attendances` (query by `date` or `from`/`to`), `/api/cash-advances`, `/api/settings`, `/api/dashboard` (unsettled total per worker for Beranda), and:
  - `GET /api/recaps` (list, newest first), `POST /api/recaps` (create draft), `GET /api/recaps/{id}`, `DELETE /api/recaps/{id}` (draft only)
  - `PATCH /api/recaps/{id}/paid` with `{worker_id, paid}` — toggles the draft's per-worker `paid` flag
  - `PATCH /api/recaps/{id}/extra-fees/{feeId}`, `DELETE /api/recaps/{id}/extra-fees/{feeId}`, `POST /api/recaps/{id}/extra-fees`
  - `POST /api/recaps/{id}/finalize`, `POST /api/recaps/{id}/reopen`
  - `GET /api/recaps/{id}/export.xlsx`, `GET /api/recaps/{id}/whatsapp`
- Attendance has a convenience endpoint to record a whole day at once: `PUT /api/attendances/day/{date}` with a list of `{worker_id, quarters, project_ids}`; workers omitted from the list have their record for that day removed. If any listed change would touch a locked record (non-NULL `recap_id` pointing at a final recap), reject the **whole request** with 409 and the locked message. Omitting a worker whose record is locked is a no-op, not an error — the record simply stays.

## 8. Frontend rules

- **Mobile first**: build layouts for 360px; use bottom navigation (Beranda, Absensi, Rekap, Pekerja, Proyek) on mobile and a sidebar on `md:` and up. Touch targets at least 44px. No horizontal scrolling. Prefer cards over wide tables on mobile.
- Use shadcn components (`bunx shadcn@latest add <component>`). Do not hand-roll buttons, dialogs, selects, or toasts that shadcn already provides. On mobile use `Drawer`/`Sheet` instead of centered dialogs for forms.
- All user-visible text in Indonesian. Examples: Simpan, Batal, Hapus, Tambah, Ubah, Cari, Belum dibayar, Sudah dibayar, Kas bon, Rekap, Absensi, Pekerja, Proyek, Penuh, Setengah hari.
- Attendance screen (the most-used screen): pick a date (defaults to today), see all active workers as a list, each with a one-tap control for ¼ / ½ / ¾ / Penuh and an "absen" (not present) state. Default when marking present is **Penuh**. Optimize for fewest taps.
- Server state via TanStack Query; invalidate related queries after mutations. Forms via react-hook-form + zod, with Indonesian error messages.
- Every list/screen must have loading, empty, and error states.
- Format all rupiah and dates through `lib/format.ts`. No inline `toLocaleString` scattered around.
- TypeScript strict mode. No `any`. Keep API types in `web/src/lib/types.ts`.

## 9. Scope and build order

Build in this order, one step at a time. Finish, test, and commit each step before starting the next. Do not start a later step early.

1. **Skeleton**: Go module, chi server, SQLite open + migration runner, health endpoint, Vite+React+Tailwind+shadcn scaffold, `go:embed` serving + SPA fallback, Makefile.
2. **Auth**: users, sessions, login/logout/me, `create-admin` command, login page, route guard.
3. **Workers**: CRUD (soft delete), list/search, form.
4. **Attendance**: day screen, `PUT /attendances/day/{date}`, fee snapshot, week view per worker.
5. **Kas bon**: CRUD (respecting lock rule), per-worker list.
6. **Recap core**: create draft (3 modes + warning, default `all_unsettled`), live calculation, `paid` toggle persisted in `recap_paid`, extra fees, finalize, locking, reopen. **Write Go unit tests for the calculation, settlement, carry-forward, and reopen logic** — this is the one place bugs cost real money.
7. **Exports**: Excel and WhatsApp text.
8. **Projects**: list/CRUD with status and dates, tag projects on attendance, show project on worker/day views, plus the **timeline/calendar view**: a read-only month view drawing each project's `start_date`–`end_date` bar overlaid with the days workers were tagged on it. Derived entirely from existing data — no planning or assignment table, no drag-and-drop editing.
9. **Settings**: working days.
10. **Polish**: dashboard (Beranda: unsettled total per worker), mobile QA pass.

**v2 (do NOT build now)**: Gantt-style editing and planned project assignments, multi-admin roles, worker self-service, notifications, anything with a WhatsApp API.

## 10. Hard rules (never do these)

- Never use floats for money or attendance fractions.
- Never hard-delete workers, or edit/delete locked (settled) records.
- Never add a "last recap date" or "paid through" field.
- Never put SQL in handlers or business logic in repos.
- Never use `time.Now()` directly for dates in business logic; use a clock helper that returns Jakarta-local dates (so tests can fix the date).
- Never string-concatenate SQL with user input; always use `?` placeholders.
- Never commit `data/`, `bin/`, `web/dist/`, `node_modules/`, or `.env`.
- Never write English text in the UI.
- Never write Indonesian in code comments, logs, CLI output, or error wrapping; those are English.
- Never rewrite or reformat files unrelated to the current task.
- Never skip tests for the recap package.
- Never run destructive commands (`rm -rf`, dropping the DB) without being asked.
- Never exclude a worker from a recap because `active = 0`.
- Never identify the automatic "Saldo sebelumnya" kas bon by its note text; use `source_recap_id`.
- Never derive a recap's displayed period from its `mode`; derive it from the included items.
- Never treat the project timeline as a planning tool; it is a read-only view of recorded attendance.

## 11. How to work

- Work in small steps. Before editing, state in 2–4 lines what you will change and in which files.
- Read existing code in the package before adding to it; follow its patterns instead of inventing new ones.
- Prefer boring, explicit code over clever abstractions. No premature generics.
- After changes: run `make test` and `make lint`, fix issues, then summarize what you did and any assumptions in a short message.
- If a requirement is ambiguous and not covered here, pick the simplest reasonable option, state it in your summary, and continue. Only stop to ask if the choice would change the data model in section 6 or the rules in section 5.
- Commit messages: imperative, English, one line (e.g. `Add attendance day endpoint`).
