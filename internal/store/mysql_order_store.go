package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/go-sql-driver/mysql"

	"github.com/hannasotolongo/trade-relay/internal/trading"
)

var ErrOrderVersionConflict = errors.New("order version conflict")

type MySQLOrderStore struct {
	db *sql.DB
}

func NewMySQLOrderStore(db *sql.DB) *MySQLOrderStore {
	return &MySQLOrderStore{
		db: db,
	}
}

func OpenMySQL(
	ctx context.Context,
	dsn string,
) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	return db, nil
}

func (s *MySQLOrderStore) Create(order trading.Order) error {
	version := order.Version
	if version == 0 {
		version = 1
	}

	_, err := s.db.Exec(
		`
		INSERT INTO orders (
			id,
			signal_id,
			account_id,
			broker_account_id,
			broker_order_id,
			symbol,
			side,
			quantity,
			filled_quantity,
			status,
			version,
			created_at,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
		order.ID,
		order.SignalID,
		order.AccountID,
		order.BrokerAccountID,
		nullableString(order.BrokerOrderID),
		order.Symbol,
		order.Side,
		order.Quantity,
		order.FilledQuantity,
		order.Status,
		version,
		order.CreatedAt,
		order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}

	return nil
}

func (s *MySQLOrderStore) Get(
	orderID string,
) (trading.Order, error) {
	var (
		order         trading.Order
		brokerOrderID sql.NullString
	)

	err := s.db.QueryRow(
		`
		SELECT
			id,
			signal_id,
			account_id,
			broker_account_id,
			broker_order_id,
			symbol,
			side,
			quantity,
			filled_quantity,
			status,
			version,
			created_at,
			updated_at
		FROM orders
		WHERE id = ?
		`,
		orderID,
	).Scan(
		&order.ID,
		&order.SignalID,
		&order.AccountID,
		&order.BrokerAccountID,
		&brokerOrderID,
		&order.Symbol,
		&order.Side,
		&order.Quantity,
		&order.FilledQuantity,
		&order.Status,
		&order.Version,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return trading.Order{}, ErrOrderNotFound
	}
	if err != nil {
		return trading.Order{}, fmt.Errorf("get order: %w", err)
	}

	if brokerOrderID.Valid {
		order.BrokerOrderID = brokerOrderID.String
	}

	return order, nil
}

func (s *MySQLOrderStore) Update(order trading.Order) error {
	if order.Version <= 0 {
		return ErrOrderVersionConflict
	}

	result, err := s.db.Exec(
		`
		UPDATE orders
		SET
			broker_order_id = ?,
			filled_quantity = ?,
			status = ?,
			updated_at = ?,
			version = version + 1
		WHERE id = ?
			AND version = ?
		`,
		nullableString(order.BrokerOrderID),
		order.FilledQuantity,
		order.Status,
		order.UpdatedAt,
		order.ID,
		order.Version,
	)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("order rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrOrderVersionConflict
	}

	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}

	return value
}
