package cmd

import (
	"context"

	"github.com/felixgeelhaar/specular/internal/session"
)

// sessionWaitPostOptions is the fleet→evidence post-wait pipeline.
type sessionWaitPostOptions struct {
	Ctx             context.Context
	Mgr             *session.Manager
	Recs            []session.Record
	Attest          bool
	Gate            bool
	Bundle          bool
	BundleOut       string
	Policies        []string
	Quiet           bool
	RequireAttested bool
	RequireProtocol bool
	RequireGoverned bool
}

// runSessionWaitPost runs optional attest → gate → evidence bundle after wait.
// --bundle implies gate; also attests waited terminal sessions when any exist.
func runSessionWaitPost(opts sessionWaitPostOptions) error {
	doAttest := opts.Attest
	doGate := opts.Gate
	if opts.Bundle {
		doGate = true
		if len(opts.Recs) > 0 {
			doAttest = true
		}
	}

	if doAttest && opts.Mgr != nil && len(opts.Recs) > 0 {
		if err := attestSessionRecords(opts.Ctx, opts.Mgr, opts.Recs, !opts.Quiet); err != nil {
			return err
		}
	}
	if doGate {
		if err := runSessionProductGate(sessionProductGateOptions{
			Quiet:           opts.Quiet,
			RequireAttested: opts.RequireAttested,
			RequireProtocol: opts.RequireProtocol,
			RequireGoverned: opts.RequireGoverned,
		}); err != nil {
			return err
		}
	}
	if opts.Bundle {
		return runSessionEvidenceBundle(sessionEvidenceBundleOptions{
			Output:   opts.BundleOut,
			Policies: opts.Policies,
			Quiet:    opts.Quiet,
		})
	}
	return nil
}
