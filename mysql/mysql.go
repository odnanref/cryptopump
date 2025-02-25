package mysql

import (
	"database/sql"
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/aleibovici/cryptopump/functions"
	"github.com/aleibovici/cryptopump/logger"
	"github.com/aleibovici/cryptopump/types"

	_ "github.com/go-sql-driver/mysql" // This blank entry is required to enable mysql connectivity
)

// DBInit export
/* This function initializes GCP mysql database connectivity */
func DBInit() *sql.DB {

	var db *sql.DB
	var err error

	// If the optional DB_TCP_HOST environment variable is set, it contains
	// the IP address and port number of a TCP connection pool to be created,
	// such as "127.0.0.1:3306". If DB_TCP_HOST is not set, a Unix socket
	// connection pool will be created instead.
	if os.Getenv("DB_TCP_HOST") != "" {

		if db, err = InitTCPConnectionPool(); err != nil {

			defer os.Exit(1)

		}

	} else {

		if db, err = InitSocketConnectionPool(); err != nil {

			defer os.Exit(1)

		}

	}

	/* Conditional defer logging when there is an error retriving data */
	defer func() {
		if err != nil {
			logger.LogEntry{ /* Log Entry */
				Config:   nil,
				Market:   nil,
				Session:  nil,
				Order:    &types.Order{},
				Message:  functions.GetFunctionName() + " - " + err.Error(),
				LogLevel: "DebugLevel",
			}.Do()
		}
	}()

	return db

}

// InitSocketConnectionPool initializes a Unix socket connection pool for
// a Cloud SQL instance of SQL Server.
func InitSocketConnectionPool() (*sql.DB, error) {

	var err error
	var dbPool *sql.DB

	// [START cloud_sql_mysql_databasesql_create_socket]
	var (
		dbUser                 = functions.MustGetenv("DB_USER")
		dbPwd                  = functions.MustGetenv("DB_PASS")
		instanceConnectionName = functions.MustGetenv("INSTANCE_CONNECTION_NAME")
		dbName                 = functions.MustGetenv("DB_NAME")
	)

	socketDir, isSet := os.LookupEnv("DB_SOCKET_DIR")
	if !isSet {
		socketDir = "/cloudsql"
	}

	var dbURI = fmt.Sprintf("%s:%s@unix(/%s/%s)/%s?parseTime=true", dbUser, dbPwd, socketDir, instanceConnectionName, dbName)

	// dbPool is the pool of database connections.
	if dbPool, err = sql.Open("mysql", dbURI); err != nil {

		return nil, fmt.Errorf("sql.Open: %v", err)

	}

	// [START_EXCLUDE]
	configureConnectionPool(dbPool)
	// [END_EXCLUDE]

	return dbPool, nil
	// [END cloud_sql_mysql_databasesql_create_socket]
}

// configureConnectionPool sets database connection pool properties.
// For more information, see https://golang.org/pkg/database/sql
func configureConnectionPool(dbPool *sql.DB) {
	// [START cloud_sql_mysql_databasesql_limit]

	// Set maximum number of connections in idle connection pool.
	dbPool.SetMaxIdleConns(5)

	// Set maximum number of open connections to the database.
	dbPool.SetMaxOpenConns(7)

	// [END cloud_sql_mysql_databasesql_limit]

	// [START cloud_sql_mysql_databasesql_lifetime]

	// Set Maximum time (in seconds) that a connection can remain open.
	dbPool.SetConnMaxLifetime(1800)

	// [END cloud_sql_mysql_databasesql_lifetime]
}

// InitTCPConnectionPool initializes a TCP connection pool for a Cloud SQL
// instance of SQL Server.
func InitTCPConnectionPool() (*sql.DB, error) {

	var err error
	var dbPool *sql.DB

	// [START cloud_sql_mysql_databasesql_create_tcp]
	var (
		dbUser    = functions.MustGetenv("DB_USER")
		dbPwd     = functions.MustGetenv("DB_PASS")
		dbTCPHost = functions.MustGetenv("DB_TCP_HOST")
		dbPort    = functions.MustGetenv("DB_PORT")
		dbName    = functions.MustGetenv("DB_NAME")
	)

	var dbURI = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPwd, dbTCPHost, dbPort, dbName)

	// dbPool is the pool of database connections.

	if dbPool, err = sql.Open("mysql", dbURI); err != nil {

		return nil, fmt.Errorf("sql.Open: %v", err)

	}

	// [START_EXCLUDE]
	configureConnectionPool(dbPool)
	// [END_EXCLUDE]

	return dbPool, nil
	// [END cloud_sql_mysql_databasesql_create_tcp]
}

