// Package recap builds, finalizes, reopens, and exports weekly pay recaps.
package recap

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"situkang/internal/clock"
)

var (
	ErrNotFound            = errors.New("recap not found")
	ErrDraftExists         = errors.New("a draft recap already exists")
	ErrNotDraft            = errors.New("recap is not a draft")
	ErrNotFinal            = errors.New("recap is not final")
	ErrNotLatest           = errors.New("only the latest final recap can be reopened")
	ErrInvalidMode         = errors.New("invalid recap mode")
	ErrInvalidRange        = errors.New("invalid recap range")
	ErrLocked              = errors.New("recap data locked")
	ErrInvalidWorker       = errors.New("invalid worker")
	ErrInvalidAmount       = errors.New("invalid extra fee amount")
	ErrDescriptionRequired = errors.New("extra fee description required")
	ErrShareNotFound       = errors.New("recap share not found")
)

const maxDescriptionLength = 200

// CreateInput selects the period of a new draft.
type CreateInput struct {
	Mode      string
	StartDate *string
	EndDate   *string
}

// Service holds the recap business rules.
type Service struct {
	db    *sql.DB
	clock clock.Clock
}

// NewService builds the recap service.
func NewService(db *sql.DB, clk clock.Clock) *Service {
	return &Service{db: db, clock: clk}
}

// List returns every recap, newest first.
func (s *Service) List(ctx context.Context) ([]Recap, error) {
	return listRecaps(ctx, s.db)
}

