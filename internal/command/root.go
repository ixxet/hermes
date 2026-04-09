package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/ixxet/hermes/internal/athena"
	"github.com/ixxet/hermes/internal/config"
	"github.com/ixxet/hermes/internal/ops"
)

type OccupancyAsker interface {
	AskOccupancy(ctx context.Context, facilityID string) (ops.OccupancyAnswer, error)
}

type ReconciliationAsker interface {
	AskReconciliation(ctx context.Context, facilityID string, window, bin time.Duration) (ops.ReconciliationAnswer, error)
}

type Dependencies struct {
	Stdout                 io.Writer
	Stderr                 io.Writer
	Version                string
	Now                    func() time.Time
	NewRequestID           func() string
	LoadConfig             func() (config.Config, error)
	NewOccupancyAsker      func(config.Config) (OccupancyAsker, error)
	NewReconciliationAsker func(config.Config) (ReconciliationAsker, error)
}

var validFormats = map[string]struct{}{
	"json": {},
	"text": {},
}

var ErrInvalidFormat = errors.New("format must be one of: json, text")

func Execute(args []string, deps Dependencies) error {
	command, trace := newRootCommand(args, deps)
	command.SetArgs(args)
	err := command.Execute()
	if err != nil && trace != nil {
		if !trace.started {
			trace.Start()
		}
		trace.Fail(err)
	}
	return err
}

func NewRootCommand(deps Dependencies) *cobra.Command {
	command, _ := newRootCommand(nil, deps)
	return command
}

func newRootCommand(args []string, deps Dependencies) (*cobra.Command, *occupancyTrace) {
	stdout := deps.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := deps.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	loadConfig := deps.LoadConfig
	if loadConfig == nil {
		loadConfig = config.Load
	}
	newOccupancyAsker := deps.NewOccupancyAsker
	if newOccupancyAsker == nil {
		newOccupancyAsker = func(cfg config.Config) (OccupancyAsker, error) {
			client, err := athena.NewClient(cfg.AthenaBaseURL, cfg.HTTPTimeout)
			if err != nil {
				return nil, err
			}
			return ops.NewOccupancyService(client), nil
		}
	}
	newReconciliationAsker := deps.NewReconciliationAsker
	if newReconciliationAsker == nil {
		newReconciliationAsker = func(cfg config.Config) (ReconciliationAsker, error) {
			client, err := athena.NewClient(cfg.AthenaBaseURL, cfg.HTTPTimeout)
			if err != nil {
				return nil, err
			}
			return ops.NewReconciliationService(client, client), nil
		}
	}
	version := deps.Version
	if version == "" {
		version = "dev"
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	newRequestID := deps.NewRequestID
	if newRequestID == nil {
		newRequestID = nextRequestID
	}

	var trace *occupancyTrace
	if question, tracer, facility, ok := tracedInvocation(args); ok {
		trace = newOccupancyTrace(stderr, now, question, tracer, newRequestID(), facility, version)
	}

	root := &cobra.Command{
		Use:           "hermes",
		Short:         "HERMES read-only staff operations CLI",
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       version,
	}
	root.SetOut(stdout)
	root.SetErr(stderr)

	askCommand := &cobra.Command{
		Use:   "ask",
		Short: "Ask one bounded staff operations question",
	}
	root.AddCommand(askCommand)
	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print HERMES version",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(stdout, version)
			return err
		},
	})

	var (
		facility      string
		format        string
		athenaBaseURL string
		timeout       time.Duration
		window        time.Duration
		bin           time.Duration
	)

	occupancyCommand := &cobra.Command{
		Use:   "occupancy",
		Short: "Read current facility occupancy from ATHENA",
		RunE: func(cmd *cobra.Command, args []string) error {
			if trace != nil {
				trace.Start()
			}

			if _, ok := validFormats[format]; !ok {
				return ErrInvalidFormat
			}

			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			cfg, err = cfg.WithOverrides(athenaBaseURL, timeout)
			if err != nil {
				return err
			}

			asker, err := newOccupancyAsker(cfg)
			if err != nil {
				return err
			}

			answer, err := asker.AskOccupancy(cmd.Context(), facility)
			if err != nil {
				return err
			}

			var writeErr error
			switch format {
			case "json":
				writeErr = writeJSON(stdout, answer)
			case "text":
				_, writeErr = fmt.Fprintf(stdout, "facility_id=%s current_count=%d observed_at=%s source_service=%s\n", answer.FacilityID, answer.CurrentCount, answer.ObservedAt, answer.SourceService)
			}
			if writeErr != nil {
				return writeErr
			}

			if trace != nil {
				trace.Complete(answer.FacilityID, answer.CurrentCount)
			}

			return nil
		},
	}
	occupancyCommand.Flags().StringVar(&facility, "facility", "", "facility identifier to query")
	occupancyCommand.Flags().StringVar(&format, "format", "json", "output format: json or text")
	occupancyCommand.Flags().StringVar(&athenaBaseURL, "athena-base-url", "", "ATHENA base URL override")
	occupancyCommand.Flags().DurationVar(&timeout, "timeout", 0, "HTTP timeout override")
	_ = occupancyCommand.MarkFlagRequired("facility")
	askCommand.AddCommand(occupancyCommand)

	reconciliationCommand := &cobra.Command{
		Use:   "reconciliation",
		Short: "Read a stable-history occupancy reconciliation report from ATHENA",
		RunE: func(cmd *cobra.Command, args []string) error {
			if trace != nil {
				trace.Start()
			}

			if _, ok := validFormats[format]; !ok {
				return ErrInvalidFormat
			}

			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			cfg, err = cfg.WithOverrides(athenaBaseURL, timeout)
			if err != nil {
				return err
			}

			asker, err := newReconciliationAsker(cfg)
			if err != nil {
				return err
			}

			answer, err := asker.AskReconciliation(cmd.Context(), facility, window, bin)
			if err != nil {
				return err
			}

			var writeErr error
			switch format {
			case "json":
				writeErr = writeJSON(stdout, answer)
			case "text":
				writeErr = writeReconciliationText(stdout, answer)
			}
			if writeErr != nil {
				return writeErr
			}

			if trace != nil {
				trace.Complete(answer.FacilityID, answer.Current.CurrentCount)
			}

			return nil
		},
	}
	reconciliationCommand.Flags().StringVar(&facility, "facility", "", "facility identifier to query")
	reconciliationCommand.Flags().DurationVar(&window, "window", 24*time.Hour, "stable history window to summarize")
	reconciliationCommand.Flags().DurationVar(&bin, "bin", time.Hour, "heat-map bucket size")
	reconciliationCommand.Flags().StringVar(&format, "format", "json", "output format: json or text")
	reconciliationCommand.Flags().StringVar(&athenaBaseURL, "athena-base-url", "", "ATHENA base URL override")
	reconciliationCommand.Flags().DurationVar(&timeout, "timeout", 0, "HTTP timeout override")
	_ = reconciliationCommand.MarkFlagRequired("facility")
	askCommand.AddCommand(reconciliationCommand)

	return root, trace
}

