package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	logger "refina-web-bff/config/log"
	grpcClient "refina-web-bff/interface/grpc/client"
	"refina-web-bff/internal/types/dto"

	tpb "github.com/MuhammadMiftaa/Refina-Protobuf/transaction"
	wpb "github.com/MuhammadMiftaa/Refina-Protobuf/wallet"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────
// Each fake embeds its interface, so only the methods a test needs are written
// and any other call panics instead of passing silently.

type fakeTransactions struct {
	grpcClient.TransactionClient
	updated  *tpb.UpdateFundTransferRequest
	deleted  string
	created  *tpb.CreateFundTransferRequest
	failWith error
}

func (f *fakeTransactions) UpdateFundTransfer(_ context.Context, req *tpb.UpdateFundTransferRequest) (*tpb.FundTransferResponse, error) {
	f.updated = req
	if f.failWith != nil {
		return nil, f.failWith
	}
	return &tpb.FundTransferResponse{TransferId: req.GetTransferId(), CashOutTransactionId: "out", CashInTransactionId: "in"}, nil
}

func (f *fakeTransactions) DeleteFundTransfer(_ context.Context, id string) (*tpb.FundTransferResponse, error) {
	f.deleted = id
	if f.failWith != nil {
		return nil, f.failWith
	}
	return &tpb.FundTransferResponse{TransferId: id, CashOutTransactionId: "out", CashInTransactionId: "in"}, nil
}

func (f *fakeTransactions) CreateFundTransfer(_ context.Context, req *tpb.CreateFundTransferRequest) (*tpb.FundTransferResponse, error) {
	f.created = req
	return &tpb.FundTransferResponse{TransferId: "t1"}, nil
}

type fakeWallets struct {
	grpcClient.WalletClient
	owners map[string]string
}

func (f *fakeWallets) GetWalletByID(_ context.Context, id string) (*wpb.Wallet, error) {
	owner, ok := f.owners[id]
	if !ok {
		return nil, status.Error(codes.NotFound, "wallet not found")
	}
	return &wpb.Wallet{Id: id, UserId: owner}, nil
}

type fakeCache struct {
	mu       sync.Mutex
	patterns []string
	keys     []string
}

func (f *fakeCache) Get(context.Context, string) ([]byte, error)              { return nil, nil }
func (f *fakeCache) Set(context.Context, string, []byte, time.Duration) error { return nil }
func (f *fakeCache) Close() error                                             { return nil }
func (f *fakeCache) Delete(_ context.Context, keys ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.keys = append(f.keys, keys...)
	return nil
}
func (f *fakeCache) DeleteByPattern(_ context.Context, pattern string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.patterns = append(f.patterns, pattern)
	return nil
}

// ── Harness ───────────────────────────────────────────────────────────────────

