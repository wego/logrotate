package main

import (
	"log"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

func execute(command string, args []string) {

	var sb strings.Builder
	sb.WriteString(command)
	sb.WriteString(strings.Join(args, " "))

	runId := uuid.NewString()

	slog.Info("executing",
		slog.String("uuid", runId),
		slog.String("command", sb.String()))

	cmd := exec.Command(command, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()

	cmd.Wait()

	slog.Info("done",
		slog.String("uuid", runId))
}

func create(wg *sync.WaitGroup) *cron.Cron {

	var schedule string = os.Args[1]
	var command string = os.Args[2]
	var args []string = os.Args[3:len(os.Args)]

	c := cron.New(
		cron.WithParser(
			cron.NewParser(
				cron.SecondOptional|cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow|cron.Descriptor)),
		cron.WithChain(
			cron.SkipIfStillRunning(cron.VerbosePrintfLogger(log.Default()))))

	slog.Info("new cron",
		slog.String("schedule", schedule))

	c.AddFunc(schedule, func() {
		wg.Add(1)
		execute(command, args)
		wg.Done()
	})

	return c
}

func stop(c *cron.Cron, wg *sync.WaitGroup) {

	slog.Info("stopping")
	c.Stop()

	slog.Info("waiting")
	wg.Wait()

	slog.Info("exiting")
	os.Exit(0)
}

func main() {

	var wg sync.WaitGroup

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	c := create(&wg)

	c.Start()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	slog.Info("received signal",
		slog.String("signal", (<-ch).String()))

	stop(c, &wg)
}
