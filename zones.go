package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-querystring/query"
)

type ZonesService service

// Owner describes the resource owner.
type Owner struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	OwnerType string `json:"type"`
}

// Zone describes a Cloudflare zone.
type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// DevMode contains the time in seconds until development expires (if
	// positive) or since it expired (if negative). It will be 0 if never used.
	DevMode           int       `json:"development_mode"`
	OriginalNS        []string  `json:"original_name_servers"`
	OriginalRegistrar string    `json:"original_registrar"`
	OriginalDNSHost   string    `json:"original_dnshost"`
	CreatedOn         time.Time `json:"created_on"`
	ModifiedOn        time.Time `json:"modified_on"`
	NameServers       []string  `json:"name_servers"`
	Owner             Owner     `json:"owner"`
	Permissions       []string  `json:"permissions"`
	Plan              ZonePlan  `json:"plan"`
	PlanPending       ZonePlan  `json:"plan_pending,omitempty"`
	Status            string    `json:"status"`
	Paused            bool      `json:"paused"`
	Type              string    `json:"type"`
	Host              struct {
		Name    string
		Website string
	} `json:"host"`
	VanityNS        []string `json:"vanity_name_servers"`
	Betas           []string `json:"betas"`
	DeactReason     string   `json:"deactivation_reason"`
	Meta            ZoneMeta `json:"meta"`
	Account         Account  `json:"account"`
	VerificationKey string   `json:"verification_key"`
}

// ZoneMeta describes metadata about a zone.
type ZoneMeta struct {
	// custom_certificate_quota is broken - sometimes it's a string, sometimes a number!
	// CustCertQuota     int    `json:"custom_certificate_quota"`
	PageRuleQuota     int  `json:"page_rule_quota"`
	WildcardProxiable bool `json:"wildcard_proxiable"`
	PhishingDetected  bool `json:"phishing_detected"`
}

// ZonePlan contains the plan information for a zone.
type ZonePlan struct {
	ZonePlanCommon
	LegacyID          string `json:"legacy_id"`
	IsSubscribed      bool   `json:"is_subscribed"`
	CanSubscribe      bool   `json:"can_subscribe"`
	LegacyDiscount    bool   `json:"legacy_discount"`
	ExternallyManaged bool   `json:"externally_managed"`
}

// ZoneRatePlan contains the plan information for a zone.
type ZoneRatePlan struct {
	ZonePlanCommon
	Components []zoneRatePlanComponents `json:"components,omitempty"`
}

// ZonePlanCommon contains fields used by various Plan endpoints.
type ZonePlanCommon struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Price     int    `json:"price,omitempty"`
	Currency  string `json:"currency,omitempty"`
	Frequency string `json:"frequency,omitempty"`
}

type zoneRatePlanComponents struct {
	Name      string `json:"name"`
	Default   int    `json:"Default"`
	UnitPrice int    `json:"unit_price"`
}

// ZoneID contains only the zone ID.
type ZoneID struct {
	ID string `json:"id"`
}

// ZoneResponse represents the response from the Zone endpoint containing a single zone.
type ZoneResponse struct {
	Response
	Result Zone `json:"result"`
}

// ZonesResponse represents the response from the Zone endpoint containing multiple zones.
type ZonesResponse struct {
	Response
	Result []Zone `json:"result"`
}

type ZoneParams struct {
	Match       string `url:"match,omitempty"`
	Name        string `url:"name,omitempty"`
	AccountName string `url:"account.name,omitempty"`
	Status      string `url:"status,omitempty"`
	AccountID   string `url:"account.id,omitempty"`
	Direction   string `url:"direction,omitempty"`

	// ResultInfo
}

type Account struct {
	ID       string           `json:"id,omitempty"`
	Name     string           `json:"name,omitempty"`
	Type     string           `json:"type,omitempty"`
	Settings *AccountSettings `json:"settings,omitempty"`
}

// AccountSettings outlines the available options for an account.
type AccountSettings struct {
	EnforceTwoFactor bool `json:"enforce_twofactor"`
}

// Get fetches a single zone.
//
// API reference: https://api.cloudflare.com/#zone-zone-details
func (s *ZonesService) Get(ctx context.Context, zoneID string) (Zone, error) {
	if !isValidZoneIdentifier(zoneID) {
		return Zone{}, fmt.Errorf(errInvalidZoneIdentifer, zoneID)
	}

	res, _ := s.client.Call(context.Background(), http.MethodGet, "/zones/"+zoneID, nil)

	var r ZoneResponse
	err := json.Unmarshal(res, &r)
	if err != nil {
		return Zone{}, fmt.Errorf("failed to unmarshal zone JSON data: %w", err)
	}

	return r.Result, nil
}

// List returns all zones that match the provided `ZoneParams` struct.
//
// API reference: https://api.cloudflare.com/#zone-list-zones
func (s *ZonesService) List(ctx context.Context, params ZoneParams) ([]Zone, error) {
	v, _ := query.Values(params)
	queryParams := v.Encode()
	if queryParams != "" {
		queryParams = "?" + queryParams
	}

	res, _ := s.client.Call(context.Background(), http.MethodGet, "/zones"+queryParams, nil)

	var r ZonesResponse
	err := json.Unmarshal(res, &r)
	if err != nil {
		return []Zone{}, fmt.Errorf("failed to unmarshal zone JSON data: %w", err)
	}

	return r.Result, nil
}

// Delete deletes a zone based on ID.
//
// API reference: https://api.cloudflare.com/#zone-delete-zone
func (s *ZonesService) Delete(ctx context.Context, zoneID string) error {
	if !isValidZoneIdentifier(zoneID) {
		return fmt.Errorf(errInvalidZoneIdentifer, zoneID)
	}

	res, _ := s.client.Call(context.Background(), http.MethodDelete, "/zones/"+zoneID, nil)

	var r ZoneResponse
	err := json.Unmarshal(res, &r)
	if err != nil {
		return fmt.Errorf("failed to unmarshal zone JSON data: %w", err)
	}

	return nil
}
