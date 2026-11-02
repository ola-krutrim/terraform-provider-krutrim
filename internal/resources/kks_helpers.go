package resources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	krutrim "github.com/ola-krutrim/krutrim-go-sdk"
)

const (
	kksPollInterval = 15 * time.Second
	kksCreateTimeout = 45 * time.Minute
	kksDeleteTimeout = 30 * time.Minute
)

type kksNotFoundError struct {
	resource string
}

func (e kksNotFoundError) Error() string {
	if e.resource == "" {
		return "kks resource not found"
	}
	return e.resource + " not found"
}

func kksIsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var nf kksNotFoundError
	if errors.As(err, &nf) {
		return true
	}
	var apiErr *krutrim.Error
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "404") || strings.Contains(msg, "not found")
}

func kksIsTerminalFailure(status string) bool {
	switch strings.ToUpper(status) {
	case "FAILED", "ERROR", "CREATE_FAILED", "DELETE_FAILED", "UPDATE_FAILED":
		return true
	default:
		return false
	}
}

func kksIsPending(status string) bool {
	switch strings.ToUpper(status) {
	case "", "CREATING", "UPDATING", "DELETING", "PENDING", "PROVISIONING", "IN_PROGRESS":
		return true
	default:
		return false
	}
}

func waitForKKSStatus(
	ctx context.Context,
	timeout time.Duration,
	getStatus func(context.Context) (string, error),
	ready func(string) bool,
) (string, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(kksPollInterval)
	defer ticker.Stop()

	for {
		status, err := getStatus(ctx)
		if err != nil {
			return "", err
		}
		tflog.Debug(ctx, "KKS status poll", map[string]interface{}{"status": status})

		if kksIsTerminalFailure(status) {
			return status, fmt.Errorf("resource entered failed status %q", status)
		}
		if ready(status) {
			return status, nil
		}

		if time.Now().After(deadline) {
			return status, fmt.Errorf("timed out waiting for ready status (last=%q)", status)
		}

		select {
		case <-ctx.Done():
			return status, ctx.Err()
		case <-ticker.C:
		}
	}
}

func waitForKKSGone(ctx context.Context, timeout time.Duration, probe func(context.Context) error) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(kksPollInterval)
	defer ticker.Stop()

	for {
		err := probe(ctx)
		if err == nil {
			// Still exists.
		} else if kksIsNotFound(err) {
			return nil
		} else {
			return err
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for resource deletion")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