func writeJSON(writer io.Writer, payload any) error {
	encoder := json.NewEncoder(writer)
	return encoder.Encode(payload)
}

func writeReconciliationText(writer io.Writer, answer ops.ReconciliationAnswer) error {
	if _, err := fmt.Fprintf(
		writer,
		"facility_id=%s source_service=%s window_start=%s window_end=%s current_count=%d current_observed_at=%s opening_count=%d net_change=%d committed_entries=%d committed_exits=%d failed_observations=%d observed_pass_without_change=%d peak_occupancy=%d peak_observed_at=%s\n",
		answer.FacilityID,
		answer.SourceService,
		answer.WindowStart,
		answer.WindowEnd,
		answer.Current.CurrentCount,
		answer.Current.ObservedAt,
		answer.Report.OpeningCount,
		answer.Report.NetChange,
		answer.Report.CommittedEntries,
		answer.Report.CommittedExits,
		answer.Report.FailedObservations,
		answer.Report.ObservedPassWithoutChange,
		answer.Report.PeakOccupancy,
		answer.Report.PeakObservedAt,
	); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(
		writer,
		"inspect_next=%s reason=%q window_start=%s window_end=%s\n",
		answer.InspectNext.Category,
		answer.InspectNext.Reason,
		answer.InspectNext.WindowStart,
		answer.InspectNext.WindowEnd,
	); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(writer, "heat_map:"); err != nil {
		return err
	}

	for _, cell := range answer.HeatMap {
		if _, err := fmt.Fprintf(
			writer,
			"window_start=%s window_end=%s heat_level=%d occupancy_peak=%d occupancy_end=%d committed_entries=%d committed_exits=%d failed_observations=%d observed_pass_without_change=%d\n",
			cell.WindowStart,
			cell.WindowEnd,
			cell.HeatLevel,
			cell.OccupancyPeak,
			cell.OccupancyEnd,
			cell.CommittedEntries,
			cell.CommittedExits,
			cell.FailedObservations,
			cell.ObservedPassWithoutChange,
		); err != nil {
			return err
		}
	}

	return nil
}
