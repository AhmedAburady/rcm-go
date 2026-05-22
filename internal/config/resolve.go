package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
)

type resolveTask struct {
	ptr *string
	raw string
}

// resolveRefs resolves op:// (1Password) and ${ENV} references in the given
// struct pointers, walking each recursively.
//
// This is the single place that performs secret-bearing authentication against
// 1Password (via `op inject`). Callers therefore control *when* that
// authentication happens by choosing what to pass and when to call it — see
// Load (connectivity, eager) and ResolveRatholeSecrets (deployment secrets,
// deferred to the point of consumption).
func resolveRefs(targets ...any) error {
	var tasks []resolveTask
	for _, t := range targets {
		v := reflect.ValueOf(t)
		if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("resolveRefs: target must be a non-nil pointer to struct, got %T", t)
		}
		tasks = append(tasks, collectTasks(v.Elem())...)
	}
	if len(tasks) == 0 {
		return nil
	}

	// Resolve ${ENV} refs inline, collect op:// refs for batch resolution.
	var opTasks []resolveTask
	for _, t := range tasks {
		if strings.Contains(t.raw, "${") {
			*t.ptr = expandEnvVars(t.raw)
		} else {
			opTasks = append(opTasks, t)
		}
	}

	if err := opInjectBatch(opTasks); err != nil {
		return fmt.Errorf("failed to resolve config values:\n%w", err)
	}
	return nil
}

// opInjectBatch resolves all op:// references in a single `op inject` call.
func opInjectBatch(tasks []resolveTask) error {
	if len(tasks) == 0 {
		return nil
	}

	// Build template: {{ op://ref1 }}<delim>{{ op://ref2 }}...
	const delim = "\n---RCM_SEP---\n"
	parts := make([]string, len(tasks))
	for i, t := range tasks {
		parts[i] = "{{ " + t.raw + " }}"
	}
	input := strings.Join(parts, delim)

	cmd := exec.Command("op", "inject")
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("op inject: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return fmt.Errorf("op inject: %w", err)
	}

	values := strings.Split(string(out), delim)
	if len(values) != len(tasks) {
		return fmt.Errorf("op inject: expected %d values, got %d", len(tasks), len(values))
	}

	var errs []error
	for i, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			errs = append(errs, fmt.Errorf("op inject: empty value for %q", tasks[i].raw))
			continue
		}
		*tasks[i].ptr = trimmed
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// collectTasks recursively walks struct fields and collects string fields needing resolution.
func collectTasks(v reflect.Value) []resolveTask {
	var tasks []resolveTask

	for i := range v.NumField() {
		field := v.Field(i)

		switch field.Kind() {
		case reflect.Struct:
			tasks = append(tasks, collectTasks(field)...)
		case reflect.String:
			s := field.String()
			if strings.HasPrefix(s, "op://") || strings.Contains(s, "${") {
				tasks = append(tasks, resolveTask{
					ptr: field.Addr().Interface().(*string),
					raw: s,
				})
			}
		}
	}

	return tasks
}

func expandEnvVars(s string) string {
	return os.Expand(s, os.Getenv)
}
