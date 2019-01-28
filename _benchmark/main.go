package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gofrs/uuid"
	"github.com/kamichidu/goen"
)

type benchmarkStats struct {
	Count int64

	NumChunks int64

	TotalDuration time.Duration

	MinDuration time.Duration

	MaxDuration time.Duration

	MeanDuration time.Duration

	Throughput float64
}

func printStats(title string, stats *benchmarkStats, err error) {
	fmt.Printf("%v\n", title)
	if err != nil {
		fmt.Printf("error      = %v\n", err)
	} else {
		fmt.Printf("count      = %v\n", stats.Count)
		fmt.Printf("chunks     = %v\n", stats.NumChunks)
		fmt.Printf("total time = %v\n", stats.TotalDuration)
		fmt.Printf("min time   = %v spent per chunk\n", stats.MinDuration)
		fmt.Printf("max time   = %v spent per chunk\n", stats.MaxDuration)
		fmt.Printf("mean time  = %v spent per chunk\n", stats.MeanDuration)
		fmt.Printf("throughput = %.2f records per second\n", stats.Throughput)
	}
}

type runner struct {
	DialectName string

	DB *sql.DB

	Compiler goen.PatchCompiler

	ChunkNum int

	UsePreparedStatement bool
}

var stmtCacher *goen.StmtCacher

func (r *runner) Setup() {
	if _, err := r.DB.Exec(ddl); err != nil {
		panic(fmt.Sprintf("failed to create schema: %v", err))
	}
}

func (r *runner) TearDown() {
	if _, err := r.DB.Exec(`delete from users`); err != nil {
		panic(fmt.Sprintf("failed to clean up: %v", err))
	}
}

func (r *runner) Do() int64 {
	dbc := goen.NewDBContext(r.DialectName, r.DB)
	dbc.Compiler = r.Compiler
	if r.UsePreparedStatement {
		dbc.QueryRunner = stmtCacher
	}
	for n := r.ChunkNum; n > 0; n-- {
		dbc.Patch(goen.InsertPatch(
			"users",
			[]string{
				"id",
				"name",
				"created_at",
			},
			[]interface{}{
				uuid.Must(uuid.NewV4()).String(),
				uuid.Must(uuid.NewV4()).String(),
				time.Now(),
			}))
	}
	if err := dbc.SaveChanges(); err != nil {
		panic(fmt.Sprintf("SaveChanges: %v", err))
	}
	return int64(r.ChunkNum)
}

func doBenchmark(d time.Duration, fn func() int64) (stats *benchmarkStats, err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("PANIC:%v", v)
		}
	}()

	stopTimer := time.After(d)
	stats = new(benchmarkStats)
loop:
	for {
		select {
		case <-stopTimer:
			break loop
		default:
		}
		st := time.Now()
		cnt := fn()
		d := time.Since(st)
		stats.NumChunks++
		stats.Count += cnt
		stats.TotalDuration += d
		if stats.MinDuration == 0 || stats.MinDuration > d {
			stats.MinDuration = d
		}
		if stats.MaxDuration == 0 || stats.MaxDuration < d {
			stats.MaxDuration = d
		}
	}
	if stats.NumChunks > 0 {
		stats.MeanDuration = stats.TotalDuration / time.Duration(stats.NumChunks)
	}
	if stats.TotalDuration > 0 {
		stats.Throughput = float64(stats.Count) / stats.TotalDuration.Seconds()
	}
	return stats, nil
}

func usageFunc(flgs *flag.FlagSet) func() {
	return func() {
		p := func(s string, a ...interface{}) {
			fmt.Fprintf(flgs.Output(), s, a...)
		}

		p("Usage:\n")
		p("  %v [options]\n", flgs.Name())
		p("\n")
		flgs.PrintDefaults()
	}
}

func main() {
	flgs := flag.NewFlagSet("goen-benchmark", flag.ExitOnError)
	flgs.SetOutput(os.Stderr)
	durationStr := flgs.String("duration", "30s", "The duration of benchmark")
	chunkNum := flgs.Int("chunkNum", 1, `The number of chunk size; affected when -compiler="bulk"`)
	compilerStr := flgs.String("compiler", "default", `Which compiler will be used (choices: "default", "bulk")`)
	enablePreparedStatement := flgs.Bool("enable-prepared-statement", false, "Enable prepared statement")
	flgs.Usage = usageFunc(flgs)
	if err := flgs.Parse(os.Args[1:]); err != nil {
		panic(err)
	}
	duration, err := time.ParseDuration(*durationStr)
	if err != nil {
		panic(err)
	}
	var compiler goen.PatchCompiler
	switch s := *compilerStr; s {
	case "default":
		*chunkNum = 1
		compiler = goen.DefaultCompiler
	case "bulk":
		compiler = goen.BulkCompiler
		if *chunkNum <= 0 {
			panic(fmt.Sprintf("Invalid chunk number: %v", *chunkNum))
		}
	default:
		panic(fmt.Sprintf("Invalid compiler %q", s))
	}

	fmt.Println("# Benchmark parameters")
	fmt.Printf("duration = %v\n", duration)
	fmt.Printf("chunkNum = %v\n", *chunkNum)
	fmt.Printf("compiler = %v\n", *compilerStr)
	fmt.Printf("enablePreparedStatement = %v\n", *enablePreparedStatement)
	fmt.Println()

	dialectName, db := openDB()
	defer db.Close()

	stmtCacher = goen.NewStmtCacher(db)
	defer stmtCacher.Close()

	r := &runner{
		DialectName:          dialectName,
		DB:                   db,
		Compiler:             compiler,
		ChunkNum:             *chunkNum,
		UsePreparedStatement: *enablePreparedStatement,
	}

	r.Setup()
	defer r.TearDown()

	stats, err := doBenchmark(duration, r.Do)
	printStats("# Benchmark result", stats, err)

	stmtStats := stmtCacher.StmtStats()
	fmt.Println()
	fmt.Println("# Prepared statement stats")
	fmt.Printf("cached statements = %v\n", stmtStats.CachedStmts)
}
