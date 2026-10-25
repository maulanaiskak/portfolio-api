// Command seedcv imports a CV snapshot (JSON) into Postgres in a single
// transaction and verifies row counts before committing.
//
//	go run ./cmd/seedcv                         # seed/cv.json (from the PDF CV)
//	go run ./cmd/seedcv -file legacy-cv.json    # output of scripts/export-cv-data.mjs
//	go run ./cmd/seedcv -replace                # wipe and reload CV tables
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/maulanaiskak/portfolio-api/internal/config"
	"github.com/maulanaiskak/portfolio-api/internal/database"
	"github.com/maulanaiskak/portfolio-api/internal/dto"
	"github.com/maulanaiskak/portfolio-api/internal/server"
)

func main() {
	file := flag.String("file", "seed/cv.json", "path to the CV snapshot JSON")
	replace := flag.Bool("replace", false, "delete existing CV rows before importing")
	flag.Parse()

	if err := run(*file, *replace); err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}
}

func run(file string, replace bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req, err := readSnapshot(file)
	if err != nil {
		return err
	}

	cfg, err := config.NewConfig()
	if err != nil {
		return err
	}
	pool, err := database.NewPool(ctx, cfg.DbConfig)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := database.Migrate(pool); err != nil {
		return err
	}

	cv := req.ToEntity()
	counts, err := server.NewServices(pool).Cv.ImportCv(ctx, cv, replace)
	if err != nil {
		return err
	}

	want := cv.Counts()
	fmt.Printf("%-18s %8s %8s\n", "table", "input", "db")
	fmt.Printf("%-18s %8d %8d\n", "experiences", want.Experiences, counts.Experiences)
	fmt.Printf("%-18s %8d %8d\n", "projects", want.Projects, counts.Projects)
	fmt.Printf("%-18s %8d %8d\n", "education", want.Education, counts.Education)
	fmt.Printf("%-18s %8d %8d\n", "skill categories", want.SkillCategories, counts.SkillCategories)
	fmt.Printf("%-18s %8d %8d\n", "skill items", want.SkillItems, counts.SkillItems)
	fmt.Println("OK: counts match, transaction committed")
	return nil
}

func readSnapshot(path string) (dto.CvImportRequest, error) {
	var req dto.CvImportRequest
	f, err := os.Open(path)
	if err != nil {
		return req, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return req, fmt.Errorf("decode %s: %w", path, err)
	}
	return req, nil
}
