package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	_ "github.com/lib/pq"

	"github.com/banking-assistant/backend/internal/models"
)

var db *sql.DB

func InitPostgres(ctx context.Context, dsn string) error {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return err
	}
	if err := migrate(ctx, conn); err != nil {
		conn.Close()
		return err
	}
	db = conn
	return seedCustomers(ctx)
}

func CustomerStoreName() string {
	return "local-rbci"
}

func ClosePostgres() error {
	if db == nil {
		return nil
	}
	err := db.Close()
	db = nil
	return err
}

func migrate(ctx context.Context, conn *sql.DB) error {
	_, err := conn.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS customers (
  customer_id text PRIMARY KEY,
  national_id text NOT NULL UNIQUE,
  name text NOT NULL,
  age integer NOT NULL,
  gender text NOT NULL DEFAULT '',
  occupation text NOT NULL DEFAULT '',
  employment_type text NOT NULL DEFAULT '',
  customer_type text NOT NULL DEFAULT 'real',
  account_open_date text NOT NULL DEFAULT '',
  is_existing boolean NOT NULL DEFAULT true
);
CREATE TABLE IF NOT EXISTS financial_profiles (
  customer_id text PRIMARY KEY REFERENCES customers(customer_id) ON DELETE CASCADE,
  monthly_income double precision NOT NULL DEFAULT 0,
  account_turnover_3m double precision NOT NULL DEFAULT 0,
  account_turnover_12m double precision NOT NULL DEFAULT 0,
  total_deposits double precision NOT NULL DEFAULT 0,
  active_loans integer NOT NULL DEFAULT 0,
  total_loan_amount double precision NOT NULL DEFAULT 0,
  installment_default integer NOT NULL DEFAULT 0,
  spending_pattern text NOT NULL DEFAULT 'unknown',
  payment_history text NOT NULL DEFAULT 'unknown',
  has_guarantor boolean NOT NULL DEFAULT false
);
CREATE TABLE IF NOT EXISTS risk_assessments (
  customer_id text PRIMARY KEY REFERENCES customers(customer_id) ON DELETE CASCADE,
  risk_level text NOT NULL DEFAULT 'medium',
  risk_score double precision NOT NULL DEFAULT 50,
  reason text NOT NULL DEFAULT '',
  is_cold_start boolean NOT NULL DEFAULT false
);
CREATE TABLE IF NOT EXISTS local_rbci_meta (
  key text PRIMARY KEY,
  value text NOT NULL
);`)
	return err
}

func seedCustomers(ctx context.Context) error {
	if db != nil {
		var seeded string
		err := db.QueryRowContext(ctx, `SELECT value FROM local_rbci_meta WHERE key='local_rbci_seeded'`).Scan(&seeded)
		if err == nil && seeded == "true" {
			return nil
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var count int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM customers`).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			_, err := db.ExecContext(ctx, `INSERT INTO local_rbci_meta (key, value) VALUES ('local_rbci_seeded', 'true') ON CONFLICT (key) DO UPDATE SET value='true'`)
			return err
		}
	}
	for _, identity := range Identities {
		financial := Financials[identity.CustomerID]
		risk := Risks[identity.CustomerID]
		if err := saveCustomer(ctx, models.CustomerRecord{
			Identity: identity, Financial: financial, Risk: risk,
		}, true); err != nil {
			return err
		}
	}
	if db != nil {
		_, err := db.ExecContext(ctx, `INSERT INTO local_rbci_meta (key, value) VALUES ('local_rbci_seeded', 'true') ON CONFLICT (key) DO UPDATE SET value='true'`)
		return err
	}
	return nil
}

