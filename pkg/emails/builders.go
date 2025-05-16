package emails

import (
	"net/url"

	"github.com/trisacrypto/envoy/pkg/verification"
)

//===========================================================================
// Sunrise Invite
//===========================================================================

const (
	SunriseInviteRE       = "Travel rule compliance exchange requested"
	SunriseInviteTemplate = "sunrise_invite"
)

func NewSunriseInvite(recipient string, data SunriseInviteData) (*Email, error) {
	return New(recipient, SunriseInviteRE, SunriseInviteTemplate, data)
}

// SunriseInviteData is used to complete the sunrise_invite template.
type SunriseInviteData struct {
	ContactName     string
	ComplianceName  string
	OriginatorName  string
	BeneficiaryName string
	BaseURL         *url.URL
	Token           verification.VerificationToken
	SupportEmail    string
	ComplianceEmail string
}

func (s SunriseInviteData) VerifyURL() string {
	if s.BaseURL == nil {
		return ""
	}

	params := make(url.Values, 1)
	params.Set("token", s.Token.String())

	s.BaseURL.RawQuery = params.Encode()
	return s.BaseURL.String()
}

//===========================================================================
// Verify Email
//===========================================================================

const (
	VerifyEmailRE       = "One-time verification code"
	VerifyEmailTemplate = "verify_email"
)

func NewVerifyEmail(recipient string, data VerifyEmailData) (*Email, error) {
	return New(recipient, VerifyEmailRE, VerifyEmailTemplate, data)
}

// VerifyEmailData is used to send a one-time code to the original email for verification.
type VerifyEmailData struct {
	Code           string
	SupportEmail   string
	ComplianceName string
}

// ===========================================================================
// Reset Password Email
// ===========================================================================

const (
	ResetPasswordRE       = "TRISA Envoy password reset request"
	ResetPasswordTemplate = "reset_password"
)

func NewResetPasswordEmail(recipient string, data ResetPasswordEmailData) (*Email, error) {
	return New(recipient, ResetPasswordRE, ResetPasswordTemplate, data)
}

// ResetPasswordEmailData is used to complete the reset_password template.
type ResetPasswordEmailData struct {
	ContactName  string                         // the user's name, if available
	BaseURL      *url.URL                       // the Envoy node's url
	Token        verification.VerificationToken // verification token for reset password link record
	SupportEmail string                         // the Envoy node's support email address
}

func (s ResetPasswordEmailData) VerifyURL() string {
	if s.BaseURL == nil {
		return ""
	}

	params := make(url.Values, 1)
	params.Set("token", s.Token.String())

	s.BaseURL.RawQuery = params.Encode()
	return s.BaseURL.String()
}
