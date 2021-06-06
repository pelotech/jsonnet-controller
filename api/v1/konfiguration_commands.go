package v1

func (k *Konfiguration) newArgs(cmd string) []string {
	args := []string{cmd, "--namespace", k.GetNamespace()}

	// Add any global arguments provided by the user.
	if globalArgs := k.GetKubecfgArgs(); len(globalArgs) != 0 {
		args = append(args, globalArgs...)
	}

	return args
}

// ToUpdateArgs converts this Konfiguration schema into kubecfg update
// arguments.
func (k *Konfiguration) ToUpdateArgs(dryRun bool) []string {
	args := k.newArgs("update")

	// Check if we are adding garbage collection flags.
	if k.GCEnabled() {
		args = append(args, []string{"--gc-tag", k.GetClusterName()}...)
	}

	// Check if disabling validation.
	if !k.ValidateEnabled() {
		args = append(args, "--validate=false")
	}

	// Check if defining external or top-level arguments.
	if vars := k.GetVariables(); vars != nil {
		args = vars.AppendToArgs(args)
	}

	if dryRun {
		args = append(args, "--dry-run")
	}

	// Finally add the path
	args = append(args, k.GetPaths()...)

	return args
}

// ToDiffArgs converts this Konfiguration schema into kubecfg diff arguments.
func (k *Konfiguration) ToDiffArgs() []string {
	args := k.newArgs("diff")
	// Check if defining external or top-level arguments.
	if vars := k.GetVariables(); vars != nil {
		args = vars.AppendToArgs(args)
	}
	// Append the diff strategy
	args = append(args, []string{"--diff-strategy", k.GetDiffStrategy()}...)
	return args
}