// SaveOrder Save order to database
func SaveOrder(
	sessionData *types.Session,
	order *types.Order,
	orderIDSource int64, /* OrderIDSource */
	orderPrice float64 /* OrderPrice */) (err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.SaveOrder(?,?,?,?,?,?,?,?,?,?,?,?)",
		order.ClientOrderID,
		order.CumulativeQuoteQuantity,
		order.ExecutedQuantity,
		order.OrderID,
		orderIDSource, /* OrderIDSource */
		orderPrice,
		order.Side,
		order.Status,
		order.Symbol,
		order.TransactTime,
		sessionData.ThreadID,
		sessionData.ThreadIDSession); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:  nil,
			Market:  nil,
			Session: sessionData,
			Order: &types.Order{
				OrderID: order.OrderID,
				Price:   orderPrice,
			},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return err

	}

	defer rows.Close() /* Close rows */

	return nil

}

// UpdateOrder Update order
func UpdateOrder(
	sessionData *types.Session,
	OrderID int64,
	CumulativeQuoteQuantity float64,
	ExecutedQuantity float64,
	Price float64,
	Status string) (err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.UpdateOrder(?,?,?,?,?)",
		OrderID,
		CumulativeQuoteQuantity,
		ExecutedQuantity,
		Price,
		Status); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:  nil,
			Market:  nil,
			Session: sessionData,
			Order: &types.Order{
				OrderID: int64(OrderID),
				Price:   Price,
			},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return err

	}

	defer rows.Close() /* Close rows */

	return nil

}

// UpdateSession Update existing session on Session table
func UpdateSession(
	configData *types.Config,
	sessionData *types.Session) (err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.UpdateSession(?,?,?,?,?,?,?)",
		sessionData.ThreadID,
		sessionData.ThreadIDSession,
		configData.ExchangeName,
		sessionData.SymbolFiat,
		sessionData.SymbolFiatFunds,
		sessionData.DiffTotal,
		sessionData.Status); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   configData,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return err

	}

	defer rows.Close() /* Close rows */

	return nil

}

// UpdateGlobal Update global settings
func UpdateGlobal(
	sessionData *types.Session) (err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.UpdateGlobal(?,?,?,?)",
		sessionData.Global.Profit,
		sessionData.Global.ProfitNet,
		sessionData.Global.ProfitPct,
		time.Now().Unix()); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return err

	}

	defer rows.Close() /* Close rows */

	return nil

}

// SaveGlobal Save initial global settings
func SaveGlobal(
	sessionData *types.Session) (err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.SaveGlobal(?,?,?,?)",
		sessionData.Global.Profit,
		sessionData.Global.ProfitNet,
		sessionData.Global.ProfitPct,
		time.Now().Unix()); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return err

	}

	defer rows.Close() /* Close rows */

	return nil

}

// SaveSession Save new session to Session table.
func SaveSession(
	configData *types.Config,
	sessionData *types.Session) (err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.SaveSession(?,?,?,?,?,?,?)",
		sessionData.ThreadID,
		sessionData.ThreadIDSession,
		configData.ExchangeName,
		sessionData.SymbolFiat,
		sessionData.SymbolFiatFunds,
		sessionData.DiffTotal,
		sessionData.Status); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   configData,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return err

	}

	defer rows.Close() /* Close rows */

	return nil

}

// DeleteSession Delete session from Session table
func DeleteSession(
	sessionData *types.Session) (err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.DeleteSession(?)",
		sessionData.ThreadID); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return err

	}

	defer rows.Close() /* Close rows */

	return nil

}

