package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/lib/pq"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// Names of the flags taken by the application
	flagUser     = "user"
	flagPass     = "pass"
	flagHost     = "host"
	flagPort     = "port"
	flagDatabase = "database"
	flagLockID   = "lockid"
	flagSSLMode  = "sslmode"
	flagWait     = "wait"
)

func main() {

	// Extract flags and environment variables
	user := viper.GetString(flagUser)
	pass := viper.GetString(flagPass)
	host := viper.GetString(flagHost)
	port := viper.GetInt(flagPort)
	dbName := viper.GetString(flagDatabase)
	sslMode := viper.GetString(flagSSLMode)
	wait := uint32(viper.GetInt(flagWait))
	lockID := uint32(viper.GetInt(flagLockID))

	// List of positional arguments after all flags have been parsed.
	cmd := pflag.Args()

	log.Printf("Connecting to PostgreSQL at %s@%s:%d/%s.", user, host, port, dbName)

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		user, pass, host, port, dbName, sslMode,
	)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error connecting PostgreSQL: %s", err.Error())
	}

	defer db.Close()

	// Take out lock from database
	var lock bool
	if wait == 0 {
		log.Printf("Trying to obtain lock.")
		lock, err = getLockTry(db, lockID)
	} else {
		log.Printf("Obtaining lock with %d seconds timeout..", wait)
		lock, err = getLockWait(db, lockID, wait)
	}

	if err != nil {
		log.Fatalf("Error trying to obtain lock: %s", err.Error())
	}

	if lock {
		// Lock obtained, run the subprocess
		log.Printf("Lock ID %d obtained successfully!", lockID)
		log.Printf("Executing command: %s", cmd)

		err := exec.Command(cmd[0], cmd[1:]...).Run()
		if err != nil {
			log.Fatal(err)
		}

		log.Print("Execution finished, exiting and cleaning up lock.")
		os.Exit(0)
	} else {
		log.Printf("Could not obtain lock %d, skipping command", lockID)
	}

	os.Exit(0)
}

// getLockTry attempts to take out a lock from the database. Always immediately returns.
// Returns true if the lock was successfully obtained, false when the lock is held by another client.
func getLockTry(db *sql.DB, lockID uint32) (bool, error) {

	var lock bool
	err := db.QueryRow("SELECT pg_try_advisory_lock($1);", lockID).Scan(&lock)
	if err != nil {
		return false, err
	}

	return lock, nil
}

// getLockWait attempts to take out a lock from the database, blocking until the lock is available.
// Returns true if the lock was successfully obtained, false when the lock is (still) held by another
// client at the time of the expiry of the wait timer.
func getLockWait(db *sql.DB, lockID uint32, wait uint32) (bool, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
	defer cancel()

	// pg_advisory_lock does not return any values. If the query returns,
	// the lock was obtained.
	_, err := db.QueryContext(ctx, "SELECT pg_advisory_lock($1);", lockID)
	if err, ok := err.(*pq.Error); ok {
		// If the query was cancelled, return false (no lock obtained)
		// 57014 is the postgres error code for canceled queries
		if err.Code == "57014" {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}

func init() {

	pflag.Usage = func() {
		fmt.Fprint(os.Stderr, `pglock runs a command while holding a lock in a PostgreSQL database.

Any concurrent invocations that fail to obtain a lock will not run
the given command and will exit with return code 0.

The flags below can be configured as environment variables with the PGLOCK_ prefix.
PGLOCK_PASS needs to be configured to authenticate to the database.

`)

		pflag.PrintDefaults()
	}

	// Listen to environment variables with PGLOCK_ prefix.
	viper.SetEnvPrefix("pglock")

	// Command line flags
	pflag.Uint32(flagLockID, 1, "The numeric lock ID to claim in PostgreSQL.")
	pflag.String(flagHost, "localhost", "Hostname of the PostgreSQL instance.")
	pflag.Uint16(flagPort, 5432, "Port the PostgreSQL instance is listening on.")
	pflag.String(flagDatabase, "postgres", "Database name to connect to on PostgreSQL.")
	pflag.String(flagUser, "", "Username to authenticate to the PostgreSQL.")
	pflag.String(flagSSLMode, "disable", "The SSL mode of the PostgreSQL client.")
	pflag.Uint32(flagWait, 0, "Amount of seconds to wait for a lock to be obtained.")

	pflag.Parse()

	// Only get pass from the environment, specifying on command line
	// will leave password in shell history.
	viper.BindEnv("pass")

	viper.AutomaticEnv()

	// Load command line flag values into Viper
	viper.BindPFlags(pflag.CommandLine)

	// We need at least one positional argument to invoke as a command
	if pflag.NArg() == 0 {
		log.Fatal("Need at least one positional argument (command to run).")
	}

	if viper.GetString(flagUser) == "" {
		log.Fatal("Username cannot be empty.")
	}
}
