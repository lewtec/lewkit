package dotenv

// Quote is how an assignment value is written.
// Bare writes the bytes as they are. Single quotes keep every byte.
// Double quotes use backslash escapes.
//
//lewkit:enum
type Quote uint8

const (
	QuoteBare   Quote = iota + 1 // bare
	QuoteSingle                  // single
	QuoteDouble                  // double
)