// GetSessionStatus check for system error status
func GetSessionStatus(
	sessionData *types.Session) (threadID string, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetSessionStatus()"); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return "", err

	}

	for rows.Next() {
		var status bool
		err = rows.Scan(&threadID, &status)
		if status {
			return
		}
	}

	defer rows.Close() /* Close rows */

	return threadID, nil

}

// SaveThreadTransaction Save Thread cycle to database
func SaveThreadTransaction(
	sessionData *types.Session,
	OrderID int64,
	CumulativeQuoteQuantity float64,
	Price float64,
	ExecutedQuantity float64) (err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.SaveThreadTransaction(?,?,?,?,?,?)",
		sessionData.ThreadID,
		sessionData.ThreadIDSession,
		OrderID,
		CumulativeQuoteQuantity,
		Price,
		ExecutedQuantity); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:  nil,
			Market:  nil,
			Session: sessionData,
			Order: &types.Order{
				OrderID: int64(OrderID),
				Price:   Price,
			},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return err

	}

	defer rows.Close() /* Close rows */

	return nil

}

// DeleteThreadTransactionByOrderID function
func DeleteThreadTransactionByOrderID(
	sessionData *types.Session,
	orderID int64) (err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.DeleteThreadTransactionByOrderID(?)",
		orderID); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:  nil,
			Market:  nil,
			Session: sessionData,
			Order: &types.Order{
				OrderID: int64(orderID),
			},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return err

	}

	defer rows.Close() /* Close rows */

	return nil

}

// GetThreadTransactionCount Get Thread count
func GetThreadTransactionCount(
	sessionData *types.Session) (count int, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetThreadTransactionCount(?)",
		sessionData.ThreadID); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return 0, err

	}

	for rows.Next() {
		err = rows.Scan(&count)
	}

	defer rows.Close() /* Close rows */

	return count, err

}

// GetLastOrderTransactionPrice Get time for last transaction the ThreadID
func GetLastOrderTransactionPrice(
	sessionData *types.Session,
	Side string) (price float64, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetLastOrderTransactionPrice(?,?)",
		sessionData.ThreadID,
		Side); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return 0, err

	}

	for rows.Next() {
		err = rows.Scan(&price)
	}

	defer rows.Close() /* Close rows */

	return price, err

}

// GetLastOrderTransactionSide Get Side for last transaction the ThreadID
func GetLastOrderTransactionSide(
	sessionData *types.Session) (side string, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetLastOrderTransactionSide(?)",
		sessionData.ThreadID); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return "", err

	}

	for rows.Next() {
		err = rows.Scan(&side)
	}

	defer rows.Close() /* Close rows */

	return side, err

}

// GetOrderTransactionSideLastTwo function
func GetOrderTransactionSideLastTwo(
	sessionData *types.Session) (side1 string, side2 string, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetOrderTransactionSideLastTwo(?)",
		sessionData.ThreadID); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return "", "", err

	}

	for rows.Next() {
		err = rows.Scan(&side1, &side2)
	}

	defer rows.Close() /* Close rows */

	return side1, side2, err

}

// GetOrderSymbol Get symbol for ThreadID
func GetOrderSymbol(
	sessionData *types.Session) (symbol string, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetOrderSymbol(?)",
		sessionData.ThreadID); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return "", err

	}

	for rows.Next() {
		err = rows.Scan(&symbol)
	}

	defer rows.Close() /* Close rows */

	return symbol, err

}

// GetThreadTransactionDistinct Get Thread Distinct
func GetThreadTransactionDistinct(
	sessionData *types.Session) (threadID string, threadIDSession string, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetThreadTransactionDistinct()"); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return "", "", err

	}

	for rows.Next() {
		err = rows.Scan(
			&threadID,
			&threadIDSession)

		/* Verify if lock file for thread exist. If lock file doesn't exist leave function with empty thread */
		if _, err := os.Stat(threadID + ".lock"); err != nil {

			break

		} else {

			threadID = ""
			threadIDSession = ""

		}

	}

	defer rows.Close() /* Close rows */

	return threadID, threadIDSession, err

}

