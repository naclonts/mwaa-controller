	// AirflowConfigurationOptions: GetEnvironment returns redacted values
	// ("***") for sensitive config options. Preserve the desired values to
	// prevent false drift detection and infinite update loops.
	//
	// NOTE: This means we cannot detect drift for these fields if they are
	// changed out-of-band (e.g. via the console). The MWAA API does not
	// provide unredacted values, so there is no way to compare. If a user
	// changes a sensitive config option outside the controller, the
	// controller will not notice until the user updates the CR spec.
	if r.ko.Spec.AirflowConfigurationOptions != nil {
		ko.Spec.AirflowConfigurationOptions = r.ko.Spec.AirflowConfigurationOptions
	}

	// Return terminal error for failure states so updateConditions preserves it.
	// NOTE: UPDATE_FAILED is intentionally excluded — MWAA rolls back to the
	// previous configuration and the environment returns to AVAILABLE, so the
	// controller should requeue and let the user fix their spec.
	if ko.Status.Status != nil {
		status := *ko.Status.Status
		if status == string(svcsdktypes.EnvironmentStatusCreateFailed) ||
			status == string(svcsdktypes.EnvironmentStatusUnavailable) {
			return &resource{ko}, ackerr.NewTerminalError(
				fmt.Errorf("environment is in terminal state: %s", status),
			)
		}
	}
