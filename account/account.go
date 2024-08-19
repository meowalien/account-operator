package account

type account struct {
	id       string
	name     string
	currency string
}

func (a *account) Name() string {
	return a.name
}

func (a *account) Currency() string {
	return a.currency
}

func (a *account) ID() string {
	return a.id
}