// GetOrderTransactionPending Get 1 order with pending FILLED status
func GetOrderTransactionPending(
	sessionData *types.Session) (order types.Order, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetOrderTransactionPending(?)",
		sessionData.ThreadID); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return types.Order{}, err

	}

	for rows.Next() {
		err = rows.Scan(
			&order.OrderID,
			&order.Symbol)
	}

	defer rows.Close() /* Close rows */

	return order, err

}

// GetThreadTransactionByPrice retrieve lowest price order from Thread database
func GetThreadTransactionByPrice(
	marketData *types.Market,
	sessionData *types.Session) (order types.Order, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetThreadTransactionByPrice(?,?)",
		sessionData.ThreadID,
		marketData.Price); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return types.Order{}, err

	}

	for rows.Next() {
		err = rows.Scan(
			&order.CumulativeQuoteQuantity,
			&order.OrderID,
			&order.Price,
			&order.ExecutedQuantity,
			&order.TransactTime)
	}

	defer rows.Close() /* Close rows */

	return order, err

}

// GetThreadTransactionByPriceHigher function returns the highert Thread order above a certain treshold.
// It is used for STOPLOSS Loss as ratio that should trigger a sale
func GetThreadTransactionByPriceHigher(
	marketData *types.Market,
	sessionData *types.Session) (order types.Order, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetThreadTransactionByPriceHigher(?,?)",
		sessionData.ThreadID,
		marketData.Price); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   marketData,
			Session:  sessionData,
			Order:    nil,
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return order, err

	}

	for rows.Next() {
		err = rows.Scan(
			&order.CumulativeQuoteQuantity,
			&order.OrderID,
			&order.Price,
			&order.ExecutedQuantity,
			&order.TransactTime)
	}

	defer rows.Close() /* Close rows */

	return order, err

}

// GetThreadLastTransaction function returns the last BUY transaction for a Thread
func GetThreadLastTransaction(
	sessionData *types.Session) (order types.Order, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetThreadLastTransaction(?)",
		sessionData.ThreadID); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return types.Order{}, err

	}

	for rows.Next() {
		err = rows.Scan(
			&order.CumulativeQuoteQuantity,
			&order.OrderID,
			&order.Price,
			&order.ExecutedQuantity,
			&order.TransactTime)
	}

	defer rows.Close() /* Close rows */

	return order, err

}

// GetOrderByOrderID Return order by OrderID (uses ThreadID as filter)
func GetOrderByOrderID(
	sessionData *types.Session) (order types.Order, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetOrderByOrderID(?,?)",
		sessionData.ForceSellOrderID,
		sessionData.ThreadID); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:  nil,
			Market:  nil,
			Session: sessionData,
			Order: &types.Order{
				OrderID: sessionData.ForceSellOrderID,
			},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return types.Order{}, err

	}

	for rows.Next() {
		err = rows.Scan(
			&order.OrderID,
			&order.Price,
			&order.ExecutedQuantity,
			&order.CumulativeQuoteQuantity,
			&order.TransactTime)
	}

	defer rows.Close() /* Close rows */

	return order, err

}

// GetThreadTransactiontUpmarketPriceCount function
func GetThreadTransactiontUpmarketPriceCount(
	sessionData *types.Session,
	price float64) (count int, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetThreadTransactiontUpmarketPriceCount(?,?)",
		sessionData.ThreadID,
		price); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return 0, err

	}

	for rows.Next() {
		err = rows.Scan(&count)
	}

	defer rows.Close() /* Close rows */

	return count, err

}

// GetOrderTransactionCount Retrieve transaction count by Side and minutes
func GetOrderTransactionCount(
	sessionData *types.Session,
	side string) (count float64, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetOrderTransactionCount(?,?,?)",
		sessionData.ThreadID,
		side,
		(60 * -1)); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return 0, err

	}

	for rows.Next() {
		err = rows.Scan(&count)
	}

	defer rows.Close() /* Close rows */

	return count, err

}

