package postgresql

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var client *sql.DB

func ConnectDB() error {
	host := viper.GetString("db.postgresql.host")
	port := viper.GetInt("db.postgresql.port")
	user := viper.GetString("db.postgresql.user")
	password := viper.GetString("db.postgresql.password")
	dbname := viper.GetString("db.postgresql.dbname")

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	// Open the connection
	client, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		return err
	}

	// Test the connection
	err = client.Ping()
	if err != nil {
		return err
	}
	logrus.Infof("Successfully connected to the database!")
	return nil
}

func DisconnectDB() {
	if err := client.Close(); err != nil {
		logrus.Errorf("Failed to disconnect from DB: %v", err)
	}
	logrus.Println("Disconnected from DB!")
}

func GetClient() *sql.DB {
	return client
}
