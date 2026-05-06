package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
)

// e164Regex matches valid E.164 format: + followed by 1-15 digits
var e164Regex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

// ValidateE164 checks if a phone number is in valid E.164 international format.
// E.164 format: +[country code][subscriber number] — max 15 digits after +
// Examples: +919876543210, +12125551234, +447911123456
func ValidateE164(phone string) error {
	if !e164Regex.MatchString(phone) {
		return fmt.Errorf("invalid phone number format. Must be in E.164 format (e.g. +919876543210)")
	}
	return nil
}

// NormalizeToE164 attempts to convert a raw phone number to E.164 format.
// Handles: "9876543210" (India) -> "+919876543210"
// If it already starts with +, it just validates.
func NormalizeToE164(raw string, defaultCountryCode string) (string, error) {
	// Already has + prefix
	if len(raw) > 0 && raw[0] == '+' {
		if err := ValidateE164(raw); err != nil {
			return "", err
		}
		return raw, nil
	}

	// Strip all non-digit characters
	cleaned := regexp.MustCompile(`\D`).ReplaceAllString(raw, "")

	// If it starts with 0, remove leading zero
	if len(cleaned) > 1 && cleaned[0] == '0' {
		cleaned = cleaned[1:]
	}

	// Prepend default country code
	if defaultCountryCode != "" {
		cleaned = "+" + defaultCountryCode + cleaned
	} else {
		cleaned = "+" + cleaned
	}

	if err := ValidateE164(cleaned); err != nil {
		return "", err
	}
	return cleaned, nil
}

// PhoneLineType represents the type of phone line detected by Lookup API.
type PhoneLineType string

const (
	LineTypeMobile   PhoneLineType = "mobile"
	LineTypeLandline PhoneLineType = "landline"
	LineTypeVoIP     PhoneLineType = "voip"
	LineTypeUnknown  PhoneLineType = "unknown"
)

// CarrierInfo holds information about the phone number's carrier.
type CarrierInfo struct {
	Name              string
	Type              PhoneLineType // mobile, landline, voip
	MobileCountryCode string
	MobileNetworkCode string
}

// PhoneLookupResult is the result from the Twilio Lookup API.
type PhoneLookupResult struct {
	PhoneNumber string // E.164 formatted number
	CountryCode string // e.g. "IN", "US"
	Carrier     CarrierInfo
	IsValid     bool // Whether the number is valid and reachable
	IsVoIP      bool // Whether the number is a VoIP/virtual number
}

// PhoneLookupProvider checks phone numbers via the Twilio Lookup API.
type PhoneLookupProvider struct {
	accountSid string
	authToken  string
	blockVoip  bool
}

// NewPhoneLookupProvider creates a new phone lookup provider.
// In production, panics if Twilio credentials are missing — no dev bypass allowed.
func NewPhoneLookupProvider(sid, token string, blockVoip bool) *PhoneLookupProvider {
	if sid == "" || sid == "your_twilio_sid" {
		// NEVER allow dev bypass in production
		env := os.Getenv("ENVIRONMENT")
		if env == "production" || env == "prod" {
			panic("CRITICAL: Twilio Lookup credentials not configured in production. Set TWILIO_ACCOUNT_SID in app.env")
		}
		return nil // nil means dev mode — skip lookup (development only)
	}
	return &PhoneLookupProvider{
		accountSid: sid,
		authToken:  token,
		blockVoip:  blockVoip,
	}
}

// Lookup performs a carrier and caller-name lookup on a phone number.
// Returns detailed carrier info including whether it's VoIP.
func (p *PhoneLookupProvider) Lookup(phoneNumber string) (*PhoneLookupResult, error) {
	if p == nil {
		// Dev mode: accept everything
		return &PhoneLookupResult{
			PhoneNumber: phoneNumber,
			Carrier: CarrierInfo{
				Name: "Dev Mode",
				Type: LineTypeMobile,
			},
			IsValid: true,
			IsVoIP:  false,
		}, nil
	}

	apiURL := fmt.Sprintf("https://lookups.twilio.com/v2/PhoneNumbers/%s", url.PathEscape(phoneNumber))

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create lookup request: %w", err)
	}

	q := req.URL.Query()
	q.Set("Fields", "line_type_intelligence,carrier")
	req.URL.RawQuery = q.Encode()

	req.SetBasicAuth(p.accountSid, p.authToken)
	req.Header.Add("Accept", "application/json")

	client := &http.Client{Timeout: 10000000000} // 10s timeout
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lookup request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errorBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errorBody)
		return nil, fmt.Errorf("lookup API error: HTTP %d — %v", resp.StatusCode, errorBody)
	}

	var result struct {
		PhoneNumber string `json:"phone_number"`
		CountryCode string `json:"country_code"`
		Carrier     struct {
			Name              string `json:"carrier_name"`
			Type              string `json:"type"`
			MobileCountryCode string `json:"mobile_country_code"`
			MobileNetworkCode string `json:"mobile_network_code"`
		} `json:"carrier"`
		LineTypeIntelligence struct {
			LineType string `json:"line_type"`
		} `json:"line_type_intelligence"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode lookup response: %w", err)
	}

	lineType := parseLineType(result.LineTypeIntelligence.LineType)
	isVoip := lineType == LineTypeVoIP

	return &PhoneLookupResult{
		PhoneNumber: result.PhoneNumber,
		CountryCode: result.CountryCode,
		Carrier: CarrierInfo{
			Name:              result.Carrier.Name,
			Type:              lineType,
			MobileCountryCode: result.Carrier.MobileCountryCode,
			MobileNetworkCode: result.Carrier.MobileNetworkCode,
		},
		IsValid: lineType != LineTypeUnknown,
		IsVoIP:  isVoip,
	}, nil
}

// ValidateAndCheck combines E.164 validation with Lookup API check.
// Returns an error if the number is invalid or blocked (e.g., VoIP).
func (p *PhoneLookupProvider) ValidateAndCheck(phoneNumber string) (*PhoneLookupResult, error) {
	// Step 1: Validate E.164 format
	if err := ValidateE164(phoneNumber); err != nil {
		return nil, err
	}

	// Step 2: Lookup via Twilio API
	result, err := p.Lookup(phoneNumber)
	if err != nil {
		return nil, err
	}

	// Step 3: Block VoIP numbers if configured
	if p != nil && p.blockVoip && result.IsVoIP {
		return nil, fmt.Errorf("virtual/VoIP phone numbers are not allowed. Please use a real mobile number.")
	}

	return result, nil
}

func parseLineType(raw string) PhoneLineType {
	switch raw {
	case "mobile":
		return LineTypeMobile
	case "landline":
		return LineTypeLandline
	case "voip":
		return LineTypeVoIP
	default:
		return LineTypeUnknown
	}
}
