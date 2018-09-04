# `pglock`

Use PostgreSQL advisory locking to run a command exactly once over many
concurrent invocations. Inspired by `consul lock`.

---

## Usage

```
export PGLOCK_PASS="secret-password"
pglock --user postgres sleep 2
```

When executed on many hosts concurrently, only the first agent
that manages to obtain a lock will execute, the others will exit.

Does not (yet) support retries or waiting for other processes to terminate.
