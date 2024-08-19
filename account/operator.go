package account

import (
	"account-operator/code"
	"account-operator/market"
	"account-operator/postgresql"
	"account-operator/price"
	"account-operator/protocol"
	"account-operator/quit"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"math/big"
	"regexp"
	"strings"
)

type Account interface {
	ID() string
	Name() string
	Currency() string
}

type Operator interface {
	Start()
	Close()
	CreateAccount(userID string, currency string, accountName string) (Account, error)
	ListAccount(str string) ([]Account, error)
	Deposit(accountID string, amount string) error
	Withdraw(accountID string, amount string) error
	DeleteAccount(accountID string) error
	// MarketOrder Use the currency from fromAccountID to purchase the currency of toAccountID with the amount
	MarketOrder(req TradeOrderRequest) error
}

func NewOperator(msgs price.Delivers, marketInst market.Market) Operator {
	return &operator{
		marketInst:    marketInst,
		priceDelivers: msgs,
		stop:          make(chan struct{}, 1),
	}
}

type operator struct {
	marketInst    market.Market
	priceDelivers price.Delivers
	stop          chan struct{}
}

type TradeOrderRequest struct {
	BaseCurrencyAccount  string `json:"base_account_id" binding:"required"`
	QuoteCurrencyAccount string `json:"quote_account_id" binding:"required"`
	Symbol               string `json:"symbol" binding:"required"`
	Side                 string `json:"side" binding:"required"`
	Type                 string `json:"type" binding:"required"`
	Quantity             string `json:"quantity" binding:"required"`
	// could be ignored for market order
	Price string `json:"price"`
}

func (o *operator) MarketOrder(req TradeOrderRequest) error {
	dbClient := postgresql.GetClient()
	getCurrencyQuery := "SELECT currency FROM account WHERE id = $1"
	var baseCurrency string
	err := dbClient.QueryRow(getCurrencyQuery, req.BaseCurrencyAccount).Scan(&baseCurrency)
	if err != nil {
		return fmt.Errorf("failed to get fromAccountID currency: %w", err)
	}
	var quoteCurrency string
	err = dbClient.QueryRow(getCurrencyQuery, req.QuoteCurrencyAccount).Scan(&quoteCurrency)
	if err != nil {
		return fmt.Errorf("failed to get toAccountID currency: %w", err)
	}

	symbol := req.Symbol
	shouldBeSymbol := fmt.Sprintf("%s%s", baseCurrency, quoteCurrency)
	if symbol != shouldBeSymbol {
		return fmt.Errorf("account currency mismatch: %s != %s", symbol, shouldBeSymbol)
	}

	switch req.Type {
	case "market":
		err = isValidAmount(req.Quantity)
		if err != nil {
			return fmt.Errorf("quantity should be a valid numeric value: %w", err)
		}
		err = o.marketInst.MarketOrder(symbol, o.marketOrderCallBack(req.BaseCurrencyAccount, req.QuoteCurrencyAccount, req.Quantity, req.Side))
		if err != nil {
			return err
		}
		return nil

	//case "limit":
	//	return o.marketInst.LimitOrder(symbol, req.BaseCurrencyAccount, req.QuoteCurrencyAccount, req.Side, req.Quantity, req.Price)
	default:
		return fmt.Errorf("invalid type: %s", req.Type)
	}
}

func (o *operator) marketOrderCallBack(baseCurrencyAccountID string, quoteCurrencyAccountID string, quantity string, side string) func(price string) {
	return func(price string) {
		var priceBig big.Float
		priceBig.SetString(price)
		var quantityBig big.Float
		quantityBig.SetString(quantity)
		var amountBig big.Float
		amountBig.Mul(&priceBig, &quantityBig)

		switch side {
		case "buy":
			var exchangeRate big.Float
			exchangeRate.Quo(&amountBig, &quantityBig)
			dbClient := postgresql.GetClient()
			tx, err := dbClient.Begin()
			if err != nil {
				logrus.Errorf("failed to start transaction: %s", err)
				return
			}
			defer tx.Rollback()

			transferLogQuery := "INSERT INTO transfer_log (from_account, to_account, exchange_rate, from_amount , to_amount) VALUES ($1, $2, $3, $4, $5);"
			_, err = tx.Exec(transferLogQuery, quoteCurrencyAccountID, baseCurrencyAccountID, exchangeRate.String(), amountBig.String(), quantityBig.String())
			if err != nil {
				logrus.Errorf("failed to log transfer: %s", err)
				return
			}

			updateAccountQuery := "UPDATE account SET balance = balance + $1 WHERE id = $2;"
			_, err = tx.Exec(updateAccountQuery, quantity, baseCurrencyAccountID)
			if err != nil {
				logrus.Errorf("failed to update account: %s", err)
				return
			}

			_, err = tx.Exec(updateAccountQuery, fmt.Sprintf("-%s", amountBig.String()), quoteCurrencyAccountID)
			if err != nil {
				logrus.Errorf("failed to update account: %s", err)
				return
			}

			err = tx.Commit()
			if err != nil {
				logrus.Errorf("failed to commit transaction: %s", err)
				return
			}
		case "sell":
			panic("not implemented")
		}
	}
}