// Create makes the single allowed draft recap.
func (s *Service) Create(ctx context.Context, in CreateInput) (Recap, error) {
	start, end, err := s.period(in)
	if err != nil {
		return Recap{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Recap{}, fmt.Errorf("begin create recap: %w", err)
	}
	defer tx.Rollback()

	exists, err := draftExists(ctx, tx)
	if err != nil {
		return Recap{}, err
	}
	if exists {
		return Recap{}, ErrDraftExists
	}

	id, err := insertRecap(ctx, tx, in.Mode, start, end, s.clock.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return Recap{}, err
	}
	if err := tx.Commit(); err != nil {
		return Recap{}, fmt.Errorf("commit create recap: %w", err)
	}

	return getRecap(ctx, s.db, id)
}

func (s *Service) period(in CreateInput) (*string, *string, error) {
	switch in.Mode {
	case "all_unsettled":
		return nil, nil, nil
	case "since_monday":
		start := clock.FormatDate(clock.LastMonday(s.clock))
		end := clock.FormatDate(s.clock.Today())
		return &start, &end, nil
	case "custom":
		if in.StartDate == nil || in.EndDate == nil {
			return nil, nil, ErrInvalidRange
		}
		start, err := clock.ParseDate(*in.StartDate)
		if err != nil {
			return nil, nil, ErrInvalidRange
		}
		end, err := clock.ParseDate(*in.EndDate)
		if err != nil || end.Before(start) {
			return nil, nil, ErrInvalidRange
		}
		startValue := clock.FormatDate(start)
		endValue := clock.FormatDate(end)
		return &startValue, &endValue, nil
	default:
		return nil, nil, ErrInvalidMode
	}
}

// Detail computes the live draft contents.
func (s *Service) Detail(ctx context.Context, id int64) (Detail, error) {
	recap, err := s.get(ctx, s.db, id)
	if err != nil {
		return Detail{}, err
	}
	return compute(ctx, s.db, recap)
}

func (s *Service) get(ctx context.Context, q queryer, id int64) (Recap, error) {
	recap, err := getRecap(ctx, q, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Recap{}, ErrNotFound
	}
	return recap, err
}

// ShareToken returns the existing public token for a final recap.
func (s *Service) ShareToken(ctx context.Context, recapID int64) (string, error) {
	recap, err := s.get(ctx, s.db, recapID)
	if err != nil {
		return "", err
	}
	if recap.Status != "final" {
		return "", ErrNotFinal
	}
	token, err := shareToken(ctx, s.db, recapID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrShareNotFound
	}
	if err != nil {
		return "", fmt.Errorf("load recap share: %w", err)
	}
	return token, nil
}

// CreateShare returns the public token for a final recap, creating it once.
func (s *Service) CreateShare(ctx context.Context, recapID int64) (string, error) {
	recap, err := s.get(ctx, s.db, recapID)
	if err != nil {
		return "", err
	}
	if recap.Status != "final" {
		return "", ErrNotFinal
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate share token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)

	if err := insertShare(ctx, s.db, recapID, token, s.clock.Now().UTC().Format(time.RFC3339)); err != nil {
		return "", err
	}

	token, err = shareToken(ctx, s.db, recapID)
	if err != nil {
		return "", fmt.Errorf("load recap share: %w", err)
	}
	return token, nil
}

// RevokeShare removes the public token of a recap.
func (s *Service) RevokeShare(ctx context.Context, recapID int64) error {
	if _, err := s.get(ctx, s.db, recapID); err != nil {
		return err
	}
	return deleteShare(ctx, s.db, recapID)
}

// PublicDetail resolves a share token into the frozen recap view.
func (s *Service) PublicDetail(ctx context.Context, token string) (Detail, error) {
	if token == "" {
		return Detail{}, ErrShareNotFound
	}

	recapID, err := shareRecapID(ctx, s.db, token)
	if errors.Is(err, sql.ErrNoRows) {
		return Detail{}, ErrShareNotFound
	}
	if err != nil {
		return Detail{}, fmt.Errorf("load recap share: %w", err)
	}

	recap, err := s.get(ctx, s.db, recapID)
	if errors.Is(err, ErrNotFound) {
		return Detail{}, ErrShareNotFound
	}
	if err != nil {
		return Detail{}, err
	}
	if recap.Status != "final" {
		return Detail{}, ErrShareNotFound
	}
	return compute(ctx, s.db, recap)
}
// SetPaid toggles whether a worker is paid in this draft.
func (s *Service) SetPaid(ctx context.Context, recapID, workerID int64, paid bool) error {
	recap, err := s.get(ctx, s.db, recapID)
	if err != nil {
		return err
	}
	if recap.Status != "draft" {
		return ErrNotDraft
	}
	ok, err := workerExists(ctx, s.db, workerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidWorker
	}
	return upsertPaid(ctx, s.db, recapID, workerID, paid)
}

// AddFee attaches a signed extra fee to the draft.
func (s *Service) AddFee(ctx context.Context, recapID, workerID, amount int64, description string) (ExtraFee, error) {
	if err := s.requireDraft(ctx, recapID); err != nil {
		return ExtraFee{}, err
	}
	description = strings.TrimSpace(description)
	if amount == 0 {
		return ExtraFee{}, ErrInvalidAmount
	}
	if description == "" || len([]rune(description)) > maxDescriptionLength {
		return ExtraFee{}, ErrDescriptionRequired
	}
	ok, err := workerExists(ctx, s.db, workerID)
	if err != nil {
		return ExtraFee{}, err
	}
	if !ok {
		return ExtraFee{}, ErrInvalidWorker
	}

	id, err := insertFee(ctx, s.db, recapID, workerID, amount, description, s.clock.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return ExtraFee{}, err
	}
	return ExtraFee{ID: id, WorkerID: workerID, Amount: amount, Description: description}, nil
}

// UpdateFee edits an extra fee of the draft.
func (s *Service) UpdateFee(ctx context.Context, recapID, feeID, amount int64, description string) error {
	if err := s.requireDraft(ctx, recapID); err != nil {
		return err
	}
	description = strings.TrimSpace(description)
	if amount == 0 {
		return ErrInvalidAmount
	}
	if description == "" || len([]rune(description)) > maxDescriptionLength {
		return ErrDescriptionRequired
	}

	owner, _, err := feeOwner(ctx, s.db, feeID)
	if errors.Is(err, sql.ErrNoRows) || owner != recapID {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return updateFee(ctx, s.db, feeID, amount, description)
}

// DeleteFee removes an extra fee from the draft.
func (s *Service) DeleteFee(ctx context.Context, recapID, feeID int64) error {
	if err := s.requireDraft(ctx, recapID); err != nil {
		return err
	}

	owner, _, err := feeOwner(ctx, s.db, feeID)
	if errors.Is(err, sql.ErrNoRows) || owner != recapID {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return deleteFee(ctx, s.db, feeID)
}

func (s *Service) requireDraft(ctx context.Context, recapID int64) error {
	recap, err := s.get(ctx, s.db, recapID)
	if err != nil {
		return err
	}
	if recap.Status != "draft" {
		return ErrLocked
	}
	return nil
}

// Finalize settles every paid worker in one transaction and closes the recap.
func (s *Service) Finalize(ctx context.Context, recapID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin finalize: %w", err)
	}
	defer tx.Rollback()

	recap, err := s.get(ctx, tx, recapID)
	if err != nil {
		return err
	}
	if recap.Status != "draft" {
		return ErrNotDraft
	}

	detail, err := compute(ctx, tx, recap)
	if err != nil {
		return err
	}

	now := s.clock.Now().UTC().Format(time.RFC3339)
	today := clock.FormatDate(s.clock.Today())

	for _, line := range detail.Workers {
		if !line.Paid {
			if err := detachWorkerFees(ctx, tx, recapID, line.WorkerID); err != nil {
				return err
			}
			continue
		}

		attendanceIDs := make([]int64, 0, len(line.Attendance))
		for _, item := range line.Attendance {
			attendanceIDs = append(attendanceIDs, item.ID)
		}
		if err := settleAttendances(ctx, tx, attendanceIDs, recapID); err != nil {
			return err
		}

		advanceIDs := make([]int64, 0, len(line.CashAdvances))
		for _, item := range line.CashAdvances {
			advanceIDs = append(advanceIDs, item.ID)
		}
		if err := settleAdvances(ctx, tx, advanceIDs, recapID); err != nil {
			return err
		}

		if err := insertLine(ctx, tx, recapID, line); err != nil {
			return err
		}

		if line.Net < 0 {
			note := fmt.Sprintf("Saldo sebelumnya (rekap #%d)", recapID)
			if err := insertAutoAdvance(ctx, tx, line.WorkerID, recapID, -line.Net, note, today, now); err != nil {
				return err
			}
		}
	}

	if err := setStatus(ctx, tx, recapID, "final", &now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit finalize: %w", err)
	}
	return nil
}

// Reopen turns the latest final recap back into a draft.
func (s *Service) Reopen(ctx context.Context, recapID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin reopen: %w", err)
	}
	defer tx.Rollback()

	recap, err := s.get(ctx, tx, recapID)
	if err != nil {
		return err
	}
	if recap.Status != "final" {
		return ErrNotFinal
	}

	latest, err := latestFinalID(ctx, tx)
	if err != nil {
		return err
	}
	if latest != recapID {
		return ErrNotLatest
	}

	exists, err := draftExists(ctx, tx)
	if err != nil {
		return err
	}
	if exists {
		return ErrDraftExists
	}

	if err := clearRecapItems(ctx, tx, recapID); err != nil {
		return err
	}
	if err := deleteAutoAdvances(ctx, tx, recapID); err != nil {
		return err
	}
	if err := deleteLines(ctx, tx, recapID); err != nil {
		return err
	}
	if err := setStatus(ctx, tx, recapID, "draft", nil); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reopen: %w", err)
	}
	return nil
}

