package formatter

import (
	"time"

	"github.com/docker/cli/internal/licenseutils"
	"github.com/docker/licensing/model"
)

const (
	defaultSubscriptionsTableFormat = "table {{.Num}}\t{{.Owner}}\t{{.ProductID}}\t{{.Expires}}\t{{.ComponentsString}}"
	defaultSubscriptionsQuietFormat = "{{.Num}}:{{.Summary}}"

	numHeader               = "NUM"
	ownerHeader             = "OWNER"
	licenseNameHeader       = "NAME"
	idHeader                = "ID"
	dockerIDHeader          = "DOCKER ID"
	productIDHeader         = "PRODUCT ID"
	productRatePlanHeader   = "PRODUCT RATE PLAN"
	productRatePlanIDHeader = "PRODUCT RATE PLAN ID"
	startHeader             = "START"
	expiresHeader           = "EXPIRES"
	stateHeader             = "STATE"
	eusaHeader              = "EUSA"
	pricingComponentsHeader = "PRICING COMPONENTS"
)

// NewSubscriptionsFormat returns a Format for rendering using a license Context
func NewSubscriptionsFormat(source string, quiet bool) Format {
	switch source {
	case TableFormatKey:
		if quiet {
			return defaultSubscriptionsQuietFormat
		}
		return defaultSubscriptionsTableFormat
	case RawFormatKey:
		if quiet {
			return `license: {{.ID}}`
		}
		return `license: {{.ID}}\nname: {{.Name}}\nowner: {{.Owner}}\ncomponents: {{.ComponentsString}}\n`
	}
	return Format(source)
}

// SubscriptionsWrite writes the context
func SubscriptionsWrite(ctx Context, subs []licenseutils.LicenseDisplay) error {
	render := func(format func(subContext subContext) error) error {
		for _, sub := range subs {
			licenseCtx := &licenseContext{trunc: ctx.Trunc, l: sub}
			if err := format(licenseCtx); err != nil {
				return err
			}
		}
		return nil
	}
	licenseCtx := licenseContext{}
	licenseCtx.header = map[string]string{
		"Num":               numHeader,
		"Owner":             ownerHeader,
		"Name":              licenseNameHeader,
		"ID":                idHeader,
		"DockerID":          dockerIDHeader,
		"ProductID":         productIDHeader,
		"ProductRatePlan":   productRatePlanHeader,
		"ProductRatePlanID": productRatePlanIDHeader,
		"Start":             startHeader,
		"Expires":           expiresHeader,
		"State":             stateHeader,
		"Eusa":              eusaHeader,
		"ComponentsString":  pricingComponentsHeader,
	}
	return ctx.Write(&licenseCtx, render)
}

type licenseContext struct {
	HeaderContext
	trunc bool
	l     licenseutils.LicenseDisplay
}

func (c *licenseContext) MarshalJSON() ([]byte, error) {
	return marshalJSON(c)
}

func (c *licenseContext) Num() int {
	return c.l.Num
}

func (c *licenseContext) Owner() string {
	return c.l.Owner
}

func (c *licenseContext) ComponentsString() string {
	return c.l.ComponentsString
}

func (c *licenseContext) Summary() string {
	return c.l.String()
}

func (c *licenseContext) Name() string {
	return c.l.Name
}

func (c *licenseContext) ID() string {
	return c.l.ID
}

func (c *licenseContext) DockerID() string {
	return c.l.DockerID
}

func (c *licenseContext) ProductID() string {
	return c.l.ProductID
}

func (c *licenseContext) ProductRatePlan() string {
	return c.l.ProductRatePlan
}

func (c *licenseContext) ProductRatePlanID() string {
	return c.l.ProductRatePlanID
}

func (c *licenseContext) Start() *time.Time {
	return c.l.Start
}

func (c *licenseContext) Expires() *time.Time {
	return c.l.Expires
}

func (c *licenseContext) State() string {
	return c.l.State
}

func (c *licenseContext) Eusa() *model.EusaState {
	return c.l.Eusa
}

func (c *licenseContext) PricingComponents() []model.SubscriptionPricingComponent {
	// Dereference the pricing component pointers in the pricing components
	// so it can be rendered properly with the template formatter

	var ret []model.SubscriptionPricingComponent
	for _, spc := range c.l.PricingComponents {
		if spc == nil {
			continue
		}
		ret = append(ret, *spc)
	}
	return ret
}
