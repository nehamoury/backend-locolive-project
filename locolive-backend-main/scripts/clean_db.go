package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	// Database connection URL
	dbURL := "postgresql://postgres:admin123@127.0.0.1:5432/privacy_social?sslmode=disable"

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// SQL to truncate all tables in the public schema
	query := `
		DO $$ 
		DECLARE 
			r RECORD; 
		BEGIN 
			FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename != 'spatial_ref_sys') LOOP 
				EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' CASCADE;'; 
			END LOOP; 
		END $$;
	`

	fmt.Println("🚀 Cleaning local database...")
	_, err = db.ExecContext(context.Background(), query)
	if err != nil {
		log.Fatalf("Failed to clean database: %v", err)
	}

	fmt.Println("✅ Success! All tables truncated. You can now signup as a fresh user.")
}
