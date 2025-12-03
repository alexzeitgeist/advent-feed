package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"
)

// Store configuration
type StoreConfig struct {
	Name     string
	APIURL   string
	BaseURL  string
	PortalID string
}

var stores = map[string]StoreConfig{
	"galaxus": {
		Name:     "Galaxus",
		APIURL:   "https://www.galaxus.ch/api/graphql/get-adventcalendar",
		BaseURL:  "https://www.galaxus.ch",
		PortalID: "22",
	},
	"digitec": {
		Name:     "Digitec",
		APIURL:   "https://www.digitec.ch/api/graphql/get-adventcalendar",
		BaseURL:  "https://www.digitec.ch",
		PortalID: "25",
	},
}

// GraphQL request/response structures
type GraphQLRequest struct {
	OperationName string         `json:"operationName"`
	Variables     map[string]any `json:"variables"`
	Query         string         `json:"query"`
}

type APIResponse struct {
	Data struct {
		AdventCalendar AdventCalendar `json:"adventCalendar"`
	} `json:"data"`
}

type AdventCalendar struct {
	CurrentDate string    `json:"currentDate"`
	Header      Header    `json:"header"`
	Products    []Product `json:"products"`
}

type Header struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl"`
}

type Product struct {
	Product struct {
		ID              string  `json:"id"`
		ProductID       int     `json:"productId"`
		Name            string  `json:"name"`
		NameProperties  string  `json:"nameProperties"`
		ProductTypeName string  `json:"productTypeName"`
		BrandName       string  `json:"brandName"`
		AverageRating   float64 `json:"averageRating"`
		TotalRatings    int     `json:"totalRatings"`
		Images          []struct {
			URL string `json:"url"`
		} `json:"images"`
	} `json:"product"`
	Offer struct {
		Price struct {
			AmountInclusive float64 `json:"amountInclusive"`
			Currency        string  `json:"currency"`
		} `json:"price"`
		SalesInformation struct {
			NumberOfItems     int    `json:"numberOfItems"`
			NumberOfItemsSold int    `json:"numberOfItemsSold"`
			ValidFrom         string `json:"validFrom"`
		} `json:"salesInformation"`
		InsteadOfPrice *struct {
			Price struct {
				AmountInclusive float64 `json:"amountInclusive"`
			} `json:"price"`
		} `json:"insteadOfPrice"`
	} `json:"offer"`
}

// Atom feed structures
type AtomFeed struct {
	XMLName  xml.Name    `xml:"feed"`
	XMLNS    string      `xml:"xmlns,attr"`
	Title    string      `xml:"title"`
	Subtitle string      `xml:"subtitle,omitempty"`
	Link     AtomLink    `xml:"link"`
	Icon     string      `xml:"icon,omitempty"`
	Updated  string      `xml:"updated"`
	ID       string      `xml:"id"`
	Author   *AtomAuthor `xml:"author,omitempty"`
	Entries  []AtomEntry `xml:"entry"`
}

type AtomAuthor struct {
	Name string `xml:"name"`
	URI  string `xml:"uri,omitempty"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
}

type AtomEntry struct {
	Title   string      `xml:"title"`
	Link    AtomLink    `xml:"link"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Summary AtomContent `xml:"summary"`
	Content AtomContent `xml:"content"`
}

type AtomContent struct {
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// Cache structure
type Cache struct {
	mu        sync.RWMutex
	data      *AtomFeed
	fetchedAt time.Time
	duration  time.Duration
}

func (c *Cache) Get() *AtomFeed {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.data != nil && time.Since(c.fetchedAt) < c.duration {
		return c.data
	}
	return nil
}

func (c *Cache) Set(feed *AtomFeed) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = feed
	c.fetchedAt = time.Now()
}

// Global configuration
type Config struct {
	Store         StoreConfig
	UserAgent     string
	Port          int
	CacheDuration time.Duration
}

var (
	config Config
	cache  *Cache
)