func TestMain(m *testing.M) {
	// Handlers log through the package logger; a quiet one keeps test output clean.
	logger.Log = logrus.New()
	logger.Log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

const testUser = "user-1"

func transferApp(tc *fakeTransactions, wc *fakeWallets, c *fakeCache) *fiber.App {
	h := NewTransactionHandler(tc, wc, c)
	app := fiber.New()
	app.Use(func(ctx *fiber.Ctx) error {
		ctx.Locals("user_data", dto.UserData{ID: testUser, Email: "u@example.com"})
		return ctx.Next()
	})
	app.Post("/transactions/transfer", h.CreateFundTransfer)
	app.Put("/transactions/transfer/:transferId", h.UpdateFundTransfer)
	app.Delete("/transactions/transfer/:transferId", h.DeleteFundTransfer)
	return app
}

func call(t *testing.T, app *fiber.App, method, path, body string) (int, dto.APIResponse) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	raw, _ := io.ReadAll(res.Body)
	var out dto.APIResponse
	_ = json.Unmarshal(raw, &out)
	return res.StatusCode, out
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestUpdateFundTransfer_PassesTheWholeTransferAndClearsCaches(t *testing.T) {
	tc, c := &fakeTransactions{}, &fakeCache{}
	app := transferApp(tc, &fakeWallets{}, c)

	code, res := call(t, app, "PUT", "/transactions/transfer/t1",
		`{"from_wallet_id":"a","to_wallet_id":"b","amount":150000,"admin_fee":2500,"transaction_date":"2026-11-01T10:00:00Z","description":"Kos"}`)

	if code != 200 || !res.Status {
		t.Fatalf("got %d %+v", code, res)
	}
	if tc.updated.GetTransferId() != "t1" || tc.updated.GetAmount() != 150000 || tc.updated.GetAdminFee() != 2500 || tc.updated.GetDescription() != "Kos" {
		t.Fatalf("unexpected request %+v", tc.updated)
	}
	// Invalidation happens before the response, so the next read is fresh.
	if len(c.patterns) == 0 || len(c.keys) < 2 {
		t.Fatalf("caches not cleared: patterns=%v keys=%v", c.patterns, c.keys)
	}
}

func TestUpdateFundTransfer_RequiresWalletsAndDate(t *testing.T) {
	tc := &fakeTransactions{}
	app := transferApp(tc, &fakeWallets{}, &fakeCache{})

	code, _ := call(t, app, "PUT", "/transactions/transfer/t1", `{"from_wallet_id":"a","amount":1}`)

	if code != 400 || tc.updated != nil {
		t.Fatalf("expected 400 without a service call, got %d", code)
	}
}

func TestUpdateFundTransfer_MapsServiceErrors(t *testing.T) {
	tc := &fakeTransactions{failWith: status.Error(codes.Unknown, "update fund transfer [id=t1]: insufficient wallet balance [wallet_id=a]")}
	app := transferApp(tc, &fakeWallets{}, &fakeCache{})

	code, res := call(t, app, "PUT", "/transactions/transfer/t1",
		`{"from_wallet_id":"a","to_wallet_id":"b","amount":1,"transaction_date":"2026-11-01T10:00:00Z"}`)

	if code != 400 || res.Message != "Insufficient wallet balance" {
		t.Fatalf("got %d %q", code, res.Message)
	}
}

func TestDeleteFundTransfer_NotFound(t *testing.T) {
	tc := &fakeTransactions{failWith: status.Error(codes.Unknown, "delete fund transfer [id=x]: transfer not found [id=x]")}
	app := transferApp(tc, &fakeWallets{}, &fakeCache{})

	code, res := call(t, app, "DELETE", "/transactions/transfer/x", "")

	if code != 404 || res.Message != "Transfer not found" || tc.deleted != "x" {
		t.Fatalf("got %d %q", code, res.Message)
	}
}

func TestCreateFundTransfer_RefusesSomeoneElsesWallet(t *testing.T) {
	tc := &fakeTransactions{}
	wc := &fakeWallets{owners: map[string]string{"mine": testUser, "theirs": "user-2"}}
	app := transferApp(tc, wc, &fakeCache{})

	code, _ := call(t, app, "POST", "/transactions/transfer",
		`{"from_wallet_id":"theirs","to_wallet_id":"mine","amount":1,"transaction_date":"2026-11-01T10:00:00Z"}`)

	if code != 403 || tc.created != nil {
		t.Fatalf("expected 403 without a transfer, got %d", code)
	}
}

func TestCreateFundTransfer_OwnWalletsGoThrough(t *testing.T) {
	tc := &fakeTransactions{}
	wc := &fakeWallets{owners: map[string]string{"a": testUser, "b": testUser}}
	app := transferApp(tc, wc, &fakeCache{})

	code, _ := call(t, app, "POST", "/transactions/transfer",
		`{"from_wallet_id":"a","to_wallet_id":"b","amount":1,"transaction_date":"2026-11-01T10:00:00Z"}`)

	if code != 201 || tc.created == nil {
		t.Fatalf("expected 201, got %d", code)
	}
}