func (o *operator) Withdraw(accountID string, amount string) error {
	err := isValidAmount(amount)
	if err != nil {
		return fmt.Errorf("failed to withdraw: %w", err)
	}

	dbClient := postgresql.GetClient()

	// Start a transaction
	tx, err := dbClient.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	deleted, err := checkIfDeleted(tx, accountID)
	if err != nil {
		return err
	}
	if deleted {
		return fmt.Errorf("%w : account: %s", code.AccountDeleted, accountID)
	}

	// Prepare the SQL statement to insert a log entry
	logQuery := `
		INSERT INTO deposit_and_withdrawal_log (account, amount)
		VALUES ($1, $2);
	`

	// Execute the SQL statement to insert a log entry
	_, err = tx.Exec(logQuery, accountID, fmt.Sprintf("-%s", amount))
	if err != nil {
		return fmt.Errorf("failed to log withdrawal: %w", err)
	}

	// Prepare the SQL statement to update the account balance
	updateQuery := `
		UPDATE account
		SET balance = balance - $1
		WHERE id = $2;
	`

	// Execute the SQL statement to update the account balance
	_, err = tx.Exec(updateQuery, amount, accountID)
	if err != nil {
		return fmt.Errorf("failed to withdraw: %w", err)
	}

	// Commit the transaction
	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("failed to commit transaction: %w", commitErr)
	}

	// Return nil
	return nil
}

func (o *operator) Deposit(accountID string, amount string) error {
	err := isValidAmount(amount)
	if err != nil {
		return fmt.Errorf("failed to deposit: %w", err)
	}

	dbClient := postgresql.GetClient()

	// Start a transaction
	tx, err := dbClient.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	deleted, err := checkIfDeleted(tx, accountID)
	if err != nil {
		return err
	}
	if deleted {
		return fmt.Errorf("%w : account: %s", code.AccountDeleted, accountID)
	}

	// Prepare the SQL statement to insert a log entry
	logQuery := `
		INSERT INTO deposit_and_withdrawal_log (account, amount)
		VALUES ($1, $2);
	`

	// Execute the SQL statement to insert a log entry
	_, err = tx.Exec(logQuery, accountID, amount)
	if err != nil {
		return fmt.Errorf("failed to log deposit: %w", err)
	}

	// Prepare the SQL statement to update the account balance
	updateQuery := `
		UPDATE account
		SET balance = balance + $1
		WHERE id = $2;
	`

	// Execute the SQL statement to update the account balance
	_, err = tx.Exec(updateQuery, amount, accountID)
	if err != nil {
		return fmt.Errorf("failed to deposit: %w", err)
	}

	// Commit the transaction
	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("failed to commit transaction: %w", commitErr)
	}

	// Return nil
	return nil
}

func (o *operator) DeleteAccount(accountID string) error {
	dbClient := postgresql.GetClient()

	// Start a transaction
	tx, err := dbClient.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare the SQL statement to mark the account as deleted
	deleteQuery := `
		UPDATE account
		SET is_deleted = TRUE
		WHERE id = $1;
	`

	// Execute the SQL statement to mark the account as deleted
	_, err = tx.Exec(deleteQuery, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	// Commit the transaction
	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("failed to commit transaction: %w", commitErr)
	}

	// Return nil
	return nil
}

