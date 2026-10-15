package gemini

// Code is a Gemini status code.
// An unknown value prints as unknown.
//
//lewkit:enum invalid=unknown
type Code int

const (
	CodeInput                    Code = 10 // input
	CodeSensitiveInput           Code = 11 // sensitive
	CodeSuccess                  Code = 20 // success
	CodeRedirectTemporary        Code = 30 // redirect
	CodeRedirectPermanent        Code = 31 // permanent
	CodeTemporaryFailure         Code = 40 // temporary
	CodeUnavailable              Code = 41 // unavailable
	CodeCGIError                 Code = 42 // cgi
	CodeProxyError               Code = 43 // proxy
	CodeSlowDown                 Code = 44 // slowdown
	CodePermanentFailure         Code = 50 // failure
	CodeNotFound                 Code = 51 // notfound
	CodeGone                     Code = 52 // gone
	CodeProxyRefused             Code = 53 // refused
	CodeBadRequest               Code = 59 // badrequest
	CodeClientCertificate        Code = 60 // certificate
	CodeCertificateNotAuthorized Code = 61 // unauthorized
	CodeCertificateInvalid       Code = 62 // invalid
)

// Category returns the [StatusCategory] for a Gemini status code.
func Category(status Code) StatusCategory {
	switch int(status) / 10 {
	case 1:
		return StatusInput
	case 2:
		return StatusSuccess
	case 3:
		return StatusRedirect
	case 4:
		return StatusTempFailure
	case 5:
		return StatusPermFailure
	case 6:
		return StatusClientCert
	default:
		return 0
	}
}
