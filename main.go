package main

import (
	"account-operator/account"
	"account-operator/config"
	"account-operator/http"
	"account-operator/log"
	"account-operator/market"
	"account-operator/postgresql"
	"account-operator/price"
	"account-operator/quit"
	"account-operator/rabbitmq"
	"account-operator/token"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

const InitializationTimeout = 30 * time.Second

const FinalizeTimeout = time.Second * 10

func main() {
	defer logrus.Info("Main exiting")
	defer quit.WaitForAllGoroutineEnd(FinalizeTimeout)
	err := config.InitConfig()
	if err != nil {
		logrus.Panicf("Failed to initialize config: %v", err)
		return
	}
	log.InitLogger()
	token.InitVerifyKey(viper.GetString("token.publicKeyPath"))

	ctx, cancel := context.WithTimeoutCause(context.Background(), InitializationTimeout, fmt.Errorf("initilization timeout"))
	defer cancel()

	err = postgresql.ConnectDB()
	if err != nil {
		logrus.Panicf("Failed to connect to DB: %v", err)
		return
	}
	defer postgresql.DisconnectDB()
	err = rabbitmq.InitRabbitMQ(ctx)
	if err != nil {
		logrus.Panicf("Failed to initialize RabbitMQ: %v", err)
		return
	}
	defer rabbitmq.CloseRabbitMQ()

	receiverInst := price.NewPriceReceiver()
	msgs, err := receiverInst.Start(ctx)
	if err != nil {
		logrus.Panicf("Failed to start price receiver: %v", err)
		return
	}
	defer receiverInst.Close()

	marketInst := market.NewMarket()

	operatorInst := account.NewOperator(msgs, marketInst)
	operatorInst.Start()
	defer operatorInst.Close()

	r, err := http.SetupRouter(operatorInst)
	if err != nil {
		logrus.Panicf("Failed to setup router: %v", err)
		return
	}
	srv := http.StartServer(r)
	defer http.ShutdownServer(srv)

	quit.WaitForQuitSignal()

}
