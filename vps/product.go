package vps

import (
	"context"
	"net/url"
	"regexp"
	"sort"
	"strconv"
)

// Product represents an available VPS product.
type Product struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Code        string       `json:"code"`
	Family      string       `json:"family"`
	Period      string       `json:"period"`
	Specs       ProductSpecs `json:"specs"`
}

// ProductSpecs represents the specifications of a Product.
type ProductSpecs struct {
	Cores     int `json:"cores"`
	RAM       int `json:"ram"`
	Bandwidth int `json:"bandwidth"`
}

// Products maps Product codes to Product details.
type Products map[string]Product

// ProductPeriod represents the billing period for a Product.
type ProductPeriod string

const (
	ProductPeriodMonth    ProductPeriod = "month"
	ProductPeriodQuarter  ProductPeriod = "quarter"
	ProductPeriodYear     ProductPeriod = "year"
	ProductPeriodOnDemand ProductPeriod = "on-demand"
	ProductPeriodAll      ProductPeriod = "all"
)

func (p ProductPeriod) Valid() bool {
	switch p {
	case ProductPeriodMonth,
		ProductPeriodQuarter,
		ProductPeriodYear,
		ProductPeriodOnDemand,
		ProductPeriodAll:
		return true
	default:
		return false
	}
}

// GetProducts retrieves VPS products.
// If period is empty the API default is used - currently "on-demand".
func (s *Service) GetProducts(ctx context.Context, period ProductPeriod) (Products, error) {
	if period != "" && !period.Valid() {
		return nil, &ErrInvalidProductPeriod{Period: period}
	}

	path := "/vps/products"
	if period != "" {
		path = path + "?period=" + url.QueryEscape(string(period))
	}

	var products Products
	if _, _, err := s.GetJSON(ctx, path, &products); err != nil {
		return nil, err
	}

	return products, nil
}

// ListProducts lists VPS products and sorts them by name.
// If period is empty the API default is used - currently "on-demand".
func (s *Service) ListProducts(ctx context.Context, period ProductPeriod) ([]Product, error) {
	all, err := s.GetProducts(ctx, period)
	if err != nil {
		return nil, err
	}

	products := make([]Product, 0, len(all))
	for _, product := range all {
		products = append(products, product)
	}

	var numberRegex = regexp.MustCompile(`\d+`)

	sort.Slice(products, func(i, j int) bool {
		ni := numberRegex.FindString(products[i].Name)
		nj := numberRegex.FindString(products[j].Name)

		vi, _ := strconv.Atoi(ni)
		vj, _ := strconv.Atoi(nj)

		if vi != vj {
			return vi < vj
		}

		if products[i].Name != products[j].Name {
			return products[i].Name < products[j].Name
		}

		return products[i].Code < products[j].Code
	})

	return products, nil
}
