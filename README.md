# `pglock`

Use PostgreSQL advisory locking to run commands safely and serially over many
concurrent invocations. Inspired by `consul lock`.

---

## Usage

```
~ pglock --help

pglock runs a command while holding a lock in a PostgreSQL database.
Any concurrent invocations that fail to obtain a lock will not run
the given command and will exit with return code 0.

The flags below can be configured as environment variables with the PGLOCK_ prefix.
PGLOCK_PASS needs to be configured to authenticate to the database.

      --database string   Database name to connect to on PostgreSQL. (default "postgres")
      --host string       Hostname of the PostgreSQL instance. (default "localhost")
      --lockid uint32     The numeric lock ID to claim in PostgreSQL. (default 1)
      --port uint16       Port the PostgreSQL instance is listening on. (default 5432)
      --sslmode string    The SSL mode of the PostgreSQL client. (default "disable")
      --user string       Username to authenticate to the PostgreSQL.
      --wait uint32       Amount of seconds to wait for a lock to be obtained.
```

### Non-blocking (fall-through)

```
export PGLOCK_PASS="secret-password"
pglock --user postgres sleep 2
2018/09/05 11:17:39 Connecting to PostgreSQL at postgres@localhost:5432/postgres.
2018/09/05 11:17:39 Trying to obtain lock.
2018/09/05 11:17:40 Lock ID 1 obtained successfully!
2018/09/05 11:17:40 Executing command: [sleep 2]
2018/09/05 11:17:42 Execution finished, exiting and cleaning up lock.

```

When executed on many hosts concurrently, only the first agent that manages to
obtain a lock will execute, the others will exit. This does not ensure that
all agents will run their commands; see [Blocking](#Blocking) for this use case.

### Blocking

Blocking mode waits for the lock to be obtained, up to a given timeout
(in seconds). This will guarantee that all agents execute their commands,
as long as each of them can obtain the lock within the timeout.

```
export PGLOCK_PASS="secret-password"
pglock --user postgres --wait 10 sleep 2
2018/09/05 11:22:14 Connecting to PostgreSQL at postgres@localhost:5432/postgres.
2018/09/05 11:22:14 Obtaining lock with 10 seconds timeout..
2018/09/05 11:22:14 Lock ID 1 obtained successfully!
2018/09/05 11:22:14 Executing command: [sleep 2]
2018/09/05 11:22:16 Execution finished, exiting and cleaning up lock.
```

When running multiple concurrent instances in blocking mode, the following
output would be expected:

```
export PGLOCK_PASS="secret-password"
pglock --user postgres --wait 2 sleep 10
2018/09/05 11:27:50 Connecting to PostgreSQL at postgres@localhost:5432/postgres.
2018/09/05 11:27:50 Obtaining lock with 2 seconds timeout..
2018/09/05 11:27:53 Could not obtain lock 1, skipping command
```

The given `sleep 10` command will not be executed, and the process will return
with return code 0.