func ListCustomers(ctx context.Context) ([]models.CustomerRecord, error) {
	if db == nil {
		out := make([]models.CustomerRecord, 0, len(Identities))
		for _, identity := range Identities {
			out = append(out, models.CustomerRecord{
				Identity:  identity,
				Financial: Financials[identity.CustomerID],
				Risk:      Risks[identity.CustomerID],
			})
		}
		sort.Slice(out, func(i, j int) bool {
			return out[i].Identity.NationalID < out[j].Identity.NationalID
		})
		return out, nil
	}
	rows, err := db.QueryContext(ctx, `
SELECT national_id FROM customers ORDER BY national_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.CustomerRecord
	for rows.Next() {
		var nid string
		if err := rows.Scan(&nid); err != nil {
			return nil, err
		}
		rec, ok, err := GetCustomer(ctx, nid)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, rec)
		}
	}
	return out, rows.Err()
}

func GetCustomer(ctx context.Context, nationalID string) (models.CustomerRecord, bool, error) {
	identity, ok, err := GetIdentity(ctx, nationalID)
	if err != nil || !ok {
		return models.CustomerRecord{}, false, err
	}
	financial, ok, err := GetFinancial(ctx, identity.CustomerID)
	if err != nil || !ok {
		return models.CustomerRecord{}, false, err
	}
	risk, ok, err := GetRisk(ctx, identity.CustomerID)
	if err != nil || !ok {
		return models.CustomerRecord{}, false, err
	}
	return models.CustomerRecord{Identity: identity, Financial: financial, Risk: risk}, true, nil
}

func GetIdentity(ctx context.Context, nationalID string) (models.IdentityProfile, bool, error) {
	if db == nil {
		v, ok := Identities[nationalID]
		return v, ok, nil
	}
	var v models.IdentityProfile
	err := db.QueryRowContext(ctx, `
SELECT customer_id, national_id, name, age, gender, occupation, employment_type, customer_type, account_open_date, is_existing
FROM customers WHERE national_id=$1`, nationalID).Scan(
		&v.CustomerID, &v.NationalID, &v.Name, &v.Age, &v.Gender, &v.Occupation,
		&v.EmploymentType, &v.CustomerType, &v.AccountOpenDate, &v.IsExisting,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.IdentityProfile{}, false, nil
	}
	return v, err == nil, err
}

func GetFinancial(ctx context.Context, customerID string) (models.FinancialProfile, bool, error) {
	if db == nil {
		v, ok := Financials[customerID]
		return v, ok, nil
	}
	var v models.FinancialProfile
	err := db.QueryRowContext(ctx, `
SELECT customer_id, monthly_income, account_turnover_3m, account_turnover_12m, total_deposits,
       active_loans, total_loan_amount, installment_default, spending_pattern, payment_history, has_guarantor
FROM financial_profiles WHERE customer_id=$1`, customerID).Scan(
		&v.CustomerID, &v.MonthlyIncome, &v.AccountTurnover3M, &v.AccountTurnover12M,
		&v.TotalDeposits, &v.ActiveLoans, &v.TotalLoanAmount, &v.InstallmentDefault,
		&v.SpendingPattern, &v.PaymentHistory, &v.HasGuarantor,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.FinancialProfile{}, false, nil
	}
	return v, err == nil, err
}

func GetRisk(ctx context.Context, customerID string) (models.RiskAssessment, bool, error) {
	if db == nil {
		v, ok := Risks[customerID]
		return v, ok, nil
	}
	var v models.RiskAssessment
	err := db.QueryRowContext(ctx, `
SELECT customer_id, risk_level, risk_score, reason, is_cold_start
FROM risk_assessments WHERE customer_id=$1`, customerID).Scan(
		&v.CustomerID, &v.RiskLevel, &v.RiskScore, &v.Reason, &v.IsColdStart,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.RiskAssessment{}, false, nil
	}
	return v, err == nil, err
}

func CreateCustomer(ctx context.Context, rec models.CustomerRecord) error {
	if _, ok, err := GetIdentity(ctx, rec.Identity.NationalID); err != nil {
		return err
	} else if ok {
		return fmt.Errorf("customer already exists")
	}
	if exists, err := customerIDExists(ctx, rec.Identity.CustomerID); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("customer_id already exists")
	}
	return saveCustomer(ctx, rec, false)
}

func UpdateCustomer(ctx context.Context, nationalID string, rec models.CustomerRecord) error {
	if nationalID != rec.Identity.NationalID {
		return fmt.Errorf("national_id mismatch")
	}
	current, ok, err := GetIdentity(ctx, nationalID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("customer not found")
	}
	if current.CustomerID != rec.Identity.CustomerID {
		return fmt.Errorf("customer_id mismatch")
	}
	return saveCustomer(ctx, rec, false)
}

func DeleteCustomer(ctx context.Context, nationalID string) (bool, error) {
	if db == nil {
		identity, ok := Identities[nationalID]
		if !ok {
			return false, nil
		}
		delete(Identities, nationalID)
		delete(Financials, identity.CustomerID)
		delete(Risks, identity.CustomerID)
		return true, nil
	}
	res, err := db.ExecContext(ctx, `DELETE FROM customers WHERE national_id=$1`, nationalID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func customerIDExists(ctx context.Context, customerID string) (bool, error) {
	if db == nil {
		for _, identity := range Identities {
			if identity.CustomerID == customerID {
				return true, nil
			}
		}
		return false, nil
	}
	var found string
	err := db.QueryRowContext(ctx, `SELECT customer_id FROM customers WHERE customer_id=$1`, customerID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func saveCustomer(ctx context.Context, rec models.CustomerRecord, seed bool) error {
	rec.Identity.Occupation = normalizeEmpty(rec.Identity.Occupation, "employee")
	rec.Identity.CustomerType = normalizeEmpty(rec.Identity.CustomerType, "real")
	rec.Financial.CustomerID = rec.Identity.CustomerID
	rec.Risk.CustomerID = rec.Identity.CustomerID
	if db == nil {
		Identities[rec.Identity.NationalID] = rec.Identity
		Financials[rec.Identity.CustomerID] = rec.Financial
		Risks[rec.Identity.CustomerID] = rec.Risk
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	customerSQL := `
INSERT INTO customers (customer_id, national_id, name, age, gender, occupation, employment_type, customer_type, account_open_date, is_existing)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (national_id) DO UPDATE SET
 customer_id=EXCLUDED.customer_id, name=EXCLUDED.name, age=EXCLUDED.age, gender=EXCLUDED.gender,
 occupation=EXCLUDED.occupation, employment_type=EXCLUDED.employment_type, customer_type=EXCLUDED.customer_type,
 account_open_date=EXCLUDED.account_open_date, is_existing=EXCLUDED.is_existing`
	if _, err := tx.ExecContext(ctx, customerSQL,
		rec.Identity.CustomerID, rec.Identity.NationalID, rec.Identity.Name, rec.Identity.Age,
		rec.Identity.Gender, rec.Identity.Occupation, rec.Identity.EmploymentType,
		rec.Identity.CustomerType, rec.Identity.AccountOpenDate, rec.Identity.IsExisting,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO financial_profiles (customer_id, monthly_income, account_turnover_3m, account_turnover_12m, total_deposits,
 active_loans, total_loan_amount, installment_default, spending_pattern, payment_history, has_guarantor)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (customer_id) DO UPDATE SET
 monthly_income=EXCLUDED.monthly_income, account_turnover_3m=EXCLUDED.account_turnover_3m,
 account_turnover_12m=EXCLUDED.account_turnover_12m, total_deposits=EXCLUDED.total_deposits,
 active_loans=EXCLUDED.active_loans, total_loan_amount=EXCLUDED.total_loan_amount,
 installment_default=EXCLUDED.installment_default, spending_pattern=EXCLUDED.spending_pattern,
 payment_history=EXCLUDED.payment_history, has_guarantor=EXCLUDED.has_guarantor`,
		rec.Financial.CustomerID, rec.Financial.MonthlyIncome, rec.Financial.AccountTurnover3M,
		rec.Financial.AccountTurnover12M, rec.Financial.TotalDeposits, rec.Financial.ActiveLoans,
		rec.Financial.TotalLoanAmount, rec.Financial.InstallmentDefault, rec.Financial.SpendingPattern,
		rec.Financial.PaymentHistory, rec.Financial.HasGuarantor,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO risk_assessments (customer_id, risk_level, risk_score, reason, is_cold_start)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (customer_id) DO UPDATE SET
 risk_level=EXCLUDED.risk_level, risk_score=EXCLUDED.risk_score, reason=EXCLUDED.reason,
 is_cold_start=EXCLUDED.is_cold_start`,
		rec.Risk.CustomerID, rec.Risk.RiskLevel, rec.Risk.RiskScore, rec.Risk.Reason, rec.Risk.IsColdStart,
	); err != nil {
		return err
	}
	return tx.Commit()
}

func normalizeEmpty(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