const graphQLQuery = `query GET_ADVENTCALENDAR($date: String) {
  adventCalendar(date: $date) {
    currentDate
    header {
      title
      description
      imageUrl
      __typename
    }
    products {
      ...ProductWithOffer
      __typename
    }
    __typename
  }
}

fragment ProductWithOffer on ProductWithOffer {
  mandatorSpecificData {
    ...ProductMandatorSpecific
    __typename
  }
  product {
    ...ProductMandatorIndependent
    __typename
  }
  offer {
    ...ProductOffer
    __typename
  }
  isDefaultOffer
  __typename
}

fragment ProductMandatorSpecific on MandatorSpecificData {
  isBestseller
  isDeleted
  sectorIds
  hasVariants
  showrooms {
    siteId
    name
    __typename
  }
  __typename
}

fragment ProductMandatorIndependent on ProductV2 {
  id
  productId
  name
  nameProperties
  productTypeId
  productTypeName
  brandId
  brandName
  averageRating
  totalRatings
  totalQuestions
  images {
    url
    height
    width
    __typename
  }
  energyEfficiency {
    energyEfficiencyColorType
    energyEfficiencyLabelText
    energyEfficiencyLabelSigns
    energyEfficiencyImage {
      url
      height
      width
      __typename
    }
    isNewEnergyEfficiencyLabel
    __typename
  }
  seo {
    seoProductTypeName
    seoNameProperties
    productGroups {
      productGroup1
      productGroup2
      productGroup3
      productGroup4
      __typename
    }
    gtin
    __typename
  }
  basePrice {
    priceFactor
    value
    __typename
  }
  productDataSheet {
    name
    languages
    url
    size
    __typename
  }
  __typename
}

fragment ProductOffer on OfferV2 {
  id
  productId
  offerId
  shopOfferId
  price {
    amountInclusive
    amountExclusive
    currency
    __typename
  }
  deliveryOptions {
    mail {
      classification
      futureReleaseDate
      launchesAt
      __typename
    }
    pickup {
      siteId
      classification
      futureReleaseDate
      launchesAt
      __typename
    }
    detailsProvider {
      productId
      offerId
      refurbishedId
      resaleId
      __typename
    }
    __typename
  }
  label
  labelType
  type
  volumeDiscountPrices {
    minAmount
    price {
      amountInclusive
      amountExclusive
      currency
      __typename
    }
    isDefault
    __typename
  }
  salesInformation {
    numberOfItems
    numberOfItemsSold
    isEndingSoon
    validFrom
    __typename
  }
  incentiveText
  isIncentiveCashback
  isNew
  isSalesPromotion
  hideInProductDiscovery
  canAddToBasket
  hidePrice
  insteadOfPrice {
    type
    price {
      amountInclusive
      amountExclusive
      currency
      __typename
    }
    __typename
  }
  minOrderQuantity
  __typename
}`

func fetchAdventCalendar() (*AdventCalendar, error) {
	reqBody := []GraphQLRequest{{
		OperationName: "GET_ADVENTCALENDAR",
		Variables:     map[string]any{},
		Query:         graphQLQuery,
	}}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", config.Store.APIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers matching the browser request
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", config.Store.BaseURL)
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("User-Agent", config.UserAgent)

	// Store-specific headers
	req.Header.Set("x-dg-graphql-client-name", "isomorph")
	req.Header.Set("x-dg-language", "de-CH")
	req.Header.Set("x-dg-portal", config.Store.PortalID)
	req.Header.Set("x-dg-routename", "/advent-calendar")
	req.Header.Set("x-dg-routeowner", "stellapolaris")
	req.Header.Set("x-dg-team", "stellapolaris")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp []APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	return &apiResp[0].Data.AdventCalendar, nil
}

func calculateDiscount(current, original float64) int {
	if original <= 0 || current >= original {
		return 0
	}
	return int(((original - current) / original) * 100)
}

func parseValidFrom(validFrom string) time.Time {
	t, err := time.Parse(time.RFC3339, validFrom)
	if err != nil {
		return time.Time{}
	}
	return t
}

