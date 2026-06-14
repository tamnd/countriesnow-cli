package countriesnow

import (
	"context"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

func init() { kit.Register(Domain{}) }

// Domain is the countriesnow driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against, and
// the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "countriesnow",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "countriesnow",
			Short:  "Look up countries, capitals, currencies, flags, population, and cities.",
			Long: `countriesnow queries the free countriesnow.space API for country data:
capitals, currencies, flag images, population figures, and city lists.

No API key is required. All commands emit clean JSON records.`,
			Site: Host,
			Repo: "https://github.com/tamnd/countriesnow-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{
		Name:    "capitals",
		Group:   "read",
		List:    true,
		Summary: "List countries with their capitals",
		URIType: "country",
	}, listCapitals)

	kit.Handle(app, kit.OpMeta{
		Name:    "currencies",
		Group:   "read",
		List:    true,
		Summary: "List countries with their currencies",
		URIType: "country",
	}, listCurrencies)

	kit.Handle(app, kit.OpMeta{
		Name:    "flags",
		Group:   "read",
		List:    true,
		Summary: "List countries with their flag image URLs",
		URIType: "country",
	}, listFlags)

	kit.Handle(app, kit.OpMeta{
		Name:    "population",
		Group:   "read",
		List:    true,
		Summary: "List countries with their latest population",
		URIType: "country",
	}, listPopulation)

	kit.Handle(app, kit.OpMeta{
		Name:    "cities",
		Group:   "read",
		List:    true,
		Summary: "List cities in a country",
		URIType: "city",
		Args:    []kit.Arg{{Name: "country", Help: "country name (e.g. Japan)"}},
	}, listCities)
}

// newClient builds the client from the host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- inputs ---

type limitInput struct {
	Limit  int     `kit:"flag,inherit" help:"max results (0 = all)"`
	Client *Client `kit:"inject"`
}

type citiesInput struct {
	Country string  `kit:"arg" help:"country name"`
	Client  *Client `kit:"inject"`
}

// CityRecord wraps a city string so it emits as a JSON record.
type CityRecord struct {
	Name string `json:"name"`
}

// --- handlers ---

func listCapitals(ctx context.Context, in limitInput, emit func(*CountryInfo) error) error {
	items, err := in.Client.Capitals(ctx, in.Limit)
	if err != nil {
		return mapErr(err)
	}
	for i := range items {
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

func listCurrencies(ctx context.Context, in limitInput, emit func(*CountryInfo) error) error {
	items, err := in.Client.Currencies(ctx, in.Limit)
	if err != nil {
		return mapErr(err)
	}
	for i := range items {
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

func listFlags(ctx context.Context, in limitInput, emit func(*CountryInfo) error) error {
	items, err := in.Client.Flags(ctx, in.Limit)
	if err != nil {
		return mapErr(err)
	}
	for i := range items {
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

func listPopulation(ctx context.Context, in limitInput, emit func(*CountryPop) error) error {
	items, err := in.Client.Population(ctx, in.Limit)
	if err != nil {
		return mapErr(err)
	}
	for i := range items {
		if err := emit(&items[i]); err != nil {
			return err
		}
	}
	return nil
}

func listCities(ctx context.Context, in citiesInput, emit func(*CityRecord) error) error {
	cities, err := in.Client.Cities(ctx, in.Country)
	if err != nil {
		return mapErr(err)
	}
	for _, city := range cities {
		if err := emit(&CityRecord{Name: city}); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver ---

// Classify turns a country name into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("unrecognized countriesnow reference: %q", input)
	}
	return "country", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	if uriType != "country" && uriType != "city" {
		return "", errs.Usage("countriesnow has no resource type %q", uriType)
	}
	return "https://" + Host + "/api/v0.1/countries/capital", nil
}

func mapErr(err error) error {
	return err
}
