// Command hashpassword prints a bcrypt hash for ADMIN_PASSWORD_BCRYPT_HASH.
// The password is read from the terminal without echo (or from stdin when
// piped), so it never appears in shell history or process arguments.
//
//	go run ./cmd/hashpassword
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

const cost = 12

func main() {
	password, err := readPassword()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if len(password) < 12 {
		fmt.Fprintln(os.Stderr, "error: use at least 12 characters")
		os.Exit(1)
	}
	if len(password) > 72 {
		fmt.Fprintln(os.Stderr, "error: bcrypt only supports up to 72 bytes")
		os.Exit(1)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println(string(hash))
}

func readPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	}
	fmt.Fprint(os.Stderr, "Password: ")
	first, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	fmt.Fprint(os.Stderr, "Confirm:  ")
	second, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	if string(first) != string(second) {
		return "", fmt.Errorf("passwords do not match")
	}
	return string(first), nil
}
