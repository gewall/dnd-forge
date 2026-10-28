// Command tools holds one-off maintenance commands.
//
//	tools migrate                                  apply database migrations
//	tools seed-admin -email a@b.c [-password ...]  create or promote an admin
//	tools openapi [-o openapi.json]                write the OpenAPI document
//	tools gen-keys                                 print fresh secrets for .env
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/gewall/dnd-forge/api/internal/app"
	"github.com/gewall/dnd-forge/api/internal/config"
	"github.com/gewall/dnd-forge/api/internal/db"
)

const usage = `usage: tools <command> [flags]

commands:
  migrate      apply database migrations (goose + River)
  seed-admin   create or promote an admin account
  openapi      write the OpenAPI 3.1 document
  gen-keys     print JWT_PRIVATE_KEY, ENCRYPTION_KEY and INTERNAL_API_KEY`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "migrate":
		err = migrate(ctx)
	case "seed-admin":
		err = seedAdmin(ctx, args)
	case "openapi":
		err = openAPI(args)
	case "gen-keys":
		err = genKeys()
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", cmd, err)
		os.Exit(1)
	}
}

func migrate(ctx context.Context) error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return errors.New("DATABASE_URL is required")
	}
	pool, err := db.Open(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()
	return db.Migrate(ctx, pool, slog.Default())
}

func seedAdmin(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("seed-admin", flag.ContinueOnError)
	email := fs.String("email", "", "admin email (required)")
	password := fs.String("password", "", "password; generated when empty")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *email == "" {
		return errors.New("-email is required")
	}
	if *password == "" {
		*password = randomString(18)
		fmt.Printf("generated password: %s\n", *password)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	a, err := app.New(ctx, cfg, slog.Default())
	if err != nil {
		return err
	}
	defer a.Close()

	u, err := a.Auth.EnsureAdmin(ctx, *email, *password)
	if err != nil {
		return err
	}
	fmt.Printf("admin ready: %s (%s)\n", u.Email, u.ID)
	return nil
}

func openAPI(args []string) error {
	fs := flag.NewFlagSet("openapi", flag.ContinueOnError)
	out := fs.String("o", "openapi.json", "output file, - for stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	doc, err := app.OpenAPI()
	if err != nil {
		return err
	}
	doc = append(doc, '\n')
	if *out == "-" {
		_, err = os.Stdout.Write(doc)
		return err
	}
	return os.WriteFile(*out, doc, 0o600)
}

func genKeys() error {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	enc := make([]byte, 32)
	if _, err := rand.Read(enc); err != nil {
		return err
	}
	fmt.Printf("JWT_PRIVATE_KEY=%s\n", base64.StdEncoding.EncodeToString(priv.Seed()))
	fmt.Printf("ENCRYPTION_KEY=%s\n", base64.StdEncoding.EncodeToString(enc))
	fmt.Printf("INTERNAL_API_KEY=%s\n", randomString(32))
	return nil
}

func randomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