func buildAtomFeed(calendar *AdventCalendar) *AtomFeed {
	// Sort products by validFrom date descending
	products := make([]Product, len(calendar.Products))
	copy(products, calendar.Products)
	sort.Slice(products, func(i, j int) bool {
		ti := parseValidFrom(products[i].Offer.SalesInformation.ValidFrom)
		tj := parseValidFrom(products[j].Offer.SalesInformation.ValidFrom)
		return ti.After(tj)
	})

	entries := make([]AtomEntry, 0, len(products))
	for _, p := range products {
		productURL := fmt.Sprintf("%s/product/%d", config.Store.BaseURL, p.Product.ProductID)
		stableID := fmt.Sprintf("urn:advent:%d", p.Product.ProductID)

		// Calculate discount
		discount := 0
		if p.Offer.InsteadOfPrice != nil {
			discount = calculateDiscount(p.Offer.Price.AmountInclusive, p.Offer.InsteadOfPrice.Price.AmountInclusive)
		}

		// Build title with discount
		title := fmt.Sprintf("%s: %s", p.Product.BrandName, p.Product.Name)
		if p.Product.NameProperties != "" {
			title += " - " + p.Product.NameProperties
		}
		if discount > 0 {
			title = fmt.Sprintf("[%d%% off] %s", discount, title)
		}

		// Parse date for updated field
		validFrom := parseValidFrom(p.Offer.SalesInformation.ValidFrom)
		updatedStr := validFrom.Format(time.RFC3339)

		// Build image tag
		imageHTML := ""
		if len(p.Product.Images) > 0 {
			imageHTML = fmt.Sprintf(`<img src="%s" alt="%s" style="max-width:400px;"/><br/><br/>`, p.Product.Images[0].URL, p.Product.Name)
		}

		// Stock info
		remaining := p.Offer.SalesInformation.NumberOfItems - p.Offer.SalesInformation.NumberOfItemsSold
		stockInfo := fmt.Sprintf("%d/%d remaining", remaining, p.Offer.SalesInformation.NumberOfItems)

		// Price info
		priceInfo := fmt.Sprintf("%.2f %s", p.Offer.Price.AmountInclusive, p.Offer.Price.Currency)
		if p.Offer.InsteadOfPrice != nil {
			priceInfo = fmt.Sprintf("<strong>%.2f %s</strong> <s>%.2f %s</s>",
				p.Offer.Price.AmountInclusive, p.Offer.Price.Currency,
				p.Offer.InsteadOfPrice.Price.AmountInclusive, p.Offer.Price.Currency)
		}

		// Rating info
		ratingInfo := ""
		if p.Product.TotalRatings > 0 {
			ratingInfo = fmt.Sprintf("%.1f/5 (%d reviews)", p.Product.AverageRating, p.Product.TotalRatings)
		}

		// Build content HTML
		content := fmt.Sprintf(`%s<p><strong>Brand:</strong> %s</p>
<p><strong>Type:</strong> %s</p>
<p><strong>Price:</strong> %s</p>
<p><strong>Stock:</strong> %s</p>`,
			imageHTML,
			p.Product.BrandName,
			p.Product.ProductTypeName,
			priceInfo,
			stockInfo)

		if ratingInfo != "" {
			content += fmt.Sprintf("\n<p><strong>Rating:</strong> %s</p>", ratingInfo)
		}

		content += fmt.Sprintf(`<p><a href="%s">View on %s</a></p>`, productURL, config.Store.Name)

		entries = append(entries, AtomEntry{
			Title: title,
			Link: AtomLink{
				Href: productURL,
				Rel:  "alternate",
			},
			ID:      stableID,
			Updated: updatedStr,
			Summary: AtomContent{
				Type:    "text",
				Content: fmt.Sprintf("%s - %s - %s", p.Product.BrandName, p.Product.ProductTypeName, priceInfo),
			},
			Content: AtomContent{
				Type:    "html",
				Content: content,
			},
		})
	}

	// Fix protocol-relative URL for icon
	iconURL := calendar.Header.ImageURL
	if len(iconURL) > 2 && iconURL[:2] == "//" {
		iconURL = "https:" + iconURL
	}

	return &AtomFeed{
		XMLNS:    "http://www.w3.org/2005/Atom",
		Title:    fmt.Sprintf("%s - %s", calendar.Header.Title, config.Store.Name),
		Subtitle: calendar.Header.Description,
		Link:     AtomLink{Href: config.Store.BaseURL + "/advent-calendar", Rel: "alternate"},
		Icon:     iconURL,
		Updated:  time.Now().Format(time.RFC3339),
		ID:       "urn:advent-calendar:" + config.Store.Name,
		Author: &AtomAuthor{
			Name: config.Store.Name,
			URI:  config.Store.BaseURL,
		},
		Entries: entries,
	}
}

func getFeed() (*AtomFeed, error) {
	if cached := cache.Get(); cached != nil {
		log.Println("Serving from cache")
		return cached, nil
	}

	log.Println("Fetching fresh data from API")
	calendar, err := fetchAdventCalendar()
	if err != nil {
		return nil, err
	}

	feed := buildAtomFeed(calendar)
	cache.Set(feed)
	return feed, nil
}

func feedHandler(w http.ResponseWriter, r *http.Request) {
	feed, err := getFeed()
	if err != nil {
		log.Printf("Error fetching feed: %v", err)
		http.Error(w, "Failed to fetch feed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(feed); err != nil {
		log.Printf("Error encoding feed: %v", err)
	}
}

func main() {
	storeName := flag.String("store", "galaxus", "Store to fetch from: galaxus or digitec")
	userAgent := flag.String("ua", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36", "User-Agent header")
	port := flag.Int("port", 8080, "Port to listen on")
	cacheDuration := flag.Duration("cache", 5*time.Minute, "Cache duration (e.g., 5m, 1h)")
	flag.Parse()

	store, ok := stores[*storeName]
	if !ok {
		log.Fatalf("Unknown store: %s (valid options: galaxus, digitec)", *storeName)
	}

	config = Config{
		Store:         store,
		UserAgent:     *userAgent,
		Port:          *port,
		CacheDuration: *cacheDuration,
	}

	cache = &Cache{duration: config.CacheDuration}

	http.HandleFunc("/feed", feedHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, "%s Advent Calendar Feed\n\nGet the feed at /feed\n", config.Store.Name)
	})

	addr := fmt.Sprintf(":%d", config.Port)
	log.Printf("Starting %s Advent Calendar server on %s", config.Store.Name, addr)
	log.Printf("Feed available at http://localhost%s/feed", addr)
	log.Printf("Cache duration: %v", config.CacheDuration)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
