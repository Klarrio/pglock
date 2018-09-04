package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"

	_ "github.com/lib/pq"
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
)

func main() {

	// Extract flags and environment variables
	user := viper.GetString(flagUser)
	pass := viper.GetString(flagPass)
	host := viper.GetString(flagHost)
	port := viper.GetInt(flagPort)
	dbName := viper.GetString(flagDatabase)
	sslMode := viper.GetString(flagSSLMode)
	lockID := viper.GetInt(flagLockID)

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
	lock, err := getLock(db, uint32(lockID))
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

// getLock attempts to take out a lock from the database. Does not block.
// Returns true if the lock was successfully obtained, false when the lock is held by another client.
func getLock(db *sql.DB, lockID uint32) (bool, error) {

	var lock bool
	err := db.QueryRow("SELECT pg_try_advisory_lock($1);", lockID).Scan(&lock)
	if err != nil {
		return false, err
	}

	return lock, nil
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
	pflag.Uint32("lockid", 1, "The numeric lock ID to claim in PostgreSQL.")
	pflag.String("host", "localhost", "Hostname of the PostgreSQL instance.")
	pflag.Uint16("port", 5432, "Port the PostgreSQL instance is listening on.")
	pflag.String("database", "postgres", "Database name to connect to on PostgreSQL.")
	pflag.String("user", "", "Username to authenticate to the PostgreSQL.")
	pflag.String("sslmode", "disable", "The SSL mode of the PostgreSQL client.")

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
