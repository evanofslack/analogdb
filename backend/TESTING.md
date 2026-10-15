# Backend testing

`just backend test` runs everything with `-race`. CI runs packages in parallel next to the
postgres and clickhouse containers, so tests run on a busy machine. Write them to pass there.

## Rules

- **A flaky test may be a real race.** Check the code under test before changing the test, and
  don't add retries. `TestConcurrentReadsDuringPatch` flaked because the post cache could keep a
  stale post.
- **Never sleep to wait for another goroutine.** Synchronize with a channel, a `sync.WaitGroup`
  or a signal from the fake (see `fakeWriter.started` in `events/events_test.go`).
- **Wall-clock bounds only prove "did not wait on the slow thing".** Make the fake slow (seconds)
  and keep the bound at 100ms or more under `-race`, never a few milliseconds.
- **Read `now` once or inject a clock** (`RDB.now` in `redis/`). Assertions on time buckets must
  pass when an hour or day boundary falls between the query and the check.
- **Cache writes that race invalidation set first, then verify.** Re-read the generation or
  delete count after the set and delete on a change. Checking before the set leaves a window.

## Stress a test under load

Run new or changed concurrent tests this way before merging. The `yes` processes stand in for the
container tests CI runs alongside.

```bash
cd backend
for i in $(seq $(( $(sysctl -n hw.ncpu 2>/dev/null || nproc) * 2 ))); do (yes > /dev/null &); done
go test -race -count=300 -cpu=2 -run 'TestName' ./redis/
pkill -x yes
```
