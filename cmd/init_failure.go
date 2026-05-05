package cmd

import (
	"fmt"
	"io"
)

// FailedStep names the init step that hard-failed. Emitted in the structured
// failure trailer so callers (e.g., px-manager) can surface accurate recovery
// UX without screen-scraping. label-setup is intentionally excluded — that
// loop is warn+continue per AC5 of #847 and never reaches the trailer.
const (
	FailedStepCreateProject          = "create-project"
	FailedStepGetFields              = "get-fields"
	FailedStepValidateRequiredFields = "validate-required-fields"
	FailedStepRefetchFields          = "refetch-fields"
	FailedStepWriteConfig            = "write-config"
	FailedStepSourceCopy             = "source-copy"
)

// AllFailedSteps is the closed enum of values printInitFailureTrailer may emit
// for failedStep. Tests pin this set to keep the contract stable for callers.
var AllFailedSteps = []string{
	FailedStepCreateProject,
	FailedStepGetFields,
	FailedStepValidateRequiredFields,
	FailedStepRefetchFields,
	FailedStepWriteConfig,
	FailedStepSourceCopy,
}

// printInitFailureTrailer writes a structured machine-readable summary of an
// init failure. Format is labeled key=value lines on the supplied writer
// (typically stderr). projectNumber == 0 renders as "none" to disambiguate
// "no project created yet" from "project #0".
func printInitFailureTrailer(w io.Writer, projectNumber int, failedStep string, configRewritten bool) {
	projectField := "none"
	if projectNumber > 0 {
		projectField = fmt.Sprintf("%d", projectNumber)
	}
	fmt.Fprintln(w, "gh-pmu init failed:")
	fmt.Fprintf(w, "  destinationProjectNumber=%s\n", projectField)
	fmt.Fprintf(w, "  failedStep=%s\n", failedStep)
	fmt.Fprintf(w, "  configRewritten=%t\n", configRewritten)
}
