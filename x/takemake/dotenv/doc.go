// Package dotenv is one assignment list.
//
// [New] returns an empty [File]. Methods return the file and store each
// assignment in order. [File.Comment], [File.Bare], [File.Single],
// [File.Double], and [File.Export] record the note, the quoting, and the
// export prefix on the latest assignment. [Take] parses the text. [Make]
// prints it. A comment and the quoting survive Take, Make, Take.
// x/sops.ParseEnv drops comments. The contract is in DESIGN.md beside
// the parent directory.
package dotenv
