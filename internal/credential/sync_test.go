package credential

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeTx struct {
	execs []string
	args  [][]any
}

func (f *fakeTx) Begin(ctx context.Context) (pgx.Tx, error) { return f, nil }
func (f *fakeTx) Commit(ctx context.Context) error          { return nil }
func (f *fakeTx) Rollback(ctx context.Context) error        { return nil }
func (f *fakeTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (f *fakeTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { return nil }
func (f *fakeTx) LargeObjects() pgx.LargeObjects                               { return pgx.LargeObjects{} }
func (f *fakeTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (f *fakeTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	f.execs = append(f.execs, sql)
	f.args = append(f.args, arguments)
	return pgconn.CommandTag{}, nil
}
func (f *fakeTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (f *fakeTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row { return fakeRow{} }
func (f *fakeTx) Conn() *pgx.Conn                                               { return nil }

type fakeRow struct{}

func (fakeRow) Scan(dest ...any) error { return nil }

func TestSyncOneModelConfigCredential_DeleteOnlyWhenKeyEmpty(t *testing.T) {
	tx := &fakeTx{}
	err := syncOneModelConfigCredentialWithTx(context.Background(), tx, "gpt-4o", "")
	if err != nil {
		t.Fatalf("syncOneModelConfigCredentialWithTx error: %v", err)
	}
	if len(tx.execs) != 1 {
		t.Fatalf("expected 1 exec (delete only), got %d", len(tx.execs))
	}
	if tx.args[0][0] != "gpt-4o" {
		t.Fatalf("delete model arg = %v, want gpt-4o", tx.args[0][0])
	}
}

func TestSyncOneModelConfigCredential_DeleteAndInsertWhenKeyPresent(t *testing.T) {
	tx := &fakeTx{}
	err := syncOneModelConfigCredentialWithTx(context.Background(), tx, "gpt-4o", "sk-test")
	if err != nil {
		t.Fatalf("syncOneModelConfigCredentialWithTx error: %v", err)
	}
	if len(tx.execs) != 2 {
		t.Fatalf("expected 2 execs (delete + insert), got %d", len(tx.execs))
	}
	if tx.args[0][0] != "gpt-4o" {
		t.Fatalf("delete model arg = %v, want gpt-4o", tx.args[0][0])
	}
	if tx.args[1][0] != "gpt-4o" || tx.args[1][1] != "sk-test" {
		t.Fatalf("insert args = %v, want [gpt-4o sk-test]", tx.args[1])
	}
}