// DeleteDraft removes a draft and its extra fees.
func (s *Service) DeleteDraft(ctx context.Context, recapID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete recap: %w", err)
	}
	defer tx.Rollback()

	recap, err := s.get(ctx, tx, recapID)
	if err != nil {
		return err
	}
	if recap.Status != "draft" {
		return ErrNotDraft
	}

	if err := deleteFeesForRecap(ctx, tx, recapID); err != nil {
		return err
	}
	if err := deletePaid(ctx, tx, recapID); err != nil {
		return err
	}
	if err := deleteLines(ctx, tx, recapID); err != nil {
		return err
	}
	if err := deleteRecap(ctx, tx, recapID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete recap: %w", err)
	}
	return nil
}

func compute(ctx context.Context, q queryer, recap Recap) (Detail, error) {
	if recap.Status == "final" {
		return computeFinal(ctx, q, recap)
	}

	attendance, err := unsettledAttendances(ctx, q, recap.StartDate, recap.EndDate)
	if err != nil {
		return Detail{}, err
	}
	advances, err := unsettledAdvances(ctx, q, recap.StartDate, recap.EndDate)
	if err != nil {
		return Detail{}, err
	}
	fees, err := feesForRecap(ctx, q, recap.ID)
	if err != nil {
		return Detail{}, err
	}
	paid, err := paidMap(ctx, q, recap.ID)
	if err != nil {
		return Detail{}, err
	}
	workers, err := allWorkers(ctx, q)
	if err != nil {
		return Detail{}, err
	}

	warnings := map[int64]int{}
	if recap.StartDate != nil {
		warnings, err = attendanceWarnings(ctx, q, *recap.StartDate)
		if err != nil {
			return Detail{}, err
		}
	}

	lines := make(map[int64]*WorkerLine)
	ensure := func(workerID int64) *WorkerLine {
		line, ok := lines[workerID]
		if !ok {
			worker := workers[workerID]
			line = &WorkerLine{
				WorkerID:     workerID,
				Name:         worker.Name,
				Role:         worker.Role,
				Paid:         true,
				Attendance:   []AttendanceItem{},
				CashAdvances: []AdvanceItem{},
				ExtraFees:    []ExtraFee{},
			}
			lines[workerID] = line
		}
		return line
	}

	firstDate, lastDate := "", ""
	track := func(date string) {
		if firstDate == "" || date < firstDate {
			firstDate = date
		}
		if lastDate == "" || date > lastDate {
			lastDate = date
		}
	}

	for _, item := range attendance {
		line := ensure(item.WorkerID)
		earned := item.Snapshot * int64(item.Quarters) / 4
		line.Quarters += item.Quarters
		line.Gross += earned
		line.Attendance = append(line.Attendance, AttendanceItem{
			ID:       item.ID,
			Date:     item.Date,
			Quarters: item.Quarters,
			Earned:   earned,
		})
		track(item.Date)
	}

	for _, item := range advances {
		line := ensure(item.WorkerID)
		line.Advances += item.Amount
		line.CashAdvances = append(line.CashAdvances, AdvanceItem{
			ID:     item.ID,
			Date:   item.Date,
			Amount: item.Amount,
			Note:   item.Note,
		})
		track(item.Date)
	}

	for _, fee := range fees {
		line := ensure(fee.WorkerID)
		line.Extras += fee.Amount
		line.ExtraFees = append(line.ExtraFees, ExtraFee{
			ID:          fee.ID,
			WorkerID:    fee.WorkerID,
			Amount:      fee.Amount,
			Description: fee.Description,
		})
	}

	result := Detail{Recap: recap, Workers: make([]WorkerLine, 0, len(lines))}
	for id, line := range lines {
		line.Net = line.Gross + line.Extras - line.Advances
		line.Warning = warnings[id]
		if value, ok := paid[id]; ok {
			line.Paid = value
		}

		result.Totals.Gross += line.Gross
		result.Totals.Advances += line.Advances
		result.Totals.Extras += line.Extras
		result.Totals.Net += line.Net

		result.Workers = append(result.Workers, *line)
	}

	sort.Slice(result.Workers, func(i, j int) bool {
		return strings.ToLower(result.Workers[i].Name) < strings.ToLower(result.Workers[j].Name)
	})

	if firstDate == "" {
		firstDate = recap.CreatedAt
		if len(firstDate) > 10 {
			firstDate = firstDate[:10]
		}
		lastDate = firstDate
	}
	result.PeriodStart = firstDate
	result.PeriodEnd = lastDate

	return result, nil
}