func (o *operator) ListAccount(str string) ([]Account, error) {
	dbClient := postgresql.GetClient()

	// Start a transaction
	tx, err := dbClient.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare the SQL statement
	query := `
		SELECT id, currency, name
		FROM account
		WHERE owner = (SELECT id FROM public.users WHERE id = $1) AND is_deleted = FALSE;
	`

	// Execute the SQL statement
	rows, err := tx.Query(query, str)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	// Parse the result
	var accountInstSlice []Account
	for rows.Next() {
		var accountInst account
		err := rows.Scan(&accountInst.id, &accountInst.currency, &accountInst.name)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accountInstSlice = append(accountInstSlice, &accountInst)
	}

	// Commit the transaction
	if commitErr := tx.Commit(); commitErr != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", commitErr)
	}

	// Return the account slice
	return accountInstSlice, nil
}

func (o *operator) CreateAccount(userID string, currency string, accountName string) (Account, error) {
	dbClient := postgresql.GetClient()

	// Start a transaction
	tx, err := dbClient.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if the currency exists
	var currencyExists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM public.currency WHERE code = $1)", currency).Scan(&currencyExists)
	if err != nil {
		return nil, fmt.Errorf("failed to check currency: %w", err)
	}
	if !currencyExists {
		return nil, fmt.Errorf("%w : currency: %s", code.CurrencyNotFound, currency)
	}

	// Prepare the SQL statement
	query := `
		INSERT INTO account (currency, name, owner)
		VALUES ($1, $2, (SELECT id FROM public.users WHERE id = $3))
		RETURNING id, currency, name;
	`

	// Execute the SQL statement
	var accountInst account
	err = tx.QueryRow(query, currency, accountName, userID).Scan(&accountInst.id, &accountInst.currency, &accountInst.name)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	// Commit the transaction
	if commitErr := tx.Commit(); commitErr != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", commitErr)
	}

	// Return the created account
	return &accountInst, nil
}

func (o *operator) Start() {
	for symbol, delivery := range o.priceDelivers {
		g := quit.ReportGoroutine(fmt.Sprintf("operator for symbol %s", symbol))
		go func(g quit.Goroutine) {
			defer g.Done()
			o.run(symbol, delivery)
		}(g)
	}
	return
}

func (o *operator) run(symbol string, delivery <-chan amqp091.Delivery) {
	for {
		select {
		case <-o.stop:
			logrus.Infof("Stopping operator for symbol %s", symbol)
			return
		case msg, ok := <-delivery:
			if !ok {
				logrus.Infof("Stopping operator for symbol %s because delivery channel is closed", symbol)
				return
			}
			o.newCoinPriceMessage(msg.Body)
		}
	}
}

func (o *operator) newCoinPriceMessage(body []byte) {
	var coinPriceBody protocol.CoinPriceBody
	err := json.Unmarshal(body, &coinPriceBody)
	if err != nil {
		logrus.Errorf("Failed to unmarshal event: %s", err)
		return
	}

	o.marketInst.UpdatePrice(coinPriceBody.WsTradeEvent.Symbol, coinPriceBody.WsTradeEvent.Price)
}

func (o *operator) Close() {
	close(o.stop)
}

func checkIfDeleted(tx *sql.Tx, accountID string) (isDeleted bool, err error) {
	err = tx.QueryRow("SELECT is_deleted FROM account WHERE id = $1", accountID).Scan(&isDeleted)
	if err != nil {
		return false, fmt.Errorf("failed to check if account is deleted: %w", err)
	}
	return isDeleted, nil
}

func isValidAmount(amount string) error {
	return isValidNumeric(amount, 21, 8)
}

func isValidNumeric(amount string, precision int, scale int) error {
	// Define a regular expression that matches the numeric(precision-scale, scale) format
	regexPattern := fmt.Sprintf(`^\d{1,%d}(\.\d{1,%d})?$`, precision-scale, scale)

	// Validate the amount string against the regular expression
	matched, _ := regexp.MatchString(regexPattern, amount)
	if !matched {
		return fmt.Errorf("invalid value: %s must be a valid numeric(%d,%d) value", amount, precision, scale)
	}

	splitAmount := strings.Split(amount, ".")

	// Ensure that the value fits within the desired precision and scale
	intPart := splitAmount[0]
	fracPart := "0"
	if len(splitAmount) > 1 {
		fracPart = splitAmount[1]
	}
	if len(intPart) > (precision-scale) || len(fracPart) > scale {
		return fmt.Errorf("invalid value: %s must be a valid numeric(%d,%d) value", amount, precision, scale)
	}

	return nil
}
