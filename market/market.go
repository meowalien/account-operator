package market

import (
	"errors"
	"github.com/sirupsen/logrus"
	"go.uber.org/atomic"
	"sync"
)

type Market interface {
	UpdatePrice(symbol string, price string)
	MarketOrder(symbol string, callBAck func(string)) error
}

func NewMarket() Market {
	return &market{
		tradePairs: make(map[string]*price),
	}
}

type price struct {
	currentPrice atomic.String
}

func (p *price) UpdatePrice(currentPrice string) {
	p.currentPrice.Store(currentPrice)
}

func (p *price) CurrentPrice() string {
	return p.currentPrice.Load()
}

func newPrice() *price {
	return &price{}
}

type market struct {
	// symbol -> price
	tradePairs     map[string]*price
	tradePairsLock sync.RWMutex
}

var ErrSymbolNotFound = errors.New("symbol not found")

func (m *market) MarketOrder(symbol string, callBAck func(string)) error {
	m.tradePairsLock.RLock()
	priceInst, exists := m.tradePairs[symbol]
	m.tradePairsLock.RUnlock()
	if !exists {
		return ErrSymbolNotFound
	}

	callBAck(priceInst.CurrentPrice())
	return nil
}

func (m *market) UpdatePrice(symbol string, currentPrice string) {
	logrus.Infof("Update price for symbol %s: %s", symbol, currentPrice)
	priceInst := m.getOrCreatePrice(symbol)
	priceInst.UpdatePrice(currentPrice)
}

func (m *market) getOrCreatePrice(symbol string) (priceInst *price) {
	m.tradePairsLock.RLock()
	var exists bool
	priceInst, exists = m.tradePairs[symbol]
	m.tradePairsLock.RUnlock()
	if exists {
		return
	}

	m.tradePairsLock.Lock()
	defer m.tradePairsLock.Unlock()
	priceInst, exists = m.tradePairs[symbol]
	if exists {
		// Another goroutine has created the price instance
		return
	}
	priceInst = newPrice()
	m.tradePairs[symbol] = priceInst
	return
}