// computeFinal reads the frozen snapshot so exports and the detail screen keep
// working after the items are settled.
func computeFinal(ctx context.Context, q queryer, recap Recap) (Detail, error) {
	lineRows, err := recapLineRows(ctx, q, recap.ID)
	if err != nil {
		return Detail{}, err
	}
	attendance, err := settledAttendances(ctx, q, recap.ID)
	if err != nil {
		return Detail{}, err
	}
	advances, err := settledAdvances(ctx, q, recap.ID)
	if err != nil {
		return Detail{}, err
	}
	fees, err := feesForRecap(ctx, q, recap.ID)
	if err != nil {
		return Detail{}, err
	}

	result := Detail{Recap: recap, Workers: make([]WorkerLine, 0, len(lineRows))}
	index := make(map[int64]int, len(lineRows))
	for _, row := range lineRows {
		index[row.WorkerID] = len(result.Workers)
		result.Workers = append(result.Workers, WorkerLine{
			WorkerID:     row.WorkerID,
			Name:         row.Name,
			Role:         row.Role,
			Quarters:     row.Quarters,
			Gross:        row.Gross,
			Advances:     row.Advances,
			Extras:       row.Extras,
			Net:          row.Net,
			Paid:         true,
			Attendance:   []AttendanceItem{},
			CashAdvances: []AdvanceItem{},
			ExtraFees:    []ExtraFee{},
		})

		result.Totals.Gross += row.Gross
		result.Totals.Advances += row.Advances
		result.Totals.Extras += row.Extras
		result.Totals.Net += row.Net
	}

	firstDate, lastDate := "", ""
	track := func(date string) {
		if firstDate == "" || date < firstDate {
			firstDate = date
		}
		if lastDate == "" || date > lastDate {
			lastDate = date
		}
	}

	for _, item := range attendance {
		position, ok := index[item.WorkerID]
		if !ok {
			continue
		}
		result.Workers[position].Attendance = append(result.Workers[position].Attendance, AttendanceItem{
			ID:       item.ID,
			Date:     item.Date,
			Quarters: item.Quarters,
			Earned:   item.Snapshot * int64(item.Quarters) / 4,
		})
		track(item.Date)
	}
	for _, item := range advances {
		position, ok := index[item.WorkerID]
		if !ok {
			continue
		}
		result.Workers[position].CashAdvances = append(result.Workers[position].CashAdvances, AdvanceItem{
			ID:     item.ID,
			Date:   item.Date,
			Amount: item.Amount,
			Note:   item.Note,
		})
		track(item.Date)
	}
	for _, fee := range fees {
		position, ok := index[fee.WorkerID]
		if !ok {
			continue
		}
		result.Workers[position].ExtraFees = append(result.Workers[position].ExtraFees, ExtraFee{
			ID:          fee.ID,
			WorkerID:    fee.WorkerID,
			Amount:      fee.Amount,
			Description: fee.Description,
		})
	}

	if firstDate == "" {
		firstDate = recap.CreatedAt
		if len(firstDate) > 10 {
			firstDate = firstDate[:10]
		}
		lastDate = firstDate
	}
	result.PeriodStart = firstDate
	result.PeriodEnd = lastDate

	return result, nil
}
