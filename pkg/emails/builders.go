package emails

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
	OriginatorName  string
	BeneficiaryName string
	VerifyURL       string
}

//===========================================================================
// Verify Email
//===========================================================================

const (
	VerifyEmailRE       = "One time verification code"
	VerifyEmailTemplate = "verify_email"
)

func NewVerifyEmail(recipient string, data VerifyEmailData) (*Email, error) {
	return New(recipient, VerifyEmailRE, VerifyEmailTemplate, data)
}

// VerifyEmailData is used to send a one-time code to the original email for verification.
type VerifyEmailData struct {
	Code string
}
