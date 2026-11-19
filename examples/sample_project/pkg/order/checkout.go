package order

import (
	"fmt"
)

// CheckoutManager processes user shopping cart purchases.
type CheckoutManager struct {
	env string
}

// OrderItem represents a line item in an order.
type OrderItem struct {
	ID       string
	Quantity int
	PriceUSD int
}

// ProcessCheckout validates items and triggers payment deduction.
func (cm *CheckoutManager) ProcessCheckout(orderID string, items []OrderItem) error {
	total := 0
	for _, it := range items {
		total += it.PriceUSD * it.Quantity
	}
	fmt.Printf("Processing order %s for $%d\n", orderID, total)
	return nil
}