// GetThreadTransactionByThreadID  Retrieve transaction count by Side and minutes
func GetThreadTransactionByThreadID(
	sessionData *types.Session) (orders []types.Order, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	order := types.Order{}

	if rows, err = sessionData.Db.Query("call cryptopump.GetThreadTransactionByThreadID(?)",
		sessionData.ThreadID); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return nil, err

	}

	for rows.Next() {

		var orderID int64
		var cumulativeQuoteQty, price, executedQuantity string
		err = rows.Scan(&orderID, &cumulativeQuoteQty, &price, &executedQuantity)

		order.OrderID = orderID
		order.ExecutedQuantity = functions.StrToFloat64(executedQuantity)
		order.CumulativeQuoteQuantity = math.Round(functions.StrToFloat64(cumulativeQuoteQty)*100) / 100
		order.Price = math.Round(functions.StrToFloat64(price)*1000) / 1000
		orders = append(orders, order)

	}

	defer rows.Close() /* Close rows */

	return orders, err

}

// GetProfitByThreadID retrieve total and average percentage profit by ThreadID
func GetProfitByThreadID(sessionData *types.Session) (fiat float64, percentage float64, err error) {

	var rows *sql.Rows                        /* Rows */
	var fiatNullFloat64 sql.NullFloat64       /* handle null mysql returns */
	var percentageNullFloat64 sql.NullFloat64 /* handle null mysql returns */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetProfitByThreadID(?)",
		sessionData.ThreadID); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return fiatNullFloat64.Float64, percentageNullFloat64.Float64, err

	}

	for rows.Next() {
		err = rows.Scan(&fiatNullFloat64, &percentageNullFloat64)
	}

	defer rows.Close() /* Close rows */

	return fiatNullFloat64.Float64, (percentageNullFloat64.Float64 * 100), err

}

// GetProfit retrieve total and average percentage profit
func GetProfit(
	sessionData *types.Session) (profit float64, profitNet float64, percentage float64, err error) {

	var rows *sql.Rows                        /* Rows */
	var profitNullFloat64 sql.NullFloat64     /* handle null mysql returns */
	var profitNetNullFloat64 sql.NullFloat64  /* handle null mysql returns */
	var percentageNullFloat64 sql.NullFloat64 /* handle null mysql returns */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetProfit()"); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return profitNullFloat64.Float64, profitNetNullFloat64.Float64, percentageNullFloat64.Float64, err

	}

	for rows.Next() {
		err = rows.Scan(&profit, &profitNet, &percentage)
	}

	defer rows.Close() /* Close rows */

	return profit, profitNet, (percentage * 100), err
}

// GetGlobal get global data
func GetGlobal(sessionData *types.Session) (profit float64, profitNet float64, profitPct float64, transactTime int64, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetGlobal()"); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return 0, 0, 0, 0, err
	}

	for rows.Next() {
		err = rows.Scan(&profit, &profitNet, &profitPct, &transactTime)
	}

	defer rows.Close() /* Close rows */

	return profit, profitNet, profitPct, transactTime, err

}

// GetThreadCount Retrieve Running Thread Count
func GetThreadCount(
	sessionData *types.Session) (count int, err error) {

	var rows *sql.Rows /* Rows */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetThreadCount()"); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return 0, err

	}

	for rows.Next() {
		err = rows.Scan(&count)
	}

	defer rows.Close() /* Close rows */

	return count, err

}

// GetThreadAmount Retrieve Thread Dollar Amount
func GetThreadAmount(
	sessionData *types.Session) (amount float64, err error) {

	var rows *sql.Rows                    /* Rows */
	var amountNullFloat64 sql.NullFloat64 /* handle null mysql returns */

	if flag.Lookup("test.v") != nil { /* If the -test.v flag is set, the test database is used */
		sessionData.Db.Begin() /* Start transaction */
	}

	if rows, err = sessionData.Db.Query("call cryptopump.GetThreadTransactionAmount()"); err != nil {

		logger.LogEntry{ /* Log Entry */
			Config:   nil,
			Market:   nil,
			Session:  sessionData,
			Order:    &types.Order{},
			Message:  functions.GetFunctionName() + " - " + err.Error(),
			LogLevel: "DebugLevel",
		}.Do()

		return amountNullFloat64.Float64, err

	}

	for rows.Next() {
		err = rows.Scan(&amountNullFloat64)
	}

	defer rows.Close() /* Close rows */

	return math.Round(amountNullFloat64.Float64*100) / 100, err

}
